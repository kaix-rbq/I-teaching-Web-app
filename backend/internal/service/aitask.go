package service

import (
	"context"
	"encoding/json"
	"log/slog"
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

// aiTaskErrMaxLen 对齐 ai_tasks.error_message 的 VARCHAR(255)。
const aiTaskErrMaxLen = 255

// AiTaskService 负责 AI 评析任务链（D3 / R0.2）。
//
// 与督导主流程完全解耦（RAG §1 硬约束）：任何失败只落 ai_tasks.status='failed'，
// 不影响转写、评分与页面可用性。AI 引擎未配置时任务记 failed 并给出可读原因。
type AiTaskService interface {
	// Analyze 入队（幂等）并发起 AI 评析；有未完成任务直接返回其 taskId。
	Analyze(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.AiAnalyzeResp, error)
	// Observation 返回 AI 观察详情；teacher 侧高风险标签脱敏（D8）。
	Observation(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.AiObservationDTO, error)
	// TaskStatus 查询任务状态（pending/running/done/failed）。
	TaskStatus(ctx context.Context, taskID uint64) (*dto.AiTaskDTO, error)
	// Retry 复位失败/已完成任务并重新执行（supervisor 可重试）。
	Retry(ctx context.Context, taskID uint64) (*dto.AiTaskDTO, error)
	// ResumeStuck 启动时把遗留的 pending/running 任务重新入队（幂等）。
	ResumeStuck(ctx context.Context) (int, error)
	// EnqueueSessionEval 供转写完成回调：异步入队一次 AI 评析（不阻塞调用方）。
	EnqueueSessionEval(sessionID uint64)
}

type aiTaskService struct {
	tasks      repository.AiTaskRepository
	sessions   repository.SessionRepository
	evals      repository.EvaluationRepository
	recordings repository.RecordingRepository
	analyzer   agent.SessionAnalyzer
	agentCfg   config.AgentConfig
	evalCfg    config.EvaluationConfig
	sem        chan struct{}
}

// NewAiTaskService 构造 AI 任务服务；analyzer 为 nil 表示 AI 引擎未配置。
func NewAiTaskService(
	tasks repository.AiTaskRepository,
	sessions repository.SessionRepository,
	evals repository.EvaluationRepository,
	recordings repository.RecordingRepository,
	analyzer agent.SessionAnalyzer,
	agentCfg config.AgentConfig,
	evalCfg config.EvaluationConfig,
) AiTaskService {
	return &aiTaskService{
		tasks:      tasks,
		sessions:   sessions,
		evals:      evals,
		recordings: recordings,
		analyzer:   analyzer,
		agentCfg:   agentCfg,
		evalCfg:    evalCfg,
		sem:        make(chan struct{}, agentCfg.Concurrency()),
	}
}

// Analyze 入队并发起评析。幂等：目标已有的未完成任务直接返回，不重复建（D3）。
func (s *aiTaskService) Analyze(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.AiAnalyzeResp, error) {
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

	task, started, err := s.enqueue(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if started {
		go s.process(context.Background(), task.ID)
	}
	return &dto.AiAnalyzeResp{TaskID: task.ID, Status: task.Status}, nil
}

// EnqueueSessionEval 转写完成回调：异步入队（幂等），不阻塞调用方。
func (s *aiTaskService) EnqueueSessionEval(sessionID uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	task, started, err := s.enqueue(ctx, sessionID)
	if err != nil {
		slog.Warn("转写完成后入队 AI 评析失败", "sessionID", sessionID, "err", err)
		return
	}
	if started {
		go s.process(context.Background(), task.ID)
	}
}

// enqueue 保证 (session_eval, session, id) 任务存在。
// started=true 表示任务刚创建或刚由终态复位，需要本次触发执行；false 表示已有未完成任务（幂等）。
func (s *aiTaskService) enqueue(ctx context.Context, sessionID uint64) (*model.AiTask, bool, error) {
	existing, err := s.tasks.GetByKindRef(ctx, model.AiTaskKindSessionEval, model.AiTaskRefSession, sessionID)
	if err == nil {
		if !model.IsTerminalAiTaskStatus(existing.Status) {
			return existing, false, nil
		}
		if err := s.tasks.Update(ctx, existing.ID, map[string]any{
			"status": model.AiTaskStatusPending, "error_message": "",
			"started_at": nil, "finished_at": nil,
		}); err != nil {
			return nil, false, errcode.Wrap(errcode.Internal, "重置 AI 任务失败", err)
		}
		existing.Status = model.AiTaskStatusPending
		existing.ErrorMessage = ""
		return existing, true, nil
	}
	if !repository.IsNotFound(err) {
		return nil, false, errcode.Wrap(errcode.Internal, "查询 AI 任务失败", err)
	}

	task := &model.AiTask{
		Kind:    model.AiTaskKindSessionEval,
		RefType: model.AiTaskRefSession,
		RefID:   sessionID,
		Status:  model.AiTaskStatusPending,
	}
	if err := s.tasks.CreateIfAbsent(ctx, task); err != nil {
		return nil, false, errcode.Wrap(errcode.Internal, "创建 AI 任务失败", err)
	}
	// 并发窗口内可能已被他处创建：回读真实行；若已是 pending 则视为本次需触发。
	got, gerr := s.tasks.GetByKindRef(ctx, model.AiTaskKindSessionEval, model.AiTaskRefSession, sessionID)
	if gerr != nil {
		return task, true, nil
	}
	return got, got.Status == model.AiTaskStatusPending, nil
}

// process 是 AI 评析异步任务体。失败只落 ai_tasks.status='failed'。
func (s *aiTaskService) process(parent context.Context, taskID uint64) {
	ctx, cancel := context.WithTimeout(parent, s.agentCfg.TimeoutDuration())
	defer cancel()

	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return
	}
	fail := func(reason string) {
		if len([]rune(reason)) > aiTaskErrMaxLen {
			reason = string([]rune(reason)[:aiTaskErrMaxLen])
		}
		failCtx, cancelFail := context.WithTimeout(parent, 10*time.Second)
		defer cancelFail()
		_ = s.tasks.Update(failCtx, taskID, map[string]any{
			"status": model.AiTaskStatusFailed, "error_message": reason, "finished_at": time.Now(),
		})
	}

	if err := s.tasks.Update(ctx, taskID, map[string]any{
		"status": model.AiTaskStatusRunning, "attempts": task.Attempts + 1,
		"started_at": time.Now(), "error_message": "",
	}); err != nil {
		slog.Error("AI 任务置 running 失败", "taskID", taskID, "err", err)
		return
	}

	if s.analyzer == nil {
		fail("AI 评析引擎未配置")
		slog.Warn("AI 评析跳过：引擎未配置", "taskID", taskID)
		return
	}

	session, err := s.sessions.GetDetail(ctx, task.RefID)
	if err != nil {
		fail("查询授课记录失败")
		return
	}
	tr, err := s.recordings.GetTranscriptBySession(ctx, task.RefID)
	if err != nil || strings.TrimSpace(tr.Content) == "" {
		fail("转写未完成")
		return
	}

	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		fail("评析排队超时")
		return
	}

	result, err := s.analyzer.AnalyzeSession(ctx, agent.EvalInput{
		SessionID:   session.ID,
		CourseName:  session.CourseName,
		TeacherName: session.TeacherName,
		SessionDate: session.SessionDate.Format("2006-01-02"),
		Period:      session.Period,
		Topic:       session.Topic,
		Transcript:  tr.Content,
		Segments:    decodeSegments(tr.Segments),
	})
	if err != nil {
		fail("AI 评析调用失败")
		slog.Error("AI 评析调用失败", "taskID", taskID, "sessionID", task.RefID, "err", err)
		return
	}

	if err := s.writeAgentEvaluation(ctx, session.ID, result); err != nil {
		fail("AI 评析结果保存失败")
		slog.Error("写入 AI 评析结果失败", "taskID", taskID, "sessionID", session.ID, "err", err)
		return
	}
	if err := s.sessions.UpdateStatus(ctx, session.ID, model.SessionStatusEvaluated); err != nil {
		slog.Warn("更新场次状态为 evaluated 失败", "sessionID", session.ID, "err", err)
	}
	if err := s.tasks.Update(ctx, taskID, map[string]any{
		"status": model.AiTaskStatusDone, "error_message": "", "finished_at": time.Now(),
	}); err != nil {
		slog.Error("AI 任务置 done 失败", "taskID", taskID, "err", err)
		return
	}
	slog.Info("AI 评析完成", "taskID", taskID, "sessionID", session.ID)
}

