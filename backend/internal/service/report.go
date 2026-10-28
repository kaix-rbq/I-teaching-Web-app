package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"aijiaoxue-api/internal/agent"
	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/scoring"
)

const reportTaskPayloadVersion = 1

// ReportEnqueuer 是 S1 完成后触发 S2 的最小依赖，避免 aiTaskService 依赖整个 ReportService。
type ReportEnqueuer interface {
	Enqueue(teacherID uint64, semester string, force bool) error
}

// ReportService 负责教师级 AI 报告（S2）的异步入队、去抖、快照与读取。
type ReportService interface {
	Enqueue(teacherID uint64, semester string, force bool) error
	ProcessTask(ctx context.Context, taskID uint64)
	Start(ctx context.Context)
	GetReport(ctx context.Context, role string, deptID, userID, teacherID uint64, semester string) (*dto.AiReportEnvelope, error)
	Regenerate(ctx context.Context, role string, deptID, userID, teacherID uint64, semester string) (*dto.AiReportRegenerateResp, error)
}

type reportService struct {
	tasks    repository.AiTaskRepository
	reports  repository.TeacherAiReportRepository
	evals    repository.EvaluationRepository
	users    repository.UserRepository
	analyzer agent.ReportAnalyzer
	agentCfg config.AgentConfig
	evalCfg  config.EvaluationConfig
}

// NewReportService 构造教师级报告服务。
func NewReportService(
	tasks repository.AiTaskRepository,
	reports repository.TeacherAiReportRepository,
	evals repository.EvaluationRepository,
	users repository.UserRepository,
	analyzer agent.ReportAnalyzer,
	agentCfg config.AgentConfig,
	evalCfg config.EvaluationConfig,
) ReportService {
	return &reportService{
		tasks: tasks, reports: reports, evals: evals, users: users,
		analyzer: analyzer, agentCfg: agentCfg, evalCfg: evalCfg,
	}
}

// reportTaskPayload 记录任务输入；ai_tasks 唯一键不含 semester，payload 是唯一学期来源。
type reportTaskPayload struct {
	Version  int    `json:"version"`
	Semester string `json:"semester"`
	Force    bool   `json:"force"`
}

// Enqueue 创建或复位教师报告任务。force=true 时立即到期（用于人工重生成）。
// S1 调用方负责检查 reportEnabled；本方法只负责队列语义，便于 Regenerate 复用。
func (s *reportService) Enqueue(teacherID uint64, semester string, force bool) error {
	semester = strings.TrimSpace(semester)
	if teacherID == 0 || semester == "" {
		return fmt.Errorf("教师报告入队参数非法: teacherID=%d semester=%q", teacherID, semester)
	}
	payload, err := json.Marshal(reportTaskPayload{
		Version: reportTaskPayloadVersion, Semester: semester, Force: force,
	})
	if err != nil {
		return err
	}
	payloadStr := string(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()
	dueAt := now
	if force {
		dueAt = now.Add(-s.agentCfg.ReportDebounceDuration())
	}

	existing, err := s.tasks.GetByKindRef(ctx, model.AiTaskKindTeacherReport, model.AiTaskRefTeacher, teacherID)
	if err == nil {
		return s.tasks.Update(ctx, existing.ID, map[string]any{
			"status": model.AiTaskStatusPending, "attempts": 0,
			"payload": payloadStr, "error_message": "",
			"started_at": nil, "finished_at": nil, "updated_at": dueAt,
		})
	}
	if !repository.IsNotFound(err) {
		return err
	}

	task := &model.AiTask{
		Kind: model.AiTaskKindTeacherReport, RefType: model.AiTaskRefTeacher, RefID: teacherID,
		Status: model.AiTaskStatusPending, Payload: &payloadStr,
		UpdatedAt: dueAt,
	}
	return s.tasks.CreateIfAbsent(ctx, task)
}

// Start 启动 60 秒 ticker；只由 cmd/server/main.go 调用一次。
// reportEnabled=false 时不启动生成 worker，读接口仍可正常工作。
func (s *reportService) Start(ctx context.Context) {
	if !s.agentCfg.ReportEnabled {
		slog.Info("教师级 AI 报告生成未启用")
		return
	}
	go func() {
		s.recoverStuck(ctx)
		s.processDue(ctx)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.processDue(ctx)
			}
		}
	}()
}

