package service

import (
	"context"
	"sort"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/scoring"
)

// TeacherScoreService 是教师级/课程级评分聚合的唯一出口（S6.4 / S6.5 / S6.6）。
//
// 铁律（开发计划 §2.5.1）：教师级与课程级共用同一个 scoring.Aggregate——
// 传教师的全部场次得教师级，传某课程的场次得课程级。禁止"先算课程级再对课程取平均"，
// 否则主任端与教师端数字无法逐位一致（§8.1 一致性验收）。
type TeacherScoreService interface {
	List(ctx context.Context, role string, deptID uint64, q dto.TeacherScoreListQuery) (*dto.PageResult[dto.TeacherScoreItem], error)
	TeacherSummary(ctx context.Context, role string, deptID, userID, teacherID uint64, semester string) (*dto.TeacherEvaluationSummary, error)
	CourseSummary(ctx context.Context, role string, deptID, userID, courseID uint64, semester string) (*dto.CourseEvaluationSummary, error)
	TeacherEvaluations(ctx context.Context, role string, deptID, userID, teacherID uint64, q dto.TeacherEvaluationTimelineQuery) (*dto.PageResult[dto.TeacherEvaluationTimelineItem], error)
}

type teacherScoreService struct {
	users   repository.UserRepository
	courses repository.CourseRepository
	evals   repository.EvaluationRepository
	cfg     config.EvaluationConfig
}

// NewTeacherScoreService 构造教师评分聚合服务。
func NewTeacherScoreService(
	users repository.UserRepository,
	courses repository.CourseRepository,
	evals repository.EvaluationRepository,
	cfg config.EvaluationConfig,
) TeacherScoreService {
	return &teacherScoreService{users: users, courses: courses, evals: evals, cfg: cfg}
}

