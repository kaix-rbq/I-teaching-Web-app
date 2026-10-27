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

const (
	// aiTaskErrMaxLen 对齐 ai_tasks.error_message 的 VARCHAR(255)。
	aiTaskErrMaxLen = 255
	// maxEvalRunes 是送入 S1 的转写长度上限（按 rune 计，不是字节）。
	// 超长转写会显著推高时延与 token 成本，截断后正常评析；证据定位失败按现有规则降级。
	maxEvalRunes = 80000
)

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
	retriever  agent.Retriever
	agentCfg   config.AgentConfig
	evalCfg    config.EvaluationConfig
	reporter   ReportEnqueuer
	sem        chan struct{}
}

// NewAiTaskService 构造 AI 任务服务；analyzer 为 nil 表示 AI 引擎未配置。
func NewAiTaskService(
	tasks repository.AiTaskRepository,
	sessions repository.SessionRepository,
	evals repository.EvaluationRepository,
	recordings repository.RecordingRepository,
	analyzer agent.SessionAnalyzer,
	retriever agent.Retriever,
	agentCfg config.AgentConfig,
	evalCfg config.EvaluationConfig,
	reporters ...ReportEnqueuer,
) AiTaskService {
	svc := &aiTaskService{
		tasks:      tasks,
		sessions:   sessions,
		evals:      evals,
		recordings: recordings,
		analyzer:   analyzer,
		retriever:  retriever,
		agentCfg:   agentCfg,
		evalCfg:    evalCfg,
		sem:        make(chan struct{}, agentCfg.Concurrency()),
	}
	if len(reporters) > 0 {
		svc.reporter = reporters[0]
	}
	return svc
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
			"status": model.AiTaskStatusPending, "attempts": 0, "error_message": "",
			"started_at": nil, "finished_at": nil,
		}); err != nil {
			return nil, false, errcode.Wrap(errcode.Internal, "重置 AI 任务失败", err)
		}
		existing.Status = model.AiTaskStatusPending
		existing.Attempts = 0
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
	// teacher_report 由 ReportService 处理，不能被 S1 的 process 误当作 session 任务。
	if task.Kind != model.AiTaskKindSessionEval {
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

	// 自动重试上限（P1.5）：必须在置 running 前判断，达到上限直接 failed，不调用模型。
	// 人工 Retry 会先把 attempts 复位为 0，从而允许再次执行。
	if task.Attempts >= model.AiTaskMaxAttempts {
		fail("AI 评析重试次数已达上限")
		slog.Warn("AI 评析达到重试上限", "taskID", taskID, "attempts", task.Attempts,
			"limit", model.AiTaskMaxAttempts)
		return
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

	var knowledge []agent.RetrievedChunk
	if s.retriever != nil {
		q := strings.TrimSpace(session.CourseName + " " + session.Topic + " 课堂教学质量 五维评分锚点")
		hits, rerr := s.retriever.Retrieve(ctx, q, agent.ChunkFilter{
			Collections: []string{"kb-rubric"},
			CourseID:    &session.CourseID,
		})
		if rerr != nil {
			slog.Warn("知识库检索失败，S1 降级为无 KB", "sessionID", session.ID, "err", rerr)
		} else {
			knowledge = hits
		}
	}

	segments := decodeSegments(tr.Segments)
	transcript, segments, truncated, originalRunes := truncateEvalTranscript(tr.Content, segments)
	if truncated {
		slog.Warn("transcript truncated",
			"sessionID", session.ID, "originalRunes", originalRunes, "limit", maxEvalRunes)
	}

	result, err := s.analyzer.AnalyzeSession(ctx, agent.EvalInput{
		SessionID:   session.ID,
		CourseName:  session.CourseName,
		TeacherName: session.TeacherName,
		SessionDate: session.SessionDate.Format("2006-01-02"),
		Period:      session.Period,
		Topic:       session.Topic,
		Transcript:  transcript,
		Segments:    segments,
		Knowledge:   knowledge,
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
	// S1 成功后再触发 S2，不等待报告完成；Enqueue 失败只记日志，绝不回滚 S1。
	if s.reporter != nil && s.agentCfg.ReportEnabled {
		if err := s.reporter.Enqueue(session.TeacherID, session.Semester, false); err != nil {
			slog.Warn("入队教师报告任务失败", "sessionID", session.ID, "teacherID", session.TeacherID, "err", err)
		}
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
		"status": model.AiTaskStatusPending, "attempts": 0, "error_message": "",
		"started_at": nil, "finished_at": nil,
	}); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "重置 AI 任务失败", err)
	}
	go s.process(context.Background(), taskID)
	task.Status = model.AiTaskStatusPending
	task.Attempts = 0
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
		// teacher_report 的后台执行由 ReportService 负责，S1 的重启恢复不得误处理。
		if t.Kind != model.AiTaskKindSessionEval {
			continue
		}
		if err := s.tasks.Update(ctx, t.ID, map[string]any{"status": model.AiTaskStatusPending}); err != nil {
			slog.Warn("重排 AI 任务失败", "taskID", t.ID, "err", err)
			continue
		}
		go s.process(context.Background(), t.ID)
		n++
	}
	return n, nil
}

// truncateEvalTranscript 把送入 S1 的转写限制在 maxEvalRunes（按 rune 计）。
//
//   - 有 segments：按累计 rune 截断 segments，并用保留的 segments 文本拼接 Transcript，
//     确保 analyzer 无论读 Transcript 还是读 Segments，看到的内容都不超过上限；
//   - 无 segments：直接截断 Content；
//   - truncated 表示原始内容超过上限；originalRunes 用于日志定位数据规模。
//
// 截断导致的引用定位失败由 agent.locateQuotes 按现有规则丢弃（不放松证据校验）。
func truncateEvalTranscript(content string, segments []agent.Segment) (string, []agent.Segment, bool, int) {
	contentRunes := []rune(content)
	originalRunes := len(contentRunes)

	if len(segments) > 0 {
		segmentRunes := 0
		for _, seg := range segments {
			segmentRunes += len([]rune(seg.Text))
		}
		if segmentRunes > originalRunes {
			originalRunes = segmentRunes
		}

		remaining := maxEvalRunes
		kept := make([]agent.Segment, 0, len(segments))
		var b strings.Builder
		for _, seg := range segments {
			if remaining <= 0 {
				break
			}
			runes := []rune(seg.Text)
			if len(runes) == 0 {
				continue
			}
			if len(runes) > remaining {
				runes = runes[:remaining]
			}
			seg.Text = string(runes)
			kept = append(kept, seg)
			b.WriteString(seg.Text)
			remaining -= len(runes)
		}
		if len(kept) > 0 {
			return b.String(), kept, originalRunes > maxEvalRunes, originalRunes
		}
		// segments 全为空串时退化为纯文本截断。
		segments = nil
	}

	if len(contentRunes) <= maxEvalRunes {
		return content, segments, originalRunes > maxEvalRunes, originalRunes
	}
	return string(contentRunes[:maxEvalRunes]), segments, true, originalRunes
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
