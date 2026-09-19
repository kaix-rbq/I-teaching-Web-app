package service

import (
	"context"
	"time"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/scoring"
)

// SessionService 负责授课记录与督导评分的业务规则（S6.1 / S6.3 / S6.2）。
type SessionService interface {
	Create(ctx context.Context, req dto.SessionCreateReq) (*dto.SessionDetail, error)
	ListByCourse(ctx context.Context, role string, deptID, userID, courseID uint64, q dto.SessionListQuery) (*dto.PageResult[dto.SessionListItem], error)
	Detail(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.SessionDetail, error)
	Evaluation(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.SessionEvaluation, error)
	SubmitSupervisorEvaluation(ctx context.Context, supervisorID, sessionID uint64, req dto.SupervisorEvaluationReq) (*dto.EvaluationDTO, error)
}

type sessionService struct {
	sessions repository.SessionRepository
	evals    repository.EvaluationRepository
	courses  repository.CourseRepository
	users    repository.UserRepository
	cfg      config.EvaluationConfig
}

// NewSessionService 构造授课记录服务。
func NewSessionService(
	sessions repository.SessionRepository,
	evals repository.EvaluationRepository,
	courses repository.CourseRepository,
	users repository.UserRepository,
	cfg config.EvaluationConfig,
) SessionService {
	return &sessionService{sessions: sessions, evals: evals, courses: courses, users: users, cfg: cfg}
}

// Create 由督导创建授课记录：评价的落点（S6.1）。
// 业务规则：日期不得晚于今天（40002）；课程必须存在（40401）；同课程同班级同日同节次唯一（40901）。
func (s *sessionService) Create(ctx context.Context, req dto.SessionCreateReq) (*dto.SessionDetail, error) {
	date, err := time.Parse("2006-01-02", req.SessionDate)
	if err != nil {
		return nil, errcode.New(errcode.Params, "sessionDate 需为 YYYY-MM-DD 格式")
	}
	if date.Format("2006-01-02") > time.Now().Format("2006-01-02") {
		return nil, errcode.New(errcode.BizRule, "授课记录日期不能晚于今天")
	}

	course, err := s.courses.GetDetail(ctx, req.CourseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "课程不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课程失败", err)
	}

	if req.PlanID != nil && *req.PlanID > 0 {
		exists, perr := s.sessions.PlanExists(ctx, *req.PlanID)
		if perr != nil {
			return nil, errcode.Wrap(errcode.Internal, "校验听评课计划失败", perr)
		}
		if !exists {
			return nil, errcode.New(errcode.NotFound, "来源听评课计划不存在")
		}
	}

	duplicate, err := s.sessions.ExistsDuplicate(ctx, req.CourseID, req.ClassID, date, req.Period, 0)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "校验授课记录唯一性失败", err)
	}
	if duplicate {
		return nil, errcode.New(errcode.Conflict, "该课程同日同节次已存在授课记录")
	}

	session := &model.TeachingSession{
		CourseID:    req.CourseID,
		ClassID:     req.ClassID,
		TeacherID:   course.TeacherID,
		SessionDate: date,
		Period:      req.Period,
		Topic:       req.Topic,
		PlanID:      req.PlanID,
		Status:      model.SessionStatusScheduled,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "创建授课记录失败", err)
	}
	return s.Detail(ctx, RoleSupervisor, 0, 0, session.ID)
}