// recoverStuck 把崩溃遗留的 teacher_report running 任务复位为 pending 并立即到期。
// S1 的 ResumeStuck 会跳过 teacher_report，因此这里必须单独处理，避免任务永久卡住。
func (s *reportService) recoverStuck(ctx context.Context) {
	rows, err := s.tasks.ListResumable(ctx)
	if err != nil {
		slog.Warn("恢复遗留教师报告任务失败", "err", err)
		return
	}
	for _, task := range rows {
		if task.Kind != model.AiTaskKindTeacherReport || task.Status != model.AiTaskStatusRunning {
			continue
		}
		fields := map[string]any{
			"status": model.AiTaskStatusPending, "error_message": "",
			"started_at": nil, "updated_at": time.Now().Add(-s.agentCfg.ReportDebounceDuration()),
		}
		if err := s.tasks.Update(ctx, task.ID, fields); err != nil {
			slog.Warn("复位遗留教师报告任务失败", "taskID", task.ID, "err", err)
		}
	}
}

func (s *reportService) processDue(ctx context.Context) {
	before := time.Now().Add(-s.agentCfg.ReportDebounceDuration())
	tasks, err := s.tasks.ListDueTeacherReportTasks(ctx, before)
	if err != nil {
		slog.Warn("扫描教师报告任务失败", "err", err)
		return
	}
	for _, task := range tasks {
		go s.ProcessTask(ctx, task.ID)
	}
}

