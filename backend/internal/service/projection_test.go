package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
)

// TestSessionEvaluationSupervisorEvaluated 锁定 P1a 解耦口径：
// status（AI 已评析）不再代表督导已评，supervisorEvaluated 只看是否存在督导评价行。
func TestSessionEvaluationSupervisorEvaluated(t *testing.T) {
	sessions := &fakeSessionRepo{getDetail: func(context.Context, uint64) (*repository.SessionDetailRow, error) {
		return sessionRow(), nil
	}}
	fi := f64ptr

	cases := []struct {
		name string
		rows []repository.EvaluationRow
		want bool
	}{
		{
			name: "只有 AI 评价时为 false",
			rows: []repository.EvaluationRow{
				{Evaluation: model.Evaluation{
					SessionID: 7, EvaluatorType: model.EvaluatorAgent, TotalScore: fi(75),
				}},
			},
			want: false,
		},
		{
			name: "督导提交后为 true",
			rows: []repository.EvaluationRow{
				{Evaluation: model.Evaluation{
					SessionID: 7, EvaluatorType: model.EvaluatorAgent, TotalScore: fi(75),
				}},
				{Evaluation: model.Evaluation{
					SessionID: 7, EvaluatorType: model.EvaluatorSupervisor, EvaluatorID: 5,
					Comment: "互动偏少",
				}, EvaluatorName: "陈静"},
			},
			want: true,
		},
		{
			name: "无评价时为 false",
			rows: []repository.EvaluationRow{},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			evals := &fakeEvalRepo{listBySession: func(
				context.Context, uint64,
			) ([]repository.EvaluationRow, error) {
				return tc.rows, nil
			}}
			svc := NewSessionService(sessions, evals, &fakeCourseRepo{}, &fakeUserRepo{}, evalCfg())

			got, err := svc.Evaluation(context.Background(), RoleSupervisor, 0, 9, 7)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.SupervisorEvaluated)
			assert.Equal(t, tc.want, len(got.SupervisorScores) > 0)
			// status 仍是授课记录自己的状态，不因督导评价被改写。
			assert.Equal(t, "scheduled", got.Session.Status)
		})
	}
}

// TestToSessionListItemsSupervisorEvaluated 锁定列表投影直接透传 supervisor_evaluated，
// 不根据 status 或 supervisorScore 二次推断。
func TestToSessionListItemsSupervisorEvaluated(t *testing.T) {
	date := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	rows := []repository.SessionListRow{
		{
			ID: 1, SessionDate: date, Status: model.SessionStatusEvaluated,
			SupervisorEvaluated: false,
		},
		{
			ID: 2, SessionDate: date, Status: model.SessionStatusEvaluated,
			SupervisorEvaluated: true,
		},
	}

	items := toSessionListItems(rows)
	require.Len(t, items, 2)
	assert.False(t, items[0].SupervisorEvaluated, "AI 已评但督导未评必须为 false")
	assert.True(t, items[1].SupervisorEvaluated)
	assert.Equal(t, model.SessionStatusEvaluated, items[0].Status)
}