// ListByCourse 返回课程历史授课记录（S6.2），数据范围按角色裁剪（§5.1）。
func (s *sessionService) ListByCourse(
	ctx context.Context, role string, deptID, userID, courseID uint64, q dto.SessionListQuery,
) (*dto.PageResult[dto.SessionListItem], error) {
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

	// 学期切片（§2.5.6）：授课记录挂在课程下，学期不匹配时返回空列表。
	if q.Semester != "" && q.Semester != course.Semester {
		return dto.NewPageResult[dto.SessionListItem](nil, 0, 1, 10), nil
	}

	page, pageSize := normalizePage(q.Page, q.PageSize)
	rows, total, err := s.sessions.ListByCourse(ctx, repository.SessionListParams{
		CourseID: courseID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}
	return dto.NewPageResult(toSessionListItems(rows), total, page, pageSize), nil
}

// Detail 返回单场授课记录基本信息。
func (s *sessionService) Detail(
	ctx context.Context, role string, deptID, userID, sessionID uint64,
) (*dto.SessionDetail, error) {
	row, err := s.sessions.GetDetail(ctx, sessionID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "授课记录不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}
	if err := checkSessionScope(ScopeFor(role, deptID, userID), row); err != nil {
		return nil, err
	}
	detail := toSessionDetail(*row)
	return &detail, nil
}

// Evaluation 返回当堂课评估页聚合数据（§4.1）：场次 + 督导评分 + 智能体参考 + 评语。
func (s *sessionService) Evaluation(
	ctx context.Context, role string, deptID, userID, sessionID uint64,
) (*dto.SessionEvaluation, error) {
	row, err := s.sessions.GetDetail(ctx, sessionID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "授课记录不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}
	if err := checkSessionScope(ScopeFor(role, deptID, userID), row); err != nil {
		return nil, err
	}

	rows, err := s.evals.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询课堂评价失败", err)
	}

	out := dto.SessionEvaluation{
		Session:          toSessionDetail(*row),
		SupervisorScores: []dto.EvaluationDTO{},
	}
	for _, evalRow := range rows {
		switch evalRow.EvaluatorType {
		case model.EvaluatorSupervisor:
			out.SupervisorScores = append(out.SupervisorScores, toEvaluationDTO(evalRow))
		case model.EvaluatorAgent:
			item := toEvaluationDTO(evalRow)
			out.AgentScore = &item
		}
	}
	return &out, nil
}

// SubmitSupervisorEvaluation 提交/覆盖督导评分（S6.3，PUT 幂等，写入即生效）。
// 单次 total_score 落库时同时写 formula_version（§2.5.6 口径版本例外约定）。
func (s *sessionService) SubmitSupervisorEvaluation(
	ctx context.Context, supervisorID, sessionID uint64, req dto.SupervisorEvaluationReq,
) (*dto.EvaluationDTO, error) {
	if _, err := s.sessions.GetDetail(ctx, sessionID); err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "授课记录不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}

	dims := scoring.NewDimensionScores(req.Objective, req.Content, req.Interaction, req.Organization, req.Frontier)
	if req.Objective == nil && req.Content == nil && req.Interaction == nil &&
		req.Organization == nil && req.Frontier == nil {
		return nil, errcode.New(errcode.BizRule, "至少需要提交一个维度的评分")
	}

	total := scoring.SideTotal(dims, s.cfg.ScoringWeights())
	record := &model.Evaluation{
		SessionID:         sessionID,
		EvaluatorType:     model.EvaluatorSupervisor,
		EvaluatorID:       supervisorID,
		FormulaVersion:    s.cfg.FormulaVersion,
		ObjectiveScore:    intPtrToUint8(req.Objective),
		ContentScore:      intPtrToUint8(req.Content),
		InteractionScore:  intPtrToUint8(req.Interaction),
		OrganizationScore: intPtrToUint8(req.Organization),
		FrontierScore:     intPtrToUint8(req.Frontier),
		TotalScore:        total,
		Comment:           req.Comment,
		Highlights:        req.Highlights,
		Improvements:      req.Improvements,
		Suggestions:       req.Suggestions,
	}
	if err := s.evals.Upsert(ctx, record); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "保存督导评分失败", err)
	}

	// 督导评分写入后场次进入 evaluated（无草稿态，写入即生效）。
	if err := s.sessions.UpdateStatus(ctx, sessionID, model.SessionStatusEvaluated); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "更新授课记录状态失败", err)
	}

	// 回读持久化结果，返回真实落库状态（含幂等覆盖后的 updated_at）。
	rows, err := s.evals.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "回读督导评分失败", err)
	}
	for _, row := range rows {
		if row.EvaluatorType == model.EvaluatorSupervisor && row.EvaluatorID == supervisorID {
			dtoRow := toEvaluationDTO(row)
			return &dtoRow, nil
		}
	}
	return nil, errcode.Wrap(errcode.Internal, "回读督导评分失败", errcode.New(errcode.Internal, "评分写入后未找到"))
}

// checkSessionScope 校验授课记录落在数据范围内（同 checkCourseScope 口径）。
func checkSessionScope(scope Scope, row *repository.SessionDetailRow) error {
	if scope.DepartmentID > 0 && row.DepartmentID != scope.DepartmentID {
		return errcode.New(errcode.ForbiddenData, "无权查看其他教研室的授课记录")
	}
	if scope.TeacherID > 0 && row.TeacherID != scope.TeacherID {
		return errcode.New(errcode.ForbiddenData, "无权查看他人的授课记录")
	}
	return nil
}

// intPtrToUint8 转换请求体的维度分（1-5 已由 validator 约束）。
func intPtrToUint8(v *int) *uint8 {
	if v == nil {
		return nil
	}
	u := uint8(*v)
	return &u
}