// ProcessTask 执行一次教师报告生成；所有失败只落报告/任务状态，不向 HTTP 主链路传播。
func (s *reportService) ProcessTask(parent context.Context, taskID uint64) {
	if !s.agentCfg.ReportEnabled {
		return
	}
	ctx, cancel := context.WithTimeout(parent, s.agentCfg.TimeoutDuration())
	defer cancel()

	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return
	}
	if task.Kind != model.AiTaskKindTeacherReport || task.RefType != model.AiTaskRefTeacher ||
		task.Status == model.AiTaskStatusDone {
		return
	}

	failTask := func(reason string) {
		reason = truncateRunes(reason, aiTaskErrMaxLen)
		failCtx, cancelFail := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFail()
		_ = s.tasks.Update(failCtx, taskID, map[string]any{
			"status": model.AiTaskStatusFailed, "error_message": reason, "finished_at": time.Now(),
		})
	}

	if task.Attempts >= model.AiTaskMaxAttempts {
		failTask("教师报告生成重试次数已达上限")
		slog.Warn("教师报告达到重试上限", "taskID", taskID, "attempts", task.Attempts)
		return
	}
	if err := s.tasks.Update(ctx, taskID, map[string]any{
		"status": model.AiTaskStatusRunning, "attempts": task.Attempts + 1,
		"started_at": time.Now(), "error_message": "",
	}); err != nil {
		slog.Error("教师报告任务置 running 失败", "taskID", taskID, "err", err)
		return
	}

	var payload reportTaskPayload
	if task.Payload == nil || json.Unmarshal([]byte(*task.Payload), &payload) != nil {
		failTask("教师报告任务参数非法")
		return
	}
	semester := strings.TrimSpace(payload.Semester)
	if semester == "" {
		failTask("教师报告任务学期为空")
		return
	}
	teacherID := task.RefID

	failReport := func(reason string) {
		failCtx, cancelFail := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFail()
		if err := s.reports.MarkStatus(failCtx, teacherID, semester, model.AiTaskStatusFailed, truncateRunes(reason, aiTaskErrMaxLen)); err != nil {
			slog.Warn("写入教师报告 failed 状态失败", "teacherID", teacherID, "semester", semester, "err", err)
		}
		failTask(reason)
	}

	report, err := s.reports.Get(ctx, teacherID, semester)
	if err != nil && !repository.IsNotFound(err) {
		failReport("查询教师报告失败")
		return
	}
	if repository.IsNotFound(err) {
		report = nil
	}

	rows, err := s.evals.ListTeacherReportRows(ctx, teacherID, semester, s.formulaVersion())
	if err != nil {
		failReport("查询教师评价摘要失败")
		return
	}
	sessionIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		sessionIDs = append(sessionIDs, row.SessionID)
	}
	sort.Slice(sessionIDs, func(i, j int) bool { return sessionIDs[i] < sessionIDs[j] })

	// 快照未变且已成功生成：直接完成任务，不调用模型（force 跳过）。
	if report != nil && report.Status == model.AiTaskStatusDone && !payload.Force &&
		sameUint64s(decodeUint64Array(report.SourceSessionIDs), sessionIDs) {
		s.logTaskDone(ctx, taskID)
		return
	}

	// 先确保报告行进入 running，这样后续任何失败都能落 status='failed'。
	if report == nil {
		if err := s.reports.Upsert(ctx, &model.TeacherAiReport{
			TeacherID: teacherID, Semester: semester, CourseID: 0,
			FormulaVersion: s.formulaVersion(), AIModelVersion: s.agentCfg.LLMModel,
			PromptVersion: s.promptVersion(), Status: model.AiTaskStatusRunning,
		}); err != nil {
			failReport("创建教师报告行失败")
			return
		}
	} else if err := s.reports.MarkStatus(ctx, teacherID, semester, model.AiTaskStatusRunning, ""); err != nil {
		failReport("更新教师报告 running 状态失败")
		return
	}

	if s.analyzer == nil {
		failReport("AI 报告引擎未配置")
		return
	}

	allRows, err := s.evals.ListSessionDimRows(ctx, repository.SessionDimFilter{
		TeacherID: teacherID, Semester: semester, FormulaVersion: s.formulaVersion(),
	})
	if err != nil {
		failReport("查询教师趋势数据失败")
		return
	}
	comments, err := s.evals.ListSupervisorCommentsByTeacherSemester(ctx, teacherID, semester)
	if err != nil {
		failReport("查询督导评语失败")
		return
	}

	input := s.buildInput(ctx, teacherID, semester, rows, allRows, comments)
	result, err := s.analyzer.AnalyzeReport(ctx, input)
	if err != nil {
		failReport("AI 报告调用失败")
		slog.Error("AI 报告调用失败", "taskID", taskID, "teacherID", teacherID, "err", err)
		return
	}

	now := time.Now()
	summary := result.Summary
	highlights := result.Highlights
	trendComment := result.TrendComment
	confidence := clampConfidence(&result.Confidence)
	directions := mustJSON(append([]string{}, result.Directions...))
	suggestions := mustJSON(append([]string{}, result.Suggestions...))
	evidence := mustJSON(map[string]any{
		"citedChunks":   []string{},
		"keySessionIds": sessionIDs,
	})
	sourceIDs := mustJSON(sessionIDs)
	sample := mustJSON(map[string]any{
		"evaluatedCount": input.EvaluatedCount,
		"sessionCount":   input.SessionCount,
		"flags":          input.Flags,
	})
	reportRow := &model.TeacherAiReport{
		TeacherID: teacherID, Semester: semester, CourseID: 0,
		FormulaVersion: s.formulaVersion(), AIModelVersion: s.agentCfg.LLMModel,
		PromptVersion: s.promptVersion(),
		Summary:       &summary, Directions: &directions, Suggestions: &suggestions,
		Highlights: &highlights, TrendComment: &trendComment,
		Evidence: &evidence, SourceSessionIDs: &sourceIDs, Sample: &sample,
		Confidence: &confidence, Status: model.AiTaskStatusDone,
		ErrorMessage: "", GeneratedAt: &now,
	}
	if err := s.reports.Upsert(ctx, reportRow); err != nil {
		failReport("教师报告保存失败")
		slog.Error("保存教师报告失败", "taskID", taskID, "teacherID", teacherID, "err", err)
		return
	}
	s.logTaskDone(ctx, taskID)
	slog.Info("教师报告生成完成", "taskID", taskID, "teacherID", teacherID, "semester", semester)
}

func (s *reportService) logTaskDone(ctx context.Context, taskID uint64) {
	if err := s.tasks.Update(ctx, taskID, map[string]any{
		"status": model.AiTaskStatusDone, "error_message": "", "finished_at": time.Now(),
	}); err != nil {
		slog.Error("教师报告任务置 done 失败", "taskID", taskID, "err", err)
	}
}

