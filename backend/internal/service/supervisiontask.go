package service

import (
	"context"
	"encoding/json"
	"sort"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// sampleBatchSize 是抽样分派的批大小（B2：每满 5 条 AI 终评抽 1 条）。
const sampleBatchSize = 5

// SupervisionTaskService 负责督导抽样任务队列（§4.3 / R1.2）。
//
// 抽样规则（B2/T1，**不建计数器表**）：该教师 AI 终评每满 5 条抽 1 条；
// 候选 = 该批最近 5 条；有高风险标签优先取标签项，否则取 AI 综合分最低 1 条。
type SupervisionTaskService interface {
	Queue(ctx context.Context, q dto.SupervisorTaskListQuery) (*dto.PageResult[dto.SupervisorTaskDTO], error)
}

type supervisionTaskService struct {
	evals repository.EvaluationRepository
	cfg   config.EvaluationConfig
}

// NewSupervisionTaskService 构造抽样任务服务。
func NewSupervisionTaskService(evals repository.EvaluationRepository, cfg config.EvaluationConfig) SupervisionTaskService {
	return &supervisionTaskService{evals: evals, cfg: cfg}
}

func (s *supervisionTaskService) Queue(ctx context.Context, q dto.SupervisorTaskListQuery) (*dto.PageResult[dto.SupervisorTaskDTO], error) {
	sem, err := resolveSemester(q.Semester)
	if err != nil {
		return nil, err
	}
	rows, err := s.evals.ListAgentSessions(ctx, sem, s.cfg.FormulaVersion)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询 AI 终评场次失败", err)
	}

	// 按教师分组：rows 已按 teacher_id ASC, session_date DESC, id DESC 排序。
	items := make([]dto.SupervisorTaskDTO, 0)
	var group []repository.AgentSessionRow
	flush := func() {
		items = append(items, sampleFromBatchGroup(group, s.cfg.MinSampleSize)...)
		group = group[:0]
	}
	for _, row := range rows {
		if len(group) > 0 && group[0].TeacherID != row.TeacherID {
			flush()
		}
		group = append(group, row)
	}
	flush()

	if q.Pending {
		pending := make([]dto.SupervisorTaskDTO, 0, len(items))
		for _, it := range items {
			if !it.Reviewed {
				pending = append(pending, it)
			}
		}
		items = pending
	}

	// 待人工复评在前，其次按课次倒序（稳定）。
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Reviewed != items[j].Reviewed {
			return !items[i].Reviewed
		}
		return items[i].SessionDate > items[j].SessionDate
	})

	page, pageSize := normalizePage(q.Page, q.PageSize)
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return dto.NewPageResult(items[start:end], int64(total), page, pageSize), nil
}

// sampleFromBatchGroup 对单个教师的 AI 终评序列（倒序）做分批抽样，返回每批 1 条样本。
// seq 为按课次倒序排列的该教师全部 AI 终评场次。
func sampleFromBatchGroup(seq []repository.AgentSessionRow, _ int) []dto.SupervisorTaskDTO {
	n := len(seq)
	batches := n / sampleBatchSize
	out := make([]dto.SupervisorTaskDTO, 0, batches)
	for k := 1; k <= batches; k++ {
		// 第 k 批 = 时间上第 k 组 5 条（倒序序列中的 [n-5k, n-5(k-1)) ）。
		lo := n - sampleBatchSize*k
		hi := n - sampleBatchSize*(k-1)
		if lo < 0 || hi > n || lo >= hi {
			continue
		}
		out = append(out, pickSample(seq[lo:hi], k))
	}
	return out
}

// pickSample 在一批候选中选出 1 条：有风险标签优先取标签项，否则取 AI 总分最低。
func pickSample(batch []repository.AgentSessionRow, batchNo int) dto.SupervisorTaskDTO {
	chosen := batch[0]
	reason := "lowest"
	var chosenRisk []string
	for _, row := range batch {
		if risks := riskTagsOf(row.Evidence); len(risks) > 0 {
			chosen = row
			reason = "risk"
			chosenRisk = risks
			break
		}
	}
	if reason == "lowest" {
		best := chosen
		for _, row := range batch {
			if row.AITotal != nil && (best.AITotal == nil || *row.AITotal < *best.AITotal) {
				best = row
			}
		}
		chosen = best
	}
	return dto.SupervisorTaskDTO{
		SessionID:         chosen.SessionID,
		CourseID:          chosen.CourseID,
		CourseCode:        chosen.CourseCode,
		CourseName:        chosen.CourseName,
		TeacherID:         chosen.TeacherID,
		TeacherName:       chosen.TeacherName,
		SessionDate:       chosen.SessionDate.Format("2006-01-02"),
		Period:            chosen.Period,
		Topic:             chosen.Topic,
		Batch:             batchNo,
		AITotal:           round2Ptr(chosen.AITotal),
		Reason:            reason,
		RiskTags:          chosenRisk,
		Reviewed:          chosen.SupervisorCnt > 0,
		SupervisorComment: chosen.SupervisorNote,
	}
}

// riskTagsOf 从 evaluations.evidence JSON 中提取 riskTags（无则空）。
func riskTagsOf(evidence *string) []string {
	if evidence == nil || *evidence == "" {
		return nil
	}
	var ev struct {
		RiskTags []string `json:"riskTags"`
	}
	if err := json.Unmarshal([]byte(*evidence), &ev); err != nil {
		return nil
	}
	return ev.RiskTags
}