// writeAgentEvaluation 把 S1 结果落成 evaluations（agent 行），按 uk_eval 幂等覆盖。
func (s *aiTaskService) writeAgentEvaluation(ctx context.Context, sessionID uint64, r *agent.Result) error {
	evidenceJSON, err := json.Marshal(r.Evidence)
	if err != nil {
		return err
	}
	evidence := string(evidenceJSON)
	confidence := r.Confidence

	dims := scoring.NewDimensionScores(r.Objective, r.Content, r.Interaction, r.Organization, r.Frontier)
	record := &model.Evaluation{
		SessionID:         sessionID,
		EvaluatorType:     model.EvaluatorAgent,
		EvaluatorID:       0,
		AIModelVersion:    r.ModelVersion,
		FormulaVersion:    s.evalCfg.FormulaVersion,
		ObjectiveScore:    intPtrToUint8(r.Objective),
		ContentScore:      intPtrToUint8(r.Content),
		InteractionScore:  intPtrToUint8(r.Interaction),
		OrganizationScore: intPtrToUint8(r.Organization),
		FrontierScore:     intPtrToUint8(r.Frontier),
		TotalScore:        scoring.SideTotal(dims, s.evalCfg.ScoringWeights()),
		AIConfidence:      &confidence,
		Evidence:          &evidence,
		Comment:           r.Comment,
		Highlights:        r.Highlights,
		Improvements:      r.Improvements,
		Suggestions:       r.Suggestions,
	}
	return s.evals.Upsert(ctx, record)
}