func (s *reportService) buildInput(
	ctx context.Context, teacherID uint64, semester string,
	rows []repository.TeacherReportRow,
	allRows []repository.SessionDimRow,
	comments []string,
) agent.ReportInput {
	teacher, _ := s.users.GetByID(ctx, teacherID)
	teacherName := ""
	if teacher != nil {
		teacherName = teacher.Name
	}

	items := make([]scoring.SessionScore, 0, len(allRows))
	for _, row := range allRows {
		items = append(items, sessionScoreFromDim(row))
	}
	summary := scoring.Aggregate(items, s.evalCfg.ScoringWeights(), 0)

	sessions := make([]agent.ReportSession, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, agent.ReportSession{
			SessionID: row.SessionID, CourseID: row.CourseID,
			CourseCode: row.CourseCode, CourseName: row.CourseName,
			SessionDate: row.SessionDate.Format("2006-01-02"),
			Period:      row.Period, Topic: row.Topic,
			Objective:    uint8PtrToIntPtr(row.ObjectiveScore),
			Content:      uint8PtrToIntPtr(row.ContentScore),
			Interaction:  uint8PtrToIntPtr(row.InteractionScore),
			Organization: uint8PtrToIntPtr(row.OrganizationScore),
			Frontier:     uint8PtrToIntPtr(row.FrontierScore),
			Comment:      row.Comment, Highlights: row.Highlights,
			Improvements: row.Improvements, Suggestions: row.Suggestions,
			AIModelVersion: row.AIModelVersion, AIConfidence: row.AIConfidence,
		})
	}
	return agent.ReportInput{
		TeacherID: teacherID, TeacherName: teacherName, Semester: semester,
		EvaluatedCount:     summary.Sample.EvaluatedCount,
		SessionCount:       len(allRows),
		MinSampleSize:      s.evalCfg.MinSampleSize,
		Flags:              append([]string{}, summary.Flags...),
		Sessions:           sessions,
		Trend:              buildReportTrend(allRows, s.evalCfg.ScoringWeights()),
		SupervisorComments: append([]string{}, comments...),
	}
}

// buildReportTrend 复用 P2 的 sessionTrendValues，避免出现第二套趋势口径。
func buildReportTrend(rows []repository.SessionDimRow, weights scoring.Weights) []agent.ReportTrend {
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].SessionDate.Equal(rows[j].SessionDate) {
			return rows[i].SessionDate.Before(rows[j].SessionDate)
		}
		return rows[i].SessionID < rows[j].SessionID
	})
	out := make([]agent.ReportTrend, 0, len(trendSeriesMeta))
	for _, meta := range trendSeriesMeta {
		points := make([]agent.ReportTrendPoint, 0, len(rows))
		for index, row := range rows {
			values := sessionTrendValues(row, weights)
			points = append(points, agent.ReportTrendPoint{
				Label: fmt.Sprintf("第 %d 次课", index+1),
				Date:  row.SessionDate.Format("2006-01-02"),
				Value: values.forSeries(meta.Key),
			})
		}
		out = append(out, agent.ReportTrend{
			Key: meta.Key, Name: meta.Name, IsObservation: meta.IsObservation, Points: points,
		})
	}
	return out
}

// GetReport 返回教师级报告；无报告时 report=null，status=pending 供前端轮询。
func (s *reportService) GetReport(
	ctx context.Context, role string, deptID, userID, teacherID uint64, semester string,
) (*dto.AiReportEnvelope, error) {
	sem, err := resolveSemester(semester)
	if err != nil {
		return nil, err
	}
	if _, err := s.checkTeacherAccess(ctx, role, deptID, userID, teacherID); err != nil {
		return nil, err
	}
	report, err := s.reports.Get(ctx, teacherID, sem)
	if err != nil {
		if repository.IsNotFound(err) {
			return &dto.AiReportEnvelope{Status: model.AiTaskStatusPending}, nil
		}
		return nil, errcode.Wrap(errcode.Internal, "查询教师报告失败", err)
	}
	return &dto.AiReportEnvelope{
		Status:       report.Status,
		Report:       toAiReportDTO(report),
		ErrorMessage: report.ErrorMessage,
	}, nil
}

