package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/jwtutil"
)

// TestScopeFor 是数据范围裁剪的核心单测：三角色各一条用例。
// 铁律（后端 AGENTS.md §3）：裁剪只能发生在后端。
func TestScopeFor(t *testing.T) {
	const (
		deptID = uint64(7)
		userID = uint64(42)
	)

	cases := []struct {
		name        string
		role        string
		wantDept    uint64
		wantTeacher uint64
	}{
		{name: "主任裁剪到本教研室", role: RoleDirector, wantDept: deptID, wantTeacher: 0},
		{name: "教师裁剪到本人", role: RoleTeacher, wantDept: 0, wantTeacher: userID},
		{name: "督导不受限", role: RoleSupervisor, wantDept: 0, wantTeacher: 0},
		{name: "未知角色按不受限兜底后再由路由守卫拦截", role: "unknown", wantDept: 0, wantTeacher: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScopeFor(tc.role, deptID, userID)
			assert.Equal(t, tc.wantDept, got.DepartmentID)
			assert.Equal(t, tc.wantTeacher, got.TeacherID)
		})
	}
}

// TestCourseListAppliesScope 验证列表查询把角色范围拼进过滤条件。
func TestCourseListAppliesScope(t *testing.T) {
	cases := []struct {
		name        string
		role        string
		wantDept    uint64
		wantTeacher uint64
	}{
		{name: "主任", role: RoleDirector, wantDept: 3, wantTeacher: 0},
		{name: "教师", role: RoleTeacher, wantDept: 0, wantTeacher: 9},
		{name: "督导", role: RoleSupervisor, wantDept: 0, wantTeacher: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeCourseRepo{}
			svc := NewCourseService(repo, &fakeUserRepo{})

			_, err := svc.List(context.Background(), tc.role, 3, 9, dto.CourseListQuery{})
			require.NoError(t, err)
			require.NotNil(t, repo.lastListParams)
			assert.Equal(t, tc.wantDept, repo.lastListParams.ScopeDepartmentID)
			assert.Equal(t, tc.wantTeacher, repo.lastListParams.ScopeTeacherID)
			// 分页参数归一化：默认 page=1 / pageSize=10
			assert.Equal(t, 1, repo.lastListParams.Page)
			assert.Equal(t, 10, repo.lastListParams.PageSize)
		})
	}
}