// List 返回教师评分列表（S6.4）：director 本室 / supervisor 全校或按教研室筛选。
// 默认按姓名排序（§5.3 评分伦理），无评价教师综合分为 null 且不参与排序。
func (s *teacherScoreService) List(
	ctx context.Context, role string, deptID uint64, q dto.TeacherScoreListQuery,
) (*dto.PageResult[dto.TeacherScoreItem], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	semester, err := resolveSemester(q.Semester)
	if err != nil {
		return nil, err
	}

	var scopeDept uint64
	switch role {
	case RoleDirector:
		scopeDept = deptID
	case RoleSupervisor:
		scopeDept = q.DepartmentID
	default:
		return nil, errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message())
	}

	teachers, err := s.users.ListTeachers(ctx, scopeDept)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询教师列表失败", err)
	}

	rows, err := s.evals.ListSessionDimRows(ctx, repository.SessionDimFilter{
		ScopeDepartmentID: scopeDept,
		Semester:          semester,
		FormulaVersion:    s.cfg.FormulaVersion,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询教师评分失败", err)
	}
	byTeacher := make(map[uint64][]repository.SessionDimRow, len(teachers))
	for _, row := range rows {
		byTeacher[row.TeacherID] = append(byTeacher[row.TeacherID], row)
	}

	items := make([]dto.TeacherScoreItem, 0, len(teachers))
	for _, t := range teachers {
		items = append(items, dto.TeacherScoreItem{
			TeacherID:    t.ID,
			TeacherName:  t.Name,
			JobNo:        t.JobNo,
			ScoreSummary: s.toScoreSummary(s.aggregate(byTeacher[t.ID])),
		})
	}

	start := (page - 1) * pageSize
	if start > len(items) {
		start = len(items)
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return dto.NewPageResult(items[start:end], int64(len(items)), page, pageSize), nil
}

// TeacherSummary 返回教师级评分面板（S6.5），含按课程明细。
// 权限矩阵（§5.1）：teacher 仅自己 / director 本室 / supervisor 全校，越权 40302。
func (s *teacherScoreService) TeacherSummary(
	ctx context.Context, role string, deptID, userID, teacherID uint64, semester string,
) (*dto.TeacherEvaluationSummary, error) {
	sem, err := resolveSemester(semester)
	if err != nil {
		return nil, err
	}

	teacher, err := s.checkTeacherAccess(ctx, role, deptID, userID, teacherID)
	if err != nil {
		return nil, err
	}

	rows, err := s.evals.ListSessionDimRows(ctx, repository.SessionDimFilter{
		TeacherID:      teacherID,
		Semester:       sem,
		FormulaVersion: s.cfg.FormulaVersion,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询教师评分失败", err)
	}

	byCourse := make(map[uint64][]repository.SessionDimRow)
	for _, row := range rows {
		byCourse[row.CourseID] = append(byCourse[row.CourseID], row)
	}
	courses := make([]dto.CourseScoreItem, 0, len(byCourse))
	for courseID, courseRows := range byCourse {
		course, cerr := s.courses.GetDetail(ctx, courseID)
		if cerr != nil {
			if repository.IsNotFound(cerr) {
				continue
			}
			return nil, errcode.Wrap(errcode.Internal, "查询课程失败", cerr)
		}
		courses = append(courses, dto.CourseScoreItem{
			CourseID:     courseID,
			CourseCode:   course.Code,
			CourseName:   course.Name,
			ScoreSummary: s.toScoreSummary(s.aggregate(courseRows)),
		})
	}
	sort.Slice(courses, func(i, j int) bool { return courses[i].CourseID < courses[j].CourseID })

	return &dto.TeacherEvaluationSummary{
		TeacherID:    teacherID,
		TeacherName:  teacher.Name,
		Semester:     sem,
		Courses:      courses,
		ScoreSummary: s.toScoreSummary(s.aggregate(rows)),
	}, nil
}

// CourseSummary 返回课程级评分（S6.6 教学提优页），与教师级共用聚合函数。
func (s *teacherScoreService) CourseSummary(
	ctx context.Context, role string, deptID, userID, courseID uint64, semester string,
) (*dto.CourseEvaluationSummary, error) {
	sem, err := resolveSemester(semester)
	if err != nil {
		return nil, err
	}

	course, err := s.courses.GetDetail(ctx, courseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "课程不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课程失败", err)
	}
	if err := checkCourseScope(ScopeFor(role, deptID, userID), course); err != nil {
		return nil, err
	}

	rows, err := s.evals.ListSessionDimRows(ctx, repository.SessionDimFilter{
		CourseID:       courseID,
		Semester:       sem,
		FormulaVersion: s.cfg.FormulaVersion,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询课程评分失败", err)
	}

	return &dto.CourseEvaluationSummary{
		CourseID:     course.ID,
		CourseCode:   course.Code,
		CourseName:   course.Name,
		TeacherID:    course.TeacherID,
		TeacherName:  course.TeacherName,
		Semester:     sem,
		ScoreSummary: s.toScoreSummary(s.aggregate(rows)),
	}, nil
}

// checkTeacherAccess 校验「谁能看这个教师的评分」：teacher 仅自己 / director 本室 / supervisor 全校。
// 越权返回 40302；角色不在矩阵内返回 40301；教师不存在返回 40401。
// TeacherSummary 与 TeacherEvaluations 共用，避免两处权限口径漂移。
func (s *teacherScoreService) checkTeacherAccess(
	ctx context.Context, role string, deptID, userID, teacherID uint64,
) (*model.User, error) {
	teacher, err := s.users.GetByID(ctx, teacherID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "教师不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询教师失败", err)
	}
	if teacher.Role != model.RoleTeacher {
		return nil, errcode.New(errcode.NotFound, "教师不存在")
	}
	switch role {
	case RoleTeacher:
		if teacherID != userID {
			return nil, errcode.New(errcode.ForbiddenData, "教师只能查看本人的评分")
		}
	case RoleDirector:
		if teacher.DeptID() != deptID {
			return nil, errcode.New(errcode.ForbiddenData, "无权查看其他教研室的教师")
		}
	case RoleSupervisor:
	default:
		return nil, errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message())
	}
	return teacher, nil
}