// Regenerate 复位报告为 pending，并把任务置为立即到期；不等待生成完成。
func (s *reportService) Regenerate(
	ctx context.Context, role string, deptID, userID, teacherID uint64, semester string,
) (*dto.AiReportRegenerateResp, error) {
	sem, err := resolveSemester(semester)
	if err != nil {
		return nil, err
	}
	switch role {
	case RoleSupervisor:
	case RoleDirector:
	default:
		return nil, errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message())
	}
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
	if role == RoleDirector && teacher.DeptID() != deptID {
		return nil, errcode.New(errcode.ForbiddenData, "无权操作其他教研室的教师")
	}

	_, err = s.reports.Get(ctx, teacherID, sem)
	if err != nil && !repository.IsNotFound(err) {
		return nil, errcode.Wrap(errcode.Internal, "查询教师报告失败", err)
	}
	if repository.IsNotFound(err) {
		status := model.AiTaskStatusPending
		if uerr := s.reports.Upsert(ctx, &model.TeacherAiReport{
			TeacherID: teacherID, Semester: sem, CourseID: 0,
			FormulaVersion: s.formulaVersion(), AIModelVersion: s.agentCfg.LLMModel,
			PromptVersion: s.promptVersion(), Status: status,
		}); uerr != nil {
			return nil, errcode.Wrap(errcode.Internal, "创建教师报告失败", uerr)
		}
	} else if err := s.reports.MarkStatus(ctx, teacherID, sem, model.AiTaskStatusPending, ""); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "复位教师报告失败", err)
	}

	if err := s.Enqueue(teacherID, sem, true); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "教师报告任务入队失败", err)
	}
	return &dto.AiReportRegenerateResp{Status: model.AiTaskStatusPending}, nil
}

func (s *reportService) formulaVersion() string {
	if v := strings.TrimSpace(s.evalCfg.FormulaVersion); v != "" {
		return v
	}
	return "v1"
}

func (s *reportService) promptVersion() string {
	if v := strings.TrimSpace(s.agentCfg.PromptVersion); v != "" {
		return v
	}
	return "v1"
}

// checkTeacherAccess 与教师评分/趋势接口同口径：teacher 本人 / director 本室 / supervisor 全校。
func (s *reportService) checkTeacherAccess(
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
			return nil, errcode.New(errcode.ForbiddenData, "教师只能查看本人的报告")
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

func toAiReportDTO(report *model.TeacherAiReport) *dto.AiReportDTO {
	sample := decodeReportSample(report.Sample)
	return &dto.AiReportDTO{
		TeacherID: report.TeacherID, Semester: report.Semester, CourseID: report.CourseID,
		Summary: stringValue(report.Summary), Directions: decodeStringArray(report.Directions),
		Suggestions: decodeStringArray(report.Suggestions), Highlights: stringValue(report.Highlights),
		TrendComment: stringValue(report.TrendComment), Confidence: report.Confidence,
		EvaluatedCount: sample.EvaluatedCount, SessionCount: sample.SessionCount,
		SourceSessionIDs: decodeUint64Array(report.SourceSessionIDs),
		ModelVersion:     report.AIModelVersion, PromptVersion: report.PromptVersion,
		Status: report.Status, ErrorMessage: report.ErrorMessage,
		GeneratedAt: timeValue(report.GeneratedAt),
	}
}

type reportSample struct {
	EvaluatedCount int      `json:"evaluatedCount"`
	SessionCount   int      `json:"sessionCount"`
	Flags          []string `json:"flags"`
}

func decodeReportSample(raw *string) reportSample {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return reportSample{}
	}
	var out reportSample
	_ = json.Unmarshal([]byte(*raw), &out)
	return out
}

func decodeStringArray(raw *string) []string {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(*raw), &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

func decodeUint64Array(raw *string) []uint64 {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return []uint64{}
	}
	var out []uint64
	if err := json.Unmarshal([]byte(*raw), &out); err != nil || out == nil {
		return []uint64{}
	}
	return out
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func timeValue(v *time.Time) string {
	if v == nil {
		return ""
	}
	return v.Format(time.RFC3339)
}

func mustJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func sameUint64s(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func truncateRunes(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit])
}