// TestCheckCourseScope 覆盖详情读取的越权矩阵。
func TestCheckCourseScope(t *testing.T) {
	course := &repository.CourseRow{ID: 1, DepartmentID: 1, TeacherID: 2}

	cases := []struct {
		name    string
		scope   Scope
		wantErr bool
	}{
		{name: "主任看本室课程", scope: Scope{DepartmentID: 1}, wantErr: false},
		{name: "主任看他室课程", scope: Scope{DepartmentID: 2}, wantErr: true},
		{name: "教师看本人课程", scope: Scope{TeacherID: 2}, wantErr: false},
		{name: "教师看他人课程", scope: Scope{TeacherID: 3}, wantErr: true},
		{name: "督导看任意课程", scope: Scope{}, wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkCourseScope(tc.scope, course)
			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestNormalizePage 覆盖分页边界。
func TestNormalizePage(t *testing.T) {
	cases := []struct {
		page, pageSize         int
		wantPage, wantPageSize int
	}{
		{page: 0, pageSize: 0, wantPage: 1, wantPageSize: 10},
		{page: -3, pageSize: -1, wantPage: 1, wantPageSize: 10},
		{page: 2, pageSize: 20, wantPage: 2, wantPageSize: 20},
		{page: 1, pageSize: 999, wantPage: 1, wantPageSize: 50},
	}

	for _, tc := range cases {
		gotPage, gotSize := normalizePage(tc.page, tc.pageSize)
		assert.Equal(t, tc.wantPage, gotPage)
		assert.Equal(t, tc.wantPageSize, gotSize)
	}
}

// TestSupervisionCoverage 对照 MySQL 文档 §6 期望值：分母 6、分子 3、总体 0.5，SE 1/3→0.33。
func TestSupervisionCoverage(t *testing.T) {
	repo := &fakeSupervisionRepo{
		coverage: func(context.Context, string) (int64, int64, error) { return 6, 3, nil },
		byDept: func(context.Context, string) ([]repository.DeptCoverageRow, error) {
			return []repository.DeptCoverageRow{
				{Department: "软件工程教研室", Total: 3, Supervised: 1},
				{Department: "计算机系统教研室", Total: 2, Supervised: 1},
				{Department: "人工智能教研室", Total: 1, Supervised: 1},
			}, nil
		},
	}

	cov, err := NewSupervisionService(repo).Coverage(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 6, cov.TotalCourses)
	assert.Equal(t, 3, cov.SupervisedCourses)
	assert.InDelta(t, 0.5, cov.Rate, 1e-9)
	require.Len(t, cov.ByDepartment, 3)
	assert.InDelta(t, 0.33, cov.ByDepartment[0].Rate, 1e-9)
	assert.InDelta(t, 0.5, cov.ByDepartment[1].Rate, 1e-9)
	assert.InDelta(t, 1.0, cov.ByDepartment[2].Rate, 1e-9)
}

// TestSupervisionCoverageZeroDenominator 覆盖无开课课程时不应除零。
func TestSupervisionCoverageZeroDenominator(t *testing.T) {
	cov, err := NewSupervisionService(&fakeSupervisionRepo{}).Coverage(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0.0, cov.Rate)
	assert.Empty(t, cov.ByDepartment)
}

// TestSupervisionPlansDateValidation 校验 dateFrom / dateTo 格式。
func TestSupervisionPlansDateValidation(t *testing.T) {
	svc := NewSupervisionService(&fakeSupervisionRepo{})

	_, err := svc.Plans(context.Background(), dto.PlanListQuery{DateFrom: "2026/09/01"})
	require.Error(t, err)
	assert.Equal(t, errcode.Params, errcode.From(err).Code)

	_, err = svc.Plans(context.Background(), dto.PlanListQuery{DateFrom: "2026-09-01", DateTo: "2026-09-30"})
	assert.NoError(t, err)
}

// TestResourceDeleteOwnership 覆盖资源删除的归属校验（他人课程 → 40302）。
func TestResourceDeleteOwnership(t *testing.T) {
	otherTeacherCourse := &repository.CourseRow{ID: 1, TeacherID: 2}

	repo := &fakeResourceRepo{
		getByID: func(context.Context, uint64) (*model.Resource, error) {
			return &model.Resource{ID: 10, CourseID: 1, FilePath: "uploads/1/x.pdf"}, nil
		},
	}
	courseRepo := &fakeCourseRepo{
		getDetail: func(context.Context, uint64) (*repository.CourseRow, error) {
			return otherTeacherCourse, nil
		},
	}
	svc := NewResourceService(repo, courseRepo, config.UploadConfig{})

	err := svc.Delete(context.Background(), 3, 10)
	require.Error(t, err)
	assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
	assert.False(t, repo.deleteInvoked, "越权时不得触碰数据库")
}

// TestResourceDeleteNotFound 覆盖资源不存在 → 40401。
func TestResourceDeleteNotFound(t *testing.T) {
	repo := &fakeResourceRepo{
		getByID: func(context.Context, uint64) (*model.Resource, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	svc := NewResourceService(repo, &fakeCourseRepo{}, config.UploadConfig{})

	err := svc.Delete(context.Background(), 2, 999)
	require.Error(t, err)
	assert.Equal(t, errcode.NotFound, errcode.From(err).Code)
}

// TestCourseCreateForbiddenDepartment 覆盖主任跨教研室建课 → 40302。
func TestCourseCreateForbiddenDepartment(t *testing.T) {
	courseRepo := &fakeCourseRepo{}
	svc := NewCourseService(courseRepo, &fakeUserRepo{})

	_, err := svc.Create(context.Background(), 1, dto.CourseUpsertReq{
		Code: "SE3101", Name: "越权课程", DepartmentID: 2, TeacherID: 4,
		Semester: "2026-2027-1", Credit: 3, Hours: 48, Status: model.CourseStatusOpen,
	})
	require.Error(t, err)
	assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
	assert.False(t, courseRepo.lastExistsChecked, "越权时不应进入后续校验")
}

// TestCourseCreateConflict 覆盖 code+semester 冲突 → 40901。
func TestCourseCreateConflict(t *testing.T) {
	courseRepo := &fakeCourseRepo{
		existsCodeSem: func(context.Context, string, string, uint64) (bool, error) { return true, nil },
	}
	userRepo := &fakeUserRepo{
		getByID: func(context.Context, uint64) (*model.User, error) {
			dept := uint64(1)
			return &model.User{ID: 2, Role: model.RoleTeacher, DepartmentID: &dept}, nil
		},
	}
	svc := NewCourseService(courseRepo, userRepo)

	_, err := svc.Create(context.Background(), 1, dto.CourseUpsertReq{
		Code: "SE3101", Name: "重复课程", DepartmentID: 1, TeacherID: 2,
		Semester: "2026-2027-1", Credit: 3, Hours: 48, Status: model.CourseStatusOpen,
	})
	require.Error(t, err)
	assert.Equal(t, errcode.Conflict, errcode.From(err).Code)
}

// TestCourseCreateTeacherNotInDepartment 覆盖教师不属于所选教研室 → 40302。
func TestCourseCreateTeacherNotInDepartment(t *testing.T) {
	userRepo := &fakeUserRepo{
		getByID: func(context.Context, uint64) (*model.User, error) {
			dept := uint64(2)
			return &model.User{ID: 4, Role: model.RoleTeacher, DepartmentID: &dept}, nil
		},
	}
	svc := NewCourseService(&fakeCourseRepo{}, userRepo)

	_, err := svc.Create(context.Background(), 1, dto.CourseUpsertReq{
		Code: "SE3101", Name: "错配教师", DepartmentID: 1, TeacherID: 4,
		Semester: "2026-2027-1", Credit: 3, Hours: 48, Status: model.CourseStatusOpen,
	})
	require.Error(t, err)
	assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
}

// TestAuthLogin 覆盖登录成功、密码错误、账号不存在三种情形（错误一律 40101，防枚举）。
func TestAuthLogin(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	require.NoError(t, err)

	dept := uint64(1)
	user := &model.User{
		ID: 1, Username: "director", PasswordHash: string(hash),
		Name: "王建国", Role: model.RoleDirector, DepartmentID: &dept, JobNo: "T2001", Status: 1,
	}

	users := &fakeUserRepo{
		getByUsername: func(_ context.Context, username string) (*model.User, error) {
			if username == "director" {
				return user, nil
			}
			return nil, gorm.ErrRecordNotFound
		},
		deptName: func(context.Context, uint64) (string, error) { return "软件工程教研室", nil },
	}

	svc := NewAuthService(users, jwtutil.New("test-secret", time.Hour))

	t.Run("登录成功返回 token 与用户", func(t *testing.T) {
		resp, err := svc.Login(context.Background(), dto.LoginReq{Username: "director", Password: "123456"})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.Equal(t, "director", resp.User.Role)
		assert.Equal(t, "软件工程教研室", resp.User.Department)
	})

	t.Run("密码错误返回 40101", func(t *testing.T) {
		_, err := svc.Login(context.Background(), dto.LoginReq{Username: "director", Password: "wrong"})
		assert.Equal(t, errcode.Unauthorized, errcode.From(err).Code)
	})

	t.Run("账号不存在返回 40101", func(t *testing.T) {
		_, err := svc.Login(context.Background(), dto.LoginReq{Username: "ghost", Password: "123456"})
		assert.Equal(t, errcode.Unauthorized, errcode.From(err).Code)
	})
}

// TestToResourceDTOs 校验 uploadedAt 输出 ISO 8601。
func TestToResourceDTOs(t *testing.T) {
	ts := time.Date(2026, 9, 2, 9, 30, 0, 0, time.Local)
	out := toResourceDTOs([]repository.ResourceRow{{ID: 1, Name: "a.pdf", UploadedAt: ts}})
	require.Len(t, out, 1)
	assert.Equal(t, ts.Format(time.RFC3339), out[0].UploadedAt)
	assert.Equal(t, "a.pdf", out[0].Name)
}

// TestMapResourceType 覆盖扩展名 → 资源类型映射。
func TestMapResourceType(t *testing.T) {
	cases := map[string]string{
		".pdf":  model.ResourceTypePDF,
		".doc":  model.ResourceTypeDoc,
		".docx": model.ResourceTypeDoc,
		".ppt":  model.ResourceTypePPT,
		".pptx": model.ResourceTypePPT,
		".mp4":  model.ResourceTypeVideo,
		".zip":  model.ResourceTypeZip,
		".xyz":  model.ResourceTypeOther,
	}
	for ext, want := range cases {
		assert.Equal(t, want, mapResourceType(ext), "ext=%s", ext)
	}
}

// evalCfg 构造与开发计划 §2.2 一致的计分口径（frontier 为观测项，权重 0）。
func evalCfg() config.EvaluationConfig {
	return config.EvaluationConfig{
		Weights:          config.WeightConfig{Objective: 0.30, Content: 0.30, Interaction: 0.20, Organization: 0.20},
		SupervisorWeight: 0.5,
		AgentWeight:      0.5,
		FormulaVersion:   "v1",
		MinSampleSize:    3,
	}
}

// sessionRow 构造一场已存在的授课记录（课程 1 / 教师 2 / 软件工程教研室）。
func sessionRow() *repository.SessionDetailRow {
	return &repository.SessionDetailRow{
		ID: 7, CourseID: 1, TeacherID: 2, DepartmentID: 1,
		CourseCode: "SE3101", CourseName: "软件项目管理", Semester: "2026-2027-1",
		TeacherName: "李明", Status: "scheduled",
	}
}

// TestCheckSessionScope 覆盖授课记录读取的越权矩阵（T1.5 验收：越权 40302）。
func TestCheckSessionScope(t *testing.T) {
	row := sessionRow()

	cases := []struct {
		name    string
		scope   Scope
		wantErr bool
	}{
		{name: "主任看本室场次", scope: Scope{DepartmentID: 1}, wantErr: false},
		{name: "主任看他室场次", scope: Scope{DepartmentID: 2}, wantErr: true},
		{name: "教师看本人场次", scope: Scope{TeacherID: 2}, wantErr: false},
		{name: "教师看他人场次", scope: Scope{TeacherID: 3}, wantErr: true},
		{name: "督导看任意场次", scope: Scope{}, wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkSessionScope(tc.scope, row)
			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestResolveSemester 覆盖学期参数解析（§2.5.6：缺省当前学期，未知学期 40001）。
func TestResolveSemester(t *testing.T) {
	sem, err := resolveSemester("")
	require.NoError(t, err)
	assert.Equal(t, CurrentSemester, sem)

	sem, err = resolveSemester("2025-2026-2")
	require.NoError(t, err)
	assert.Equal(t, "2025-2026-2", sem)

	_, err = resolveSemester("2099-2100-1")
	require.Error(t, err)
	assert.Equal(t, errcode.Params, errcode.From(err).Code)
}

// TestSessionCreateRules 覆盖督导创建授课记录的业务规则（S6.1）。
func TestSessionCreateRules(t *testing.T) {
	newSvc := func(sessions *fakeSessionRepo) SessionService {
		courses := &fakeCourseRepo{
			getDetail: func(ctx context.Context, id uint64) (*repository.CourseRow, error) {
				return &repository.CourseRow{ID: 1, Code: "SE3101", Name: "软件项目管理", TeacherID: 2, Semester: "2026-2027-1"}, nil
			},
		}
		return NewSessionService(sessions, &fakeEvalRepo{}, courses, &fakeUserRepo{}, evalCfg())
	}

	t.Run("日期晚于今天返回 40002", func(t *testing.T) {
		svc := newSvc(&fakeSessionRepo{})
		_, err := svc.Create(context.Background(), dto.SessionCreateReq{CourseID: 1, SessionDate: "2099-01-01", Period: "1-2 节"})
		require.Error(t, err)
		assert.Equal(t, errcode.BizRule, errcode.From(err).Code)
	})

	t.Run("同课程同日同节次重复返回 40901", func(t *testing.T) {
		svc := newSvc(&fakeSessionRepo{
			existsDuplicate: func(ctx context.Context, courseID, classID uint64, date time.Time, period string, excludeID uint64) (bool, error) {
				return true, nil
			},
		})
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		_, err := svc.Create(context.Background(), dto.SessionCreateReq{CourseID: 1, SessionDate: yesterday, Period: "3-4 节"})
		require.Error(t, err)
		assert.Equal(t, errcode.Conflict, errcode.From(err).Code)
	})

	t.Run("成功创建：teacher_id 随课程落库且初始状态 scheduled", func(t *testing.T) {
		var created *model.TeachingSession
		sessions := &fakeSessionRepo{
			create: func(ctx context.Context, s *model.TeachingSession) error {
				s.ID = 9
				created = s
				return nil
			},
			getDetail: func(ctx context.Context, id uint64) (*repository.SessionDetailRow, error) {
				return &repository.SessionDetailRow{
					ID: 9, CourseID: 1, TeacherID: 2, Status: "scheduled",
					CourseName: "软件项目管理", TeacherName: "李明", Semester: "2026-2027-1",
				}, nil
			},
		}
		svc := newSvc(sessions)
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		detail, err := svc.Create(context.Background(), dto.SessionCreateReq{CourseID: 1, SessionDate: yesterday, Period: "验收节次", Topic: "T"})
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, uint64(2), created.TeacherID)
		assert.Equal(t, model.SessionStatusScheduled, created.Status)
		assert.Equal(t, "scheduled", detail.Status)
		assert.Equal(t, "李明", detail.TeacherName)
	})
}

// TestSubmitSupervisorEvaluationRules 覆盖评分提交的业务规则（S6.3）。
func TestSubmitSupervisorEvaluationRules(t *testing.T) {
	svc := NewSessionService(&fakeSessionRepo{
		getDetail: func(ctx context.Context, id uint64) (*repository.SessionDetailRow, error) {
			return sessionRow(), nil
		},
	}, &fakeEvalRepo{}, &fakeCourseRepo{}, &fakeUserRepo{}, evalCfg())

	_, err := svc.SubmitSupervisorEvaluation(context.Background(), 5, 7, dto.SupervisorEvaluationReq{Comment: "无维度"})
	require.Error(t, err)
	assert.Equal(t, errcode.BizRule, errcode.From(err).Code)
}

// TestSubmitSupervisorEvaluationRequiresAllDims 覆盖 F4：督导五个维度必须全部录入，
// 任一缺失返回 40002（开发计划 §2.1 / §4 错误码表）。
func TestSubmitSupervisorEvaluationRequiresAllDims(t *testing.T) {
	svc := NewSessionService(&fakeSessionRepo{
		getDetail: func(ctx context.Context, id uint64) (*repository.SessionDetailRow, error) {
			return sessionRow(), nil
		},
	}, &fakeEvalRepo{}, &fakeCourseRepo{}, &fakeUserRepo{}, evalCfg())

	i := func(v int) *int { return &v }
	cases := []struct {
		name string
		req  dto.SupervisorEvaluationReq
	}{
		{name: "缺 interaction/organization/frontier", req: dto.SupervisorEvaluationReq{Objective: i(4), Content: i(3)}},
		{name: "只缺 frontier", req: dto.SupervisorEvaluationReq{Objective: i(4), Content: i(3), Interaction: i(2), Organization: i(3)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.SubmitSupervisorEvaluation(context.Background(), 5, 7, tc.req)
			require.Error(t, err)
			assert.Equal(t, errcode.BizRule, errcode.From(err).Code)
		})
	}
}

// TestTeacherSummarySampleUsesEvaluated 覆盖 F5：样本充足性以「已评价场次」为准，
// 未评价场次不得把样本量撑到达标（§2.5.5）。
func TestTeacherSummarySampleUsesEvaluated(t *testing.T) {
	dept := uint64(1)
	users := &fakeUserRepo{getByID: func(_ context.Context, id uint64) (*model.User, error) {
		return &model.User{ID: id, Name: "李明", Role: model.RoleTeacher, DepartmentID: &dept}, nil
	}}
	f := func(v float64) *float64 { return &v }
	rows := []repository.SessionDimRow{
		{SessionID: 1, TeacherID: 2, CourseID: 1, HasSupervisor: true,
			SupObjective: f(4), SupContent: f(4), SupInteraction: f(4), SupOrganization: f(4), SupFrontier: f(4)},
		{SessionID: 2, TeacherID: 2, CourseID: 1},
		{SessionID: 3, TeacherID: 2, CourseID: 1},
	}
	evals := &fakeEvalRepo{listSessionDimRows: func(
		context.Context, repository.SessionDimFilter,
	) ([]repository.SessionDimRow, error) {
		return rows, nil
	}}
	courses := &fakeCourseRepo{getDetail: func(_ context.Context, id uint64) (*repository.CourseRow, error) {
		return &repository.CourseRow{ID: id, Code: "SE3101", Name: "软件项目管理", TeacherID: 2, DepartmentID: 1}, nil
	}}
	svc := NewTeacherScoreService(users, courses, evals, evalCfg())

	sum, err := svc.TeacherSummary(context.Background(), RoleDirector, 1, 1, 2, "")
	require.NoError(t, err)
	assert.Equal(t, 3, sum.Sample.SessionCount, "总会话数=3")
	assert.Equal(t, 1, sum.Sample.EvaluatedCount, "已评价场次=1")
	assert.False(t, sum.Sample.SampleSufficient, "已评价场次 1<3，必须标记样本不足")
	assert.Contains(t, sum.Flags, "sample_insufficient")
	require.NotNil(t, sum.CompositeScore)
	assert.Equal(t, 75.00, *sum.CompositeScore)
}

// TestSubmitSupervisorEvaluationIdempotent 验证 T1.6 验收：重复提交整行覆盖而非报错。
func TestSubmitSupervisorEvaluationIdempotent(t *testing.T) {
	stored := map[uint64]*model.Evaluation{}
	sessions := &fakeSessionRepo{
		getDetail: func(ctx context.Context, id uint64) (*repository.SessionDetailRow, error) {
			return sessionRow(), nil
		},
	}
	evals := &fakeEvalRepo{
		upsert: func(ctx context.Context, e *model.Evaluation) error {
			stored[e.EvaluatorID] = e
			return nil
		},
		listBySession: func(ctx context.Context, sessionID uint64) ([]repository.EvaluationRow, error) {
			rows := make([]repository.EvaluationRow, 0, len(stored))
			for _, e := range stored {
				rows = append(rows, repository.EvaluationRow{Evaluation: *e, EvaluatorName: "陈静"})
			}
			return rows, nil
		},
	}
	svc := NewSessionService(sessions, evals, &fakeCourseRepo{}, &fakeUserRepo{}, evalCfg())

	i := func(v int) *int { return &v }
	first, err := svc.SubmitSupervisorEvaluation(context.Background(), 5, 7, dto.SupervisorEvaluationReq{
		Objective: i(4), Content: i(3), Interaction: i(2), Organization: i(3), Frontier: i(2),
	})
	require.NoError(t, err)
	require.NotNil(t, first.TotalScore)
	assert.Equal(t, 52.5, *first.TotalScore)
	assert.Equal(t, "v1", first.FormulaVersion)
	assert.Equal(t, "evaluated", sessions.lastStatus)

	second, err := svc.SubmitSupervisorEvaluation(context.Background(), 5, 7, dto.SupervisorEvaluationReq{
		Objective: i(5), Content: i(4), Interaction: i(4), Organization: i(4), Frontier: i(3),
	})
	require.NoError(t, err)
	require.NotNil(t, second.TotalScore)
	assert.Equal(t, 82.5, *second.TotalScore)
	assert.Len(t, stored, 1)
	assert.Equal(t, "陈静", second.EvaluatorName)
}

// teacherDimRow 构造一行场次维度均分（两侧均为 4 分，便于手算 75.00）。
func teacherDimRow(sessionID, courseID uint64, date string, hasSup, hasAgent bool) repository.SessionDimRow {
	d, _ := time.Parse("2006-01-02", date)
	f := func(v float64) *float64 { return &v }
	row := repository.SessionDimRow{
		SessionID: sessionID, TeacherID: 2, CourseID: courseID,
		SessionDate: d, Period: "3-4 节", Topic: "验收主题", Status: "evaluated",
		CourseCode: "SE3101", CourseName: "软件项目管理",
		HasSupervisor: hasSup, HasAgent: hasAgent,
	}
	if hasSup {
		row.SupObjective, row.SupContent = f(4), f(4)
		row.SupInteraction, row.SupOrganization, row.SupFrontier = f(4), f(4), f(4)
	}
	if hasAgent {
		row.AiContent, row.AiInteraction = f(4), f(4)
		row.AiOrganization, row.AiFrontier = f(4), f(4)
	}
	return row
}

// TestTeacherEvaluationsTimeline 覆盖 T1.13：历次评价时间线按课次倒序、
// 只含有评价的场次，并带督导评语与智能体参考；同时覆盖分页与越权。
func TestTeacherEvaluationsTimeline(t *testing.T) {
	dept := uint64(1)
	users := &fakeUserRepo{getByID: func(_ context.Context, id uint64) (*model.User, error) {
		return &model.User{ID: id, Name: "李明", Role: model.RoleTeacher, DepartmentID: &dept}, nil
	}}
	f := func(v float64) *float64 { return &v }
	rows := []repository.SessionDimRow{
		teacherDimRow(1, 1, "2026-09-12", true, false),  // 仅督导
		teacherDimRow(2, 1, "2026-09-19", true, true),   // 双侧
		teacherDimRow(3, 1, "2026-09-26", false, false), // 未评价 → 不得进入时间线
	}
	evals := []repository.EvaluationRow{
		{Evaluation: model.Evaluation{SessionID: 2, EvaluatorType: model.EvaluatorSupervisor, EvaluatorID: 5,
			TotalScore: f(70), Comment: "讲解透彻", Highlights: "讨论有效"}, EvaluatorName: "陈静"},
		{Evaluation: model.Evaluation{SessionID: 2, EvaluatorType: model.EvaluatorAgent, EvaluatorID: 0,
			TotalScore: f(67.86), AIModelVersion: "qwen-audio-v1"}},
		{Evaluation: model.Evaluation{SessionID: 1, EvaluatorType: model.EvaluatorSupervisor, EvaluatorID: 5,
			TotalScore: f(52.5), Comment: "互动偏少"}, EvaluatorName: "陈静"},
	}
	repo := &fakeEvalRepo{
		listSessionDimRows: func(
			context.Context, repository.SessionDimFilter,
		) ([]repository.SessionDimRow, error) {
			return rows, nil
		},
		listBySessionIDs: func(_ context.Context, ids []uint64) ([]repository.EvaluationRow, error) {
			out := make([]repository.EvaluationRow, 0, len(evals))
			for _, e := range evals {
				for _, id := range ids {
					if e.SessionID == id {
						out = append(out, e)
						break
					}
				}
			}
			return out, nil
		},
	}
	svc := NewTeacherScoreService(users, &fakeCourseRepo{}, repo, evalCfg())

	res, err := svc.TeacherEvaluations(context.Background(), RoleDirector, 1, 1, 2, dto.TeacherEvaluationTimelineQuery{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total, "未评价场次不计入时间线")
	require.Len(t, res.List, 2)
	assert.Equal(t, uint64(2), res.List[0].SessionID, "必须按课次日期倒序")
	assert.Equal(t, uint64(1), res.List[1].SessionID)
	assert.Equal(t, "2026-09-19", res.List[0].SessionDate)
	assert.Equal(t, "软件项目管理", res.List[0].CourseName)

	// 场次 2：双侧全 4 分 → 融合后 75.00；督导/智能体单侧分取落库总分。
	require.NotNil(t, res.List[0].CompositeScore)
	assert.Equal(t, 75.00, *res.List[0].CompositeScore)
	require.NotNil(t, res.List[0].SupervisorScore)
	assert.Equal(t, 70.0, *res.List[0].SupervisorScore)
	require.NotNil(t, res.List[0].AgentScore)
	assert.Equal(t, 67.86, *res.List[0].AgentScore)
	require.Len(t, res.List[0].SupervisorEvaluations, 1)
	assert.Equal(t, "讲解透彻", res.List[0].SupervisorEvaluations[0].Comment)
	assert.Equal(t, "讨论有效", res.List[0].SupervisorEvaluations[0].Highlights)
	require.NotNil(t, res.List[0].AgentEvaluation)
	assert.Equal(t, "qwen-audio-v1", res.List[0].AgentEvaluation.AIModelVersion)

	// 场次 1：仅督导，智能体侧为空。
	require.NotNil(t, res.List[1].CompositeScore)
	assert.Equal(t, 75.00, *res.List[1].CompositeScore)
	assert.Equal(t, 52.5, *res.List[1].SupervisorScore)
	assert.Nil(t, res.List[1].AgentEvaluation)
	assert.Empty(t, res.List[1].SupervisorEvaluations[0].Highlights)

	t.Run("分页", func(t *testing.T) {
		p, perr := svc.TeacherEvaluations(context.Background(), RoleDirector, 1, 1, 2,
			dto.TeacherEvaluationTimelineQuery{Page: 1, PageSize: 1})
		require.NoError(t, perr)
		assert.Equal(t, int64(2), p.Total)
		require.Len(t, p.List, 1)
		assert.Equal(t, uint64(2), p.List[0].SessionID)
	})

	t.Run("教师查他人时间线返回 40302", func(t *testing.T) {
		_, aerr := svc.TeacherEvaluations(context.Background(), RoleTeacher, 1, 2, 3, dto.TeacherEvaluationTimelineQuery{})
		require.Error(t, aerr)
		assert.Equal(t, errcode.ForbiddenData, errcode.From(aerr).Code)
	})
}
