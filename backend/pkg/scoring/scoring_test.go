package scoring

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultWeights 是 §2.2 的生效权重（frontier 为观测项，权重 0）。
func defaultWeights() Weights {
	return Weights{Objective: 0.30, Content: 0.30, Interaction: 0.20, Organization: 0.20, Frontier: 0}
}

// round2 与 service 层展示口径一致：保留两位小数后再比对。
func round2(v float64) float64 { return math.Round(v*100) / 100 }

// f 构造 *float64。
func f(v float64) *float64 { return &v }

// i 构造 *int。
func i(v int) *int { return &v }

func TestDimScore(t *testing.T) {
	cases := []struct {
		name string
		raw  float64
		want float64
	}{
		{name: "最低分 1 分换算为 0", raw: 1, want: 0},
		{name: "中位 3 分换算为 50", raw: 3, want: 50},
		{name: "最高 5 分换算为 100", raw: 5, want: 100},
		{name: "多督导均分 4.5 换算为 87.5", raw: 4.5, want: 87.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, DimScore(tc.raw))
		})
	}
}

func TestWeightsValidate(t *testing.T) {
	cases := []struct {
		name    string
		weights Weights
		wantErr bool
	}{
		{name: "§2.2 生效权重合法", weights: defaultWeights(), wantErr: false},
		{name: "合计不为 1 非法", weights: Weights{Objective: 0.5, Content: 0.3}, wantErr: true},
		{name: "负权重非法", weights: Weights{Objective: -0.1, Content: 0.8, Interaction: 0.1, Organization: 0.2}, wantErr: true},
		{name: "frontier 计入权重也合法（默认配置为 0 属业务约定，非校验规则）", weights: Weights{Objective: 0.2, Content: 0.2, Interaction: 0.2, Organization: 0.2, Frontier: 0.2}, wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.weights.Validate()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestSideTotalWorkbook 覆盖 §2.4 验算例：单侧总分 = Σ w×dim，frontier 不参与。
func TestSideTotalWorkbook(t *testing.T) {
	ds := NewDimensionScores(i(5), i(4), i(3), i(5), i(2))
	got := SideTotal(ds, defaultWeights())
	require.NotNil(t, got)
	assert.Equal(t, 82.50, round2(*got))
}

// TestSideTotalSeed 对照 §3.4 对账基准的 6 条种子评价（含智能体缺 objective 时的权重再归一化）。
func TestSideTotalSeed(t *testing.T) {
	cases := []struct {
		name string
		ds   *DimensionScores
		want float64
	}{
		{name: "第 1 次课督导 52.50", ds: NewDimensionScores(i(4), i(3), i(2), i(3), i(2)), want: 52.50},
		{name: "第 2 次课督导 70.00", ds: NewDimensionScores(i(4), i(4), i(3), i(4), i(3)), want: 70.00},
		{name: "第 3 次课督导 82.50", ds: NewDimensionScores(i(5), i(4), i(4), i(4), i(3)), want: 82.50},
		{name: "第 1 次课智能体 60.71", ds: NewDimensionScores(nil, i(4), i(3), i(3), i(3)), want: 60.71},
		{name: "第 2 次课智能体 67.86", ds: NewDimensionScores(nil, i(4), i(3), i(4), i(3)), want: 67.86},
		{name: "第 3 次课智能体 75.00", ds: NewDimensionScores(nil, i(4), i(4), i(4), i(4)), want: 75.00},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SideTotal(tc.ds, defaultWeights())
			require.NotNil(t, got)
			assert.Equal(t, tc.want, round2(*got))
		})
	}
}

func TestSideTotalEdge(t *testing.T) {
	assert.Nil(t, SideTotal(nil, defaultWeights()), "无评价侧应返回 nil")
	// 仅剩观测维度：可用权重合计为 0，无法归一化。
	onlyFrontier := NewDimensionScores(nil, nil, nil, nil, i(4))
	assert.Nil(t, SideTotal(onlyFrontier, defaultWeights()))
	// 缺 objective 时权重在其余维度上重新归一化：(0.3×75+0.2×50+0.2×100)/0.7 = 75.00。
	noObjective := NewDimensionScores(nil, i(4), i(3), i(5), i(2))
	got := SideTotal(noObjective, defaultWeights())
	require.NotNil(t, got)
	assert.Equal(t, 75.00, round2(*got))
}

// seedSessions 构造 §3.4 种子数据（课程 1 · 教师 2 · 三次课 · 双侧评价）。
func seedSessions() []SessionScore {
	return []SessionScore{
		{
			SessionID: 1, CourseID: 1,
			Supervisor: NewDimensionScores(i(4), i(3), i(2), i(3), i(2)),
			Agent:      NewDimensionScores(nil, i(4), i(3), i(3), i(3)),
		},
		{
			SessionID: 2, CourseID: 1,
			Supervisor: NewDimensionScores(i(4), i(4), i(3), i(4), i(3)),
			Agent:      NewDimensionScores(nil, i(4), i(3), i(4), i(3)),
		},
		{
			SessionID: 3, CourseID: 1,
			Supervisor: NewDimensionScores(i(5), i(4), i(4), i(4), i(3)),
			Agent:      NewDimensionScores(nil, i(4), i(4), i(4), i(4)),
		},
	}
}

// TestSessionCompositeSeed 对照 §3.4 对账基准：三次课综合分 58.75 / 70.00 / 82.50。
func TestSessionCompositeSeed(t *testing.T) {
	cases := []struct {
		idx  int
		want float64
	}{{0, 58.75}, {1, 70.00}, {2, 82.50}}
	for _, tc := range cases {
		got := SessionComposite(seedSessions()[tc.idx], defaultWeights(), 0.5)
		require.NotNil(t, got)
		assert.Equal(t, tc.want, round2(*got), "session %d", tc.idx+1)
	}
}

// TestAggregateSeed 对照 §4.2 响应示例：教师级综合分 70.42、双侧分、逐维度分与样本计数。
func TestAggregateSeed(t *testing.T) {
	s := Aggregate(seedSessions(), defaultWeights(), 0.5)

	require.NotNil(t, s.Composite)
	assert.Equal(t, 70.42, round2(*s.Composite), "教师级综合分")
	require.NotNil(t, s.Supervisor)
	assert.Equal(t, 68.33, round2(*s.Supervisor), "督导侧")
	require.NotNil(t, s.Agent)
	assert.Equal(t, 67.86, round2(*s.Agent), "智能体侧（缺 objective 时再归一化）")

	assert.Equal(t, Sample{SessionCount: 3, EvaluatedCount: 3, SupervisorCount: 3, AgentCount: 3, AlignedCount: 3}, s.Sample)
	assert.Empty(t, s.Flags)

	wantDims := []struct {
		key            string
		score, sup, ai float64
		weight         float64
		isObservation  bool
		supNil, aiNil  bool
	}{
		{key: KeyObjective, score: 83.33, sup: 83.33, weight: 0.30},
		{key: KeyContent, score: 70.83, sup: 66.67, ai: 75.00, weight: 0.30},
		{key: KeyInteraction, score: 54.17, sup: 50.00, ai: 58.33, weight: 0.20},
		{key: KeyOrganization, score: 66.67, sup: 66.67, ai: 66.67, weight: 0.20},
		{key: KeyFrontier, score: 50.00, sup: 41.67, ai: 58.33, weight: 0, isObservation: true},
	}
	require.Len(t, s.Dimensions, 5)
	for idx, want := range wantDims {
		dim := s.Dimensions[idx]
		assert.Equal(t, want.key, dim.Key)
		assert.Equal(t, want.weight, dim.Weight)
		assert.Equal(t, want.isObservation, dim.IsObservation)
		require.NotNil(t, dim.Score)
		assert.Equal(t, want.score, round2(*dim.Score), "维度 %s 融合分", dim.Key)
		switch dim.Key {
		case KeyObjective:
			assert.Nil(t, dim.AgentScore, "智能体无法评价 objective")
			require.NotNil(t, dim.SupervisorScore)
			assert.Equal(t, want.sup, round2(*dim.SupervisorScore))
		default:
			require.NotNil(t, dim.SupervisorScore)
			assert.Equal(t, want.sup, round2(*dim.SupervisorScore))
			require.NotNil(t, dim.AgentScore)
			assert.Equal(t, want.ai, round2(*dim.AgentScore))
		}
	}
}

// TestAggregateIdentity 验证 §2.5.2 恒等式：教师级综合分 ≡ 各次课综合分的算术平均。
func TestAggregateIdentity(t *testing.T) {
	items := seedSessions()
	s := Aggregate(items, defaultWeights(), 0.5)
	require.NotNil(t, s.Composite)

	var sum float64
	for _, item := range items {
		c := SessionComposite(item, defaultWeights(), 0.5)
		require.NotNil(t, c)
		sum += *c
	}
	mean := sum / float64(len(items))
	assert.InDelta(t, *s.Composite, mean, 1e-9)
	assert.Equal(t, 70.42, round2(mean))
}

// TestAggregateMissingMatrix 覆盖 §2.5.5 缺失矩阵全部 5 行。
// 综合分口径（§2.5.2）：Σ w×dim、缺维度不计、不重新归一化——否则教师级恒等式被破坏；
// 单侧展示分（Summary.Supervisor/Agent）才做缺维度归一化，两者不可混用。
func TestAggregateMissingMatrix(t *testing.T) {
	supOnly := NewDimensionScores(i(4), i(4), i(4), i(4), i(4))
	aiOnly := NewDimensionScores(nil, i(4), i(3), i(3), i(3))

	cases := []struct {
		name          string
		items         []SessionScore
		wantComposite *float64
		wantAgent     *float64
		wantFlags     []string
	}{
		{
			name:      "双侧均无 → no_data",
			items:     []SessionScore{{SessionID: 1}, {SessionID: 2}},
			wantFlags: []string{"no_data"},
		},
		{
			name:          "仅督导 → sup_only",
			items:         []SessionScore{{SessionID: 1, Supervisor: supOnly}},
			wantComposite: f(75.00),
			wantFlags:     []string{"sup_only"},
		},
		{
			name:          "仅智能体 → ai_only",
			items:         []SessionScore{{SessionID: 1, Agent: aiOnly}},
			wantComposite: f(42.50),
			wantAgent:     f(60.71),
			wantFlags:     []string{"ai_only"},
		},
		{
			name: "双侧对齐 → 无标记",
			items: []SessionScore{
				{SessionID: 1, Supervisor: supOnly, Agent: aiOnly},
				{SessionID: 2, Supervisor: supOnly, Agent: aiOnly},
			},
			wantComposite: f(70.00),
			wantAgent:     f(60.71),
			wantFlags:     []string{},
		},
		{
			name: "双侧都有但场次不对齐 → disjoint（综合均值口径：两场次综合分 75/42.5 的均值）",
			items: []SessionScore{
				{SessionID: 1, Supervisor: supOnly},
				{SessionID: 2, Agent: aiOnly},
			},
			wantComposite: f(58.75),
			wantAgent:     f(60.71),
			wantFlags:     []string{"disjoint"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Aggregate(tc.items, defaultWeights(), 0.5)
			assert.Equal(t, tc.wantFlags, s.Flags)
			if tc.wantComposite == nil {
				assert.Nil(t, s.Composite)
				return
			}
			require.NotNil(t, s.Composite)
			assert.Equal(t, round2(*tc.wantComposite), round2(*s.Composite))
			if tc.wantAgent != nil {
				require.NotNil(t, s.Agent)
				assert.Equal(t, round2(*tc.wantAgent), round2(*s.Agent))
			}
		})
	}
}

// TestAggregateIdentityNonAligned 锁定 F3 修复：非全对齐数据下，
// 教师级综合分仍恒等于各场次综合分的算术平均（综合均值口径）。
func TestAggregateIdentityNonAligned(t *testing.T) {
	w := defaultWeights()
	items := []SessionScore{
		{SessionID: 1,
			Supervisor: NewDimensionScores(i(5), nil, nil, nil, nil),
			Agent:      NewDimensionScores(i(5), nil, nil, nil, nil)},
		{SessionID: 2,
			Supervisor: NewDimensionScores(i(1), nil, nil, nil, nil)},
	}
	s := Aggregate(items, w, 0.5)
	require.NotNil(t, s.Composite)
	var sum float64
	for _, it := range items {
		c := SessionComposite(it, w, 0.5)
		require.NotNil(t, c)
		sum += *c
	}
	assert.InDelta(t, *s.Composite, sum/float64(len(items)), 1e-9)
	assert.Equal(t, 15.00, round2(*s.Composite), "旧口径会得到 22.5，新口径必须是场次综合分的均值 15")
	assert.Equal(t, 2, s.Sample.EvaluatedCount)
}

// TestAggregateEvaluatedCount 覆盖 F5：已评价场次只统计至少有一侧评价的场次。
func TestAggregateEvaluatedCount(t *testing.T) {
	items := []SessionScore{
		{SessionID: 1},
		{SessionID: 2},
		{SessionID: 3, Supervisor: NewDimensionScores(i(4), i(4), i(4), i(4), i(4))},
	}
	s := Aggregate(items, defaultWeights(), 0.5)
	assert.Equal(t, 3, s.Sample.SessionCount, "总会话数含未评价场次")
	assert.Equal(t, 1, s.Sample.EvaluatedCount, "已评价场次只计有评价的")
	assert.Contains(t, s.Flags, "sup_only")
}

// TestAggregateEmpty 无任何场次时综合分必须为 nil，不得按 0 分处理（§2.5.5 硬约定 8）。
func TestAggregateEmpty(t *testing.T) {
	s := Aggregate(nil, defaultWeights(), 0.5)
	assert.Nil(t, s.Composite)
	assert.Nil(t, s.Supervisor)
	assert.Nil(t, s.Agent)
	assert.Equal(t, []string{"no_data"}, s.Flags)
	assert.Equal(t, Sample{}, s.Sample)
	assert.Len(t, s.Dimensions, 5)
}

// TestAggregateAlphaClamp α 越界时收敛到 [0,1]，不产生非法加权。
func TestAggregateAlphaClamp(t *testing.T) {
	items := []SessionScore{
		{SessionID: 1, Supervisor: NewDimensionScores(i(5), i(5), i(5), i(5), i(5)), Agent: NewDimensionScores(nil, i(4), i(3), i(3), i(3))},
	}
	supOnly := Aggregate(items, defaultWeights(), 1)
	aiClamp := Aggregate(items, defaultWeights(), 5)
	require.NotNil(t, supOnly.Composite)
	require.NotNil(t, aiClamp.Composite)
	assert.InDelta(t, *supOnly.Composite, *aiClamp.Composite, 1e-9, "alpha>1 应收敛为 1（只取督导侧）")
}
