// Package scoring 实现课堂评价的评分计算（Sprint2-3 开发计划 §2）：
// 维度权重、单次总分、课程级/教师级聚合，全部为无副作用的纯函数。
//
// 口径（变更需先改开发计划 §2，契约先行）：
//   - 维度分：dim = (v−1)/4×100，v∈{1..5}，值域完整覆盖 [0,100]；
//   - 单侧总分：Σ w×dim，该侧缺维度时权重在其余维度上重新归一化；
//   - 维度展示分：先按侧求均、再按维度 α 融合（单侧缺失取另一侧）；
//   - 综合分：取「综合均值」口径——各场次综合分（场次内逐维度融合后加权）的算术平均，
//     因此教师级综合分恒等于其各次课综合分的算术平均（§2.5.2 恒等式，恒定成立）；
//     默认数据双侧对齐时，综合均值与「维度融合后加权」逐位相同；
//   - frontier 为观测项（权重 0），不计入加权总分。
package scoring

import "fmt"

// 维度固定顺序（与 Weights、DimensionScores 的字段一一对应；维度冻结，不得随意增删）。
const dimCount = 5

// DimensionScoreKey 是维度标识（evaluations 表列名后缀 / 前端渲染 key）。
const (
	KeyObjective    = "objective"
	KeyContent      = "content"
	KeyInteraction  = "interaction"
	KeyOrganization = "organization"
	KeyFrontier     = "frontier"
)

var catalog = [dimCount]struct{ Key, Name string }{
	{KeyObjective, "教学目标与内容准确性"},
	{KeyContent, "内容质量与深度"},
	{KeyInteraction, "学生互动与参与"},
	{KeyOrganization, "课堂组织与节奏"},
	{KeyFrontier, "前沿与交叉学科"},
}

// Weights 是 5 个维度的生效权重；非零项合计必须为 1.00（服务启动时校验）。
type Weights struct {
	Objective    float64
	Content      float64
	Interaction  float64
	Organization float64
	Frontier     float64
}

// Validate 校验权重合法性：均为非负、合计为 1.00（±1e-9）。
func (w Weights) Validate() error {
	sum := 0.0
	for d := range catalog {
		wt := w.at(d)
		if wt < 0 {
			return fmt.Errorf("scoring: 维度 %s 权重为负（%v）", catalog[d].Key, wt)
		}
		sum += wt
	}
	if sum < 1-1e-9 || sum > 1+1e-9 {
		return fmt.Errorf("scoring: 生效权重合计必须为 1.00，实际 %.4f", sum)
	}
	return nil
}

// DimensionScores 是单次评价的 5 个维度原始分（1-5 标度），nil 表示该评分源无法评价该维度。
// 同场次多个同类评价人时，入库前由 SQL AVG 折算成均分，故为浮点。
type DimensionScores struct {
	Objective    *float64
	Content      *float64
	Interaction  *float64
	Organization *float64
	Frontier     *float64
}

// NewDimensionScores 由 5 个 1-5 整数维度分构造（督导评分提交路径使用）。
func NewDimensionScores(objective, content, interaction, organization, frontier *int) *DimensionScores {
	return &DimensionScores{
		Objective:    intToPtr(objective),
		Content:      intToPtr(content),
		Interaction:  intToPtr(interaction),
		Organization: intToPtr(organization),
		Frontier:     intToPtr(frontier),
	}
}

// SessionScore 是一个场次的双侧评价，nil 表示该侧无评价。
type SessionScore struct {
	SessionID  uint64
	CourseID   uint64
	Supervisor *DimensionScores
	Agent      *DimensionScores
}

// Sample 是样本量与覆盖度计数（§2.5.3：必须暴露 alignedCount，让使用者知道数字代表几次课）。
// EvaluatedCount 是「已评价场次」数（至少有一侧评价），样本充足性以此为准（§2.5.5）。
type Sample struct {
	SessionCount    int
	EvaluatedCount  int
	SupervisorCount int
	AgentCount      int
	AlignedCount    int
}

// DimensionSummary 是聚合后的单个维度结果。
type DimensionSummary struct {
	Key             string
	Name            string
	Weight          float64
	IsObservation   bool
	Score           *float64 // 双侧融合后的维度分（0-100）；双侧均缺时为 nil
	SupervisorScore *float64 // 督导侧维度均分（0-100）；无督导数据时为 nil
	AgentScore      *float64 // 智能体侧维度均分（0-100）；无智能体数据时为 nil
}

