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
