package service

import (
	"context"
	"log/slog"
	"os"
	"strings"
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
	Create(ctx context.Context, role string, deptID, userID uint64, req dto.SessionCreateReq) (*dto.SessionDetail, error)
	// Delete 删课并级联清理录音文件/转写/评价/清单关联（D6，service 事务）。
	Delete(ctx context.Context, role string, deptID, userID, sessionID uint64) error
	ListByCourse(ctx context.Context, role string, deptID, userID, courseID uint64, q dto.SessionListQuery) (*dto.PageResult[dto.SessionListItem], error)
	Detail(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.SessionDetail, error)
	Evaluation(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.SessionEvaluation, error)
	SubmitSupervisorEvaluation(ctx context.Context, supervisorID, sessionID uint64, req dto.SupervisorEvaluationReq) (*dto.EvaluationDTO, error)
	// SubmitSupervisorObservation 提交/覆盖督导纯评语观察（B8，无五维分，至少填一项）。
	SubmitSupervisorObservation(ctx context.Context, supervisorID, sessionID uint64, req dto.SupervisorObservationReq) (*dto.EvaluationDTO, error)
}

type sessionService struct {
	sessions     repository.SessionRepository
	evals        repository.EvaluationRepository
	courses      repository.CourseRepository
	users        repository.UserRepository
	recordings   repository.RecordingRepository
	improvements repository.ImprovementRepository
	cfg          config.EvaluationConfig
	notifier     Notifier
}

// NewSessionService 构造授课记录服务。notifier 可选（站内信副作用），测试可不传。
func NewSessionService(
	sessions repository.SessionRepository,
	evals repository.EvaluationRepository,
	courses repository.CourseRepository,
	users repository.UserRepository,
	cfg config.EvaluationConfig,
	notifier ...Notifier,
) SessionService {
	svc := &sessionService{sessions: sessions, evals: evals, courses: courses, users: users, cfg: cfg}
	if len(notifier) > 0 {
		svc.notifier = notifier[0]
	}
	return svc
}

// WithDeleteDeps 注入删课级联所需仓储（recordings / improvements）。
// 单独方法而非构造参数：删课是可选能力，避免污染既有构造签名与测试。
func (s *sessionService) WithDeleteDeps(recordings repository.RecordingRepository, improvements repository.ImprovementRepository) *sessionService {
	s.recordings = recordings
	s.improvements = improvements
	return s
}

func NewSessionServiceWithDelete(
	sessions repository.SessionRepository,
	evals repository.EvaluationRepository,
	courses repository.CourseRepository,
	users repository.UserRepository,
	recordings repository.RecordingRepository,
	improvements repository.ImprovementRepository,
	cfg config.EvaluationConfig,
	notifier ...Notifier,
) SessionService {
	svc := &sessionService{
		sessions: sessions, evals: evals, courses: courses, users: users,
		recordings: recordings, improvements: improvements, cfg: cfg,
	}
	if len(notifier) > 0 {
		svc.notifier = notifier[0]
	}
	return svc
}