// Summary 是聚合结果：课程级与教师级共用同一结构。
type Summary struct {
	Composite  *float64 // 综合分 = 各场次综合分的算术平均（综合均值口径）；无任何评价时为 nil
	Supervisor *float64 // 督导侧单侧分（缺维度时权重重新归一化）
	Agent      *float64 // 智能体侧单侧分
	Dimensions []DimensionSummary
	Sample     Sample
	Flags      []string // no_data / sup_only / ai_only / disjoint
}

// Aggregate 是课程级与教师级共用的唯一聚合入口。
// 传教师的全部场次即得教师级，传某课程的场次即得课程级；两次聚合口径逐位一致。
// alpha 是综合分中督导评分的权重 α（智能体为 1−α）。
func Aggregate(items []SessionScore, weights Weights, alpha float64) Summary {
	if alpha < 0 {
		alpha = 0
	} else if alpha > 1 {
		alpha = 1
	}

	s := Summary{Dimensions: newDimensionSummaries(weights)}

	var supSum, aiSum [dimCount]float64
	var supN, aiN [dimCount]int
	for _, item := range items {
		if item.Supervisor != nil {
			s.Sample.SupervisorCount++
		}
		if item.Agent != nil {
			s.Sample.AgentCount++
		}
		if item.Supervisor != nil || item.Agent != nil {
			s.Sample.EvaluatedCount++
		}
		if item.Supervisor != nil && item.Agent != nil {
			s.Sample.AlignedCount++
		}
		for d := 0; d < dimCount; d++ {
			if v := item.Supervisor.at(d); v != nil {
				supSum[d] += *v
				supN[d]++
			}
			if v := item.Agent.at(d); v != nil {
				aiSum[d] += *v
				aiN[d]++
			}
		}
	}
	s.Sample.SessionCount = len(items)

	var supMeans, aiMeans [dimCount]*float64
	for d := 0; d < dimCount; d++ {
		dim := &s.Dimensions[d]
		if supN[d] > 0 {
			v := DimScore(supSum[d] / float64(supN[d]))
			dim.SupervisorScore = &v
			supMeans[d] = &v
		}
		if aiN[d] > 0 {
			v := DimScore(aiSum[d] / float64(aiN[d]))
			dim.AgentScore = &v
			aiMeans[d] = &v
		}
		// 逐维度融合：双侧齐全取 α 加权；单侧缺失直接取另一侧；都缺为 nil。
		switch {
		case dim.SupervisorScore != nil && dim.AgentScore != nil:
			v := alpha**dim.SupervisorScore + (1-alpha)**dim.AgentScore
			dim.Score = &v
		case dim.SupervisorScore != nil:
			dim.Score = dim.SupervisorScore
		case dim.AgentScore != nil:
			dim.Score = dim.AgentScore
		}
	}

	s.Supervisor = weightedSide(weights, &supMeans)
	s.Agent = weightedSide(weights, &aiMeans)
	// 综合均值口径：综合分 = 各场次综合分的算术平均（§2.5.2 恒等式恒定成立）。
	// 默认数据双侧对齐时，它与「维度展示分加权求和」逐位一致。
	s.Composite = meanSessionComposite(items, weights, alpha)
	s.Flags = missingFlags(s.Sample)
	return s
}

// meanSessionComposite 计算各场次综合分的算术平均（综合均值口径）。
// 仅统计有评价、可算出综合分的场次（SessionComposite 为 nil 的场次不计入分母），
// 避免未评价场次被当作 0 分拉低综合分（§2.5.5 硬约定 8）。
func meanSessionComposite(items []SessionScore, weights Weights, alpha float64) *float64 {
	var sum float64
	var n int
	for _, item := range items {
		if c := SessionComposite(item, weights, alpha); c != nil {
			sum += *c
			n++
		}
	}
	if n == 0 {
		return nil
	}
	v := sum / float64(n)
	return &v
}

// SideTotal 计算单次评价某一侧的总分：Σ w×dim，缺维度时权重在其余维度上重新归一化。
// 仅用于落库 total_score 与列表快速展示，不是聚合的输入。
func SideTotal(ds *DimensionScores, weights Weights) *float64 {
	if ds == nil {
		return nil
	}
	var vals [dimCount]*float64
	for d := 0; d < dimCount; d++ {
		if v := ds.at(d); v != nil {
			converted := DimScore(*v)
			vals[d] = &converted
		}
	}
	return weightedSide(weights, &vals)
}

