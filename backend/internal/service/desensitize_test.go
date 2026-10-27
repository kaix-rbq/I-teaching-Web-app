package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewScrubberWrapper 验证 service 薄适配层仍可正常构造并调用 pkg/scrub 实现。
func TestNewScrubberWrapper(t *testing.T) {
	got, segs := NewScrubber(nil).Scrub("张伟同学回答得很好", nil)
	assert.Equal(t, "学生A同学回答得很好", got)
	assert.Empty(t, segs)
}