// Create 创建授课记录：评价的落点（S6.1）。教师建课（本人课程）与督导代录共用一个入口（B1）。
// 业务规则：日期不得晚于今天（40002）；课程必须存在（40401）；同课程同班级同日同节次唯一（40901）；
// 教师仅可为自己任课的课程建课（40302）；建课人写入 created_by（V12）。
func (s *sessionService) Create(ctx context.Context, role string, deptID, userID uint64, req dto.SessionCreateReq) (*dto.SessionDetail, error) {
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
	// 权限矩阵（计划 §5）：teacher 仅本人课程；supervisor 全部（代录）。
	switch role {
	case RoleTeacher:
		if course.TeacherID != userID {
			return nil, errcode.New(errcode.ForbiddenData, "只能为本人的课程创建授课记录")
		}
	case RoleSupervisor:
	default:
		return nil, errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message())
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

	// B11/classId 必填（计划 §4.1）：杜绝 class_id=0 导致的 uk_session 碰撞。
	// 例外：经听评课计划快捷建课（planId 非空）时计划本身无班级信息，允许 0。
	if req.ClassID == 0 && (req.PlanID == nil || *req.PlanID == 0) {
		return nil, errcode.New(errcode.BizRule, "创建授课记录须选择班级")
	}
	if req.ClassID != 0 {
		classes, cerr := s.courses.ListClasses(ctx, req.CourseID)
		if cerr != nil {
			return nil, errcode.Wrap(errcode.Internal, "查询开课班级失败", cerr)
		}
		ok := false
		for _, c := range classes {
			if c.ID == req.ClassID {
				ok = true
				break
			}
		}
		if !ok {
			return nil, errcode.New(errcode.BizRule, "班级不归属该课程")
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
		CreatedBy:   userID,
		SessionDate: date,
		Period:      req.Period,
		Topic:       req.Topic,
		PlanID:      req.PlanID,
		Status:      model.SessionStatusScheduled,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		// 唯一性预检与写入之间存在并发窗口，数据库唯一键兜底时同样返回 40901。
		if repository.IsDuplicate(err) {
			return nil, errcode.New(errcode.Conflict, "该课程同日同节次已存在授课记录")
		}
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
	// 督导已评 = 存在督导评价行；不复用 status（status 只表示 AI 已评析）。
	out.SupervisorEvaluated = len(out.SupervisorScores) > 0
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
	// 督导评分强制 5 个维度全部录入（§2.1：只有智能体侧允许 null）；缺失返回 40002（§4 错误码表）。
	if missing := missingDimensions(req); len(missing) > 0 {
		return nil, errcode.Newf(errcode.BizRule, "评分维度缺失：%s，五个维度均为必填", strings.Join(missing, "、"))
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

// SubmitSupervisorObservation 提交/覆盖督导纯评语观察（B8 / §4.3）：
// 复用 supervisor 行、五维全 NULL、只写评语列，不写 total_score，不改场次状态；
// 计分只按 AI，督导意见是独立「督导观察」模块（§6）。
func (s *sessionService) SubmitSupervisorObservation(
	ctx context.Context, supervisorID, sessionID uint64, req dto.SupervisorObservationReq,
) (*dto.EvaluationDTO, error) {
	if _, err := s.sessions.GetDetail(ctx, sessionID); err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "授课记录不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}
	if strings.TrimSpace(req.Comment) == "" && strings.TrimSpace(req.Highlights) == "" &&
		strings.TrimSpace(req.Improvements) == "" && strings.TrimSpace(req.Suggestions) == "" {
		return nil, errcode.New(errcode.BizRule, "督导观察至少填写一项评语")
	}

	record := &model.Evaluation{
		SessionID:      sessionID,
		EvaluatorType:  model.EvaluatorSupervisor,
		EvaluatorID:    supervisorID,
		FormulaVersion: s.cfg.FormulaVersion,
		Comment:        req.Comment,
		Highlights:     req.Highlights,
		Improvements:   req.Improvements,
		Suggestions:    req.Suggestions,
	}
	if err := s.evals.Upsert(ctx, record); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "保存督导观察失败", err)
	}
	// B12：督导评语 → 站内信通知授课教师（副作用，失败不影响主流程）。
	if s.notifier != nil {
		s.notifier.NotifySupervisorComment(ctx, sessionID)
	}

	rows, err := s.evals.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "回读督导观察失败", err)
	}
	for _, row := range rows {
		if row.EvaluatorType == model.EvaluatorSupervisor && row.EvaluatorID == supervisorID {
			dtoRow := toEvaluationDTO(row)
			return &dtoRow, nil
		}
	}
	return nil, errcode.New(errcode.Internal, "督导观察写入后未找到")
}

// Delete 删课：级联清理录音文件 / 转写 / 评价 / 清单关联，最后删授课记录（D6）。
//
// 权限（计划 §5）：该课程任课教师本人 / supervisor。
// 删除顺序自子到父（改进项 → 评价 → 转写 → 录音 → 场次），满足外键约束；
// 磁盘文件删除失败只记日志，不回滚（与资源删除口径一致）。
func (s *sessionService) Delete(ctx context.Context, role string, deptID, userID, sessionID uint64) error {
	row, err := s.sessions.GetDetail(ctx, sessionID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errcode.New(errcode.NotFound, "授课记录不存在")
		}
		return errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}
	switch role {
	case RoleSupervisor:
	case RoleTeacher:
		if row.TeacherID != userID {
			return errcode.New(errcode.ForbiddenData, "只能删除本人课程的授课记录")
		}
	default:
		return errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message())
	}

	if s.improvements != nil {
		if err := s.improvements.DeleteBySession(ctx, sessionID); err != nil {
			return errcode.Wrap(errcode.Internal, "清理清单关联失败", err)
		}
	}
	if err := s.evals.DeleteBySession(ctx, sessionID); err != nil {
		return errcode.Wrap(errcode.Internal, "清理课堂评价失败", err)
	}
	var filePath string
	if s.recordings != nil {
		if rec, rerr := s.recordings.GetRecordingBySession(ctx, sessionID); rerr == nil && rec != nil {
			filePath = rec.FilePath
			_ = s.recordings.DeletePlaybackLogsByRecording(ctx, rec.ID)
			if err := s.recordings.DeleteRecording(ctx, rec.ID); err != nil {
				return errcode.Wrap(errcode.Internal, "清理录音记录失败", err)
			}
		}
		if err := s.recordings.DeleteTranscriptBySession(ctx, sessionID); err != nil {
			return errcode.Wrap(errcode.Internal, "清理转写记录失败", err)
		}
	}
	if err := s.sessions.Delete(ctx, sessionID); err != nil {
		return errcode.Wrap(errcode.Internal, "删除授课记录失败", err)
	}
	if filePath != "" {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			slog.Warn("删除录音文件失败", "sessionID", sessionID, "path", filePath, "err", err)
		}
	}
	return nil
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

// missingDimensions 返回督导评分请求中缺失的维度中文名（空表示五维齐全）。
// 顺序固定为 objective→content→interaction→organization→frontier，便于测试与提示。
func missingDimensions(req dto.SupervisorEvaluationReq) []string {
	checks := []struct {
		name string
		v    *int
	}{
		{"教学目标与内容准确性", req.Objective},
		{"内容质量与深度", req.Content},
		{"学生互动与参与", req.Interaction},
		{"课堂组织与节奏", req.Organization},
		{"前沿与交叉学科", req.Frontier},
	}
	var missing []string
	for _, c := range checks {
		if c.v == nil {
			missing = append(missing, c.name)
		}
	}
	return missing
}
