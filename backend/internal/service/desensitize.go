package service

import "aijiaoxue-api/pkg/scrub"

// Scrubber 是转写文本脱敏器；实现已迁至 pkg/scrub，服务层仅保留类型别名以稳定调用点。
type Scrubber = scrub.Scrubber

// NewScrubber 依据可选的姓名词典构造脱敏器。
func NewScrubber(dict []string) Scrubber { return scrub.NewScrubber(dict) }
