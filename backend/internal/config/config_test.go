package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentConfigReportDebounceDuration(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want time.Duration
	}{
		{name: "缺省 10 分钟", raw: "", want: 10 * time.Minute},
		{name: "非法值回退 10 分钟", raw: "not-a-duration", want: 10 * time.Minute},
		{name: "非正值回退 10 分钟", raw: "0s", want: 10 * time.Minute},
		{name: "显式 1 秒", raw: "1s", want: time.Second},
		{name: "显式 5 分钟", raw: "5m", want: 5 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, AgentConfig{ReportDebounce: tc.raw}.ReportDebounceDuration())
		})
	}
}