// SessionComposite 计算单个场次的综合分（维度层融合后加权求和）。
// 教师级综合分恒等于各场次该值的算术平均（开发计划 §2.5.2 恒等式，验收对账用）。
func SessionComposite(item SessionScore, weights Weights, alpha float64) *float64 {
	if alpha < 0 {
		alpha = 0
	} else if alpha > 1 {
		alpha = 1
	}
	var fused [dimCount]*float64
	for d := 0; d < dimCount; d++ {
		var sup, ai *float64
		if v := item.Supervisor.at(d); v != nil {
			converted := DimScore(*v)
			sup = &converted
		}
		if v := item.Agent.at(d); v != nil {
			converted := DimScore(*v)
			ai = &converted
		}
		switch {
		case sup != nil && ai != nil:
			v := alpha**sup + (1-alpha)**ai
			fused[d] = &v
		case sup != nil:
			fused[d] = sup
		case ai != nil:
			fused[d] = ai
		}
	}
	dims := make([]DimensionSummary, dimCount)
	for d := range dims {
		dims[d].Score = fused[d]
	}
	return composite(weights, dims)
}

// DimScore 把 1-5 标度的原始分换算为 0-100 的维度分：dim = (v−1)/4×100。
// 取 (v−1)/4 而非 v/5，避免最低分永远是 20 分的分数压缩。
func DimScore(v float64) float64 {
	return (v - 1) / 4 * 100
}

// weightedSide 对某一侧的维度分加权求和；权重在缺维度时重新归一化（den = Σ 该侧可用权重）。
func weightedSide(weights Weights, vals *[dimCount]*float64) *float64 {
	num, den := 0.0, 0.0
	for d := 0; d < dimCount; d++ {
		if vals[d] != nil {
			num += weights.at(d) * *vals[d]
			den += weights.at(d)
		}
	}
	if den <= 0 {
		return nil
	}
	v := num / den
	return &v
}

// composite 由传入的维度分计算加权和 Σ w×dim，对可评价维度直接加权、不做重新归一化。
// 当前仅用于 SessionComposite（单场次综合分）；单侧缺失已在维度层回退补齐。
func composite(weights Weights, dims []DimensionSummary) *float64 {
	total, has := 0.0, false
	for d := range dims {
		if dims[d].Score != nil {
			total += weights.at(d) * *dims[d].Score
			has = true
		}
	}
	if !has {
		return nil
	}
	return &total
}

// missingFlags 按缺失矩阵（§2.5.5）返回状态标记；双侧齐全且场次对齐时返回空。
func missingFlags(sample Sample) []string {
	switch {
	case sample.SupervisorCount == 0 && sample.AgentCount == 0:
		return []string{"no_data"}
	case sample.AgentCount == 0:
		return []string{"sup_only"}
	case sample.SupervisorCount == 0:
		return []string{"ai_only"}
	case sample.AlignedCount == 0:
		return []string{"disjoint"}
	default:
		return []string{}
	}
}

func newDimensionSummaries(weights Weights) []DimensionSummary {
	dims := make([]DimensionSummary, dimCount)
	for d := range dims {
		dims[d] = DimensionSummary{
			Key:           catalog[d].Key,
			Name:          catalog[d].Name,
			Weight:        weights.at(d),
			IsObservation: weights.at(d) == 0,
		}
	}
	return dims
}

func (w Weights) at(d int) float64 {
	switch catalog[d].Key {
	case KeyObjective:
		return w.Objective
	case KeyContent:
		return w.Content
	case KeyInteraction:
		return w.Interaction
	case KeyOrganization:
		return w.Organization
	default:
		return w.Frontier
	}
}

func (ds *DimensionScores) at(d int) *float64 {
	if ds == nil {
		return nil
	}
	switch catalog[d].Key {
	case KeyObjective:
		return ds.Objective
	case KeyContent:
		return ds.Content
	case KeyInteraction:
		return ds.Interaction
	case KeyOrganization:
		return ds.Organization
	default:
		return ds.Frontier
	}
}

func intToPtr(v *int) *float64 {
	if v == nil {
		return nil
	}
	f := float64(*v)
	return &f
}