// TeacherEvaluations 返回教师「历次评价时间线」（S6.5 / T1.13）：按场次日期倒序，
// 每行含课次信息 + 督导结构化评语 + 智能体参考 + 场次综合分，一次拉取即可渲染，
// 免去前端「按课程取场次、再逐场取评语」的 N+1 编排。
// 权限与教师面板一致（teacher 仅自己 / director 本室 / supervisor 全校）。
func (s *teacherScoreService) TeacherEvaluations(
	ctx context.Context, role string, deptID, userID, teacherID uint64, q dto.TeacherEvaluationTimelineQuery,
) (*dto.PageResult[dto.TeacherEvaluationTimelineItem], error) {
	sem, err := resolveSemester(q.Semester)
	if err != nil {
		return nil, err
	}
	if _, err := s.checkTeacherAccess(ctx, role, deptID, userID, teacherID); err != nil {
		return nil, err
	}

	rows, err := s.evals.ListSessionDimRows(ctx, repository.SessionDimFilter{
		TeacherID:      teacherID,
		Semester:       sem,
		FormulaVersion: s.cfg.FormulaVersion,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询教师评价时间线失败", err)
	}

	// 只保留「有评价」的场次（时间线不展示空课次），按课次日期倒序。
	evaluated := make([]repository.SessionDimRow, 0, len(rows))
	for _, row := range rows {
		if row.HasSupervisor || row.HasAgent {
			evaluated = append(evaluated, row)
		}
	}
	sort.SliceStable(evaluated, func(i, j int) bool {
		if !evaluated[i].SessionDate.Equal(evaluated[j].SessionDate) {
			return evaluated[i].SessionDate.After(evaluated[j].SessionDate)
		}
		return evaluated[i].SessionID > evaluated[j].SessionID
	})

	page, pageSize := normalizePage(q.Page, q.PageSize)
	total := len(evaluated)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	pageRows := evaluated[start:end]

	// 只为当前页场次批量取评语，避免逐场查询。
	ids := make([]uint64, 0, len(pageRows))
	for _, row := range pageRows {
		ids = append(ids, row.SessionID)
	}
	evalRows, err := s.evals.ListBySessionIDs(ctx, ids)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询课堂评价失败", err)
	}
	bySession := make(map[uint64][]repository.EvaluationRow, len(pageRows))
	for _, e := range evalRows {
		bySession[e.SessionID] = append(bySession[e.SessionID], e)
	}

	weights := s.cfg.ScoringWeights()
	items := make([]dto.TeacherEvaluationTimelineItem, 0, len(pageRows))
	for _, row := range pageRows {
		item := dto.TeacherEvaluationTimelineItem{
			SessionID:             row.SessionID,
			SessionDate:           row.SessionDate.Format("2006-01-02"),
			Period:                row.Period,
			Topic:                 row.Topic,
			CourseID:              row.CourseID,
			CourseCode:            row.CourseCode,
			CourseName:            row.CourseName,
			Status:                row.Status,
			CompositeScore:        round2Ptr(scoring.SessionComposite(sessionScoreFromDim(row), weights, s.cfg.SupervisorWeight)),
			SupervisorEvaluations: []dto.EvaluationDTO{},
		}
		var supSum float64
		var supN int
		for _, e := range bySession[row.SessionID] {
			d := toEvaluationDTO(e)
			switch e.EvaluatorType {
			case model.EvaluatorSupervisor:
				item.SupervisorEvaluations = append(item.SupervisorEvaluations, d)
				if d.TotalScore != nil {
					supSum += *d.TotalScore
					supN++
				}
			case model.EvaluatorAgent:
				agent := d
				item.AgentEvaluation = &agent
				item.AgentScore = round2Ptr(d.TotalScore)
			}
		}
		if supN > 0 {
			avg := round2(supSum / float64(supN))
			item.SupervisorScore = &avg
		}
		items = append(items, item)
	}
	return dto.NewPageResult(items, int64(total), page, pageSize), nil
}