// Observation 返回 AI 观察详情（§4.2）。teacher 侧风险标签脱敏（D8）。
func (s *aiTaskService) Observation(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.AiObservationDTO, error) {
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

	out := &dto.AiObservationDTO{SessionID: sessionID, Risks: []string{}}
	if task, terr := s.tasks.GetByKindRef(ctx, model.AiTaskKindSessionEval, model.AiTaskRefSession, sessionID); terr == nil {
		out.TaskID = task.ID
		out.Status = task.Status
		out.ErrorMessage = task.ErrorMessage
	}

	rows, err := s.evals.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询 AI 观察失败", err)
	}
	for _, evalRow := range rows {
		if evalRow.EvaluatorType != model.EvaluatorAgent {
			continue
		}
		item := toEvaluationDTO(evalRow)
		out.Observation = &item
		if item.Evidence != nil && len(item.Evidence.RiskTags) > 0 {
			out.Risks = item.Evidence.RiskTags
		}
		break
	}

	// D8：高风险标签永不进教师端。
	if role == RoleTeacher {
		out.Risks = []string{}
		if out.Observation != nil && out.Observation.Evidence != nil {
			out.Observation.Evidence.RiskTags = nil
		}
	}
	return out, nil
}

// TaskStatus 查询任务状态。
func (s *aiTaskService) TaskStatus(ctx context.Context, taskID uint64) (*dto.AiTaskDTO, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "AI 任务不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询 AI 任务失败", err)
	}
	return toAiTaskDTO(task), nil
}

// Retry 复位并重新执行（supervisor 可重试，§4.2）。
func (s *aiTaskService) Retry(ctx context.Context, taskID uint64) (*dto.AiTaskDTO, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "AI 任务不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询 AI 任务失败", err)
	}
	if err := s.tasks.Update(ctx, taskID, map[string]any{
		"status": model.AiTaskStatusPending, "error_message": "", "started_at": nil, "finished_at": nil,
	}); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "重置 AI 任务失败", err)
	}
	go s.process(context.Background(), taskID)
	task.Status = model.AiTaskStatusPending
	task.ErrorMessage = ""
	return toAiTaskDTO(task), nil
}

// ResumeStuck 启动时重排遗留任务（幂等）。
func (s *aiTaskService) ResumeStuck(ctx context.Context) (int, error) {
	rows, err := s.tasks.ListResumable(ctx)
	if err != nil {
		return 0, errcode.Wrap(errcode.Internal, "查询未完成 AI 任务失败", err)
	}
	n := 0
	for _, t := range rows {
		if err := s.tasks.Update(ctx, t.ID, map[string]any{"status": model.AiTaskStatusPending}); err != nil {
			slog.Warn("重排 AI 任务失败", "taskID", t.ID, "err", err)
			continue
		}
		go s.process(context.Background(), t.ID)
		n++
	}
	return n, nil
}

// decodeSegments 解析 transcripts.segments JSON 为 agent.Segment（失败则退化为纯文本）。
func decodeSegments(raw *string) []agent.Segment {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var items []dto.TranscriptSegment
	if err := json.Unmarshal([]byte(*raw), &items); err != nil {
		return nil
	}
	out := make([]agent.Segment, 0, len(items))
	for _, it := range items {
		out = append(out, agent.Segment{Start: it.Start, End: it.End, Text: it.Text})
	}
	return out
}

func toAiTaskDTO(t *model.AiTask) *dto.AiTaskDTO {
	return &dto.AiTaskDTO{
		ID:           t.ID,
		Kind:         t.Kind,
		RefType:      t.RefType,
		RefID:        t.RefID,
		Status:       t.Status,
		Attempts:     int(t.Attempts),
		ErrorMessage: t.ErrorMessage,
		CreatedAt:    t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    t.UpdatedAt.Format(time.RFC3339),
	}
}