// sessionScoreFromDim 把一行场次维度均分映射为 scoring.SessionScore（双侧可缺）。
// 聚合与「历次评价时间线」共用，保证同一场次的分数口径完全一致。
func sessionScoreFromDim(row repository.SessionDimRow) scoring.SessionScore {
	item := scoring.SessionScore{SessionID: row.SessionID, CourseID: row.CourseID}
	if row.HasSupervisor {
		item.Supervisor = &scoring.DimensionScores{
			Objective:    row.SupObjective,
			Content:      row.SupContent,
			Interaction:  row.SupInteraction,
			Organization: row.SupOrganization,
			Frontier:     row.SupFrontier,
		}
	}
	if row.HasAgent {
		item.Agent = &scoring.DimensionScores{
			Objective:    row.AiObjective,
			Content:      row.AiContent,
			Interaction:  row.AiInteraction,
			Organization: row.AiOrganization,
			Frontier:     row.AiFrontier,
		}
	}
	return item
}

// aggregate 是全部聚合的唯一路径：仓储行 → scoring.SessionScore → Aggregate。
// 任何新增的评分视图都必须经过它，禁止另写聚合公式。
func (s *teacherScoreService) aggregate(rows []repository.SessionDimRow) scoring.Summary {
	items := make([]scoring.SessionScore, 0, len(rows))
	for _, row := range rows {
		items = append(items, sessionScoreFromDim(row))
	}
	return scoring.Aggregate(items, s.cfg.ScoringWeights(), s.cfg.SupervisorWeight)
}

// toScoreSummary 把纯函数结果映射为 DTO：分数保留两位小数，
// flags 追加 sample_insufficient（样本量规则 §2.5.5）。
func (s *teacherScoreService) toScoreSummary(sum scoring.Summary) dto.ScoreSummary {
	dims := make([]dto.DimensionScoreDTO, 0, len(sum.Dimensions))
	for _, d := range sum.Dimensions {
		dims = append(dims, dto.DimensionScoreDTO{
			Key:             d.Key,
			Name:            d.Name,
			Weight:          d.Weight,
			IsObservation:   d.IsObservation,
			Score:           round2Ptr(d.Score),
			SupervisorScore: round2Ptr(d.SupervisorScore),
			AgentScore:      round2Ptr(d.AgentScore),
		})
	}

	flags := sum.Flags
	if sum.Sample.EvaluatedCount < s.cfg.MinSampleSize {
		flags = append(flags, "sample_insufficient")
	}
	if flags == nil {
		flags = []string{}
	}

	return dto.ScoreSummary{
		CompositeScore:  round2Ptr(sum.Composite),
		SupervisorScore: round2Ptr(sum.Supervisor),
		AgentScore:      round2Ptr(sum.Agent),
		Dimensions:      dims,
		Sample: dto.SampleDTO{
			SessionCount:     sum.Sample.SessionCount,
			EvaluatedCount:   sum.Sample.EvaluatedCount,
			SupervisorCount:  sum.Sample.SupervisorCount,
			AgentCount:       sum.Sample.AgentCount,
			AlignedCount:     sum.Sample.AlignedCount,
			SampleSufficient: sum.Sample.EvaluatedCount >= s.cfg.MinSampleSize,
		},
		Flags:          flags,
		Weights:        dto.ScoreWeights{Supervisor: s.cfg.SupervisorWeight, Agent: s.cfg.AgentWeight},
		FormulaVersion: s.cfg.FormulaVersion,
	}
}

// resolveSemester 解析学期参数：缺省取当前学期，未知学期返回 40001（§2.5.6）。
func resolveSemester(q string) (string, error) {
	if q == "" {
		return CurrentSemester, nil
	}
	for _, sem := range Semesters {
		if sem == q {
			return q, nil
		}
	}
	return "", errcode.New(errcode.Params, "semester 参数非法")
}
