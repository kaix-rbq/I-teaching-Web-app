import { describe, expect, it } from 'vitest'
import {
  EVALUATION_DIMENSIONS,
  flagText,
  getDimensionAnchor,
  getScoreLevel,
  getSessionStatusMeta,
  SCORE_LEVELS
} from '@/constants'

describe('SCORE_LEVELS / getScoreLevel', () => {
  it('covers all five levels', () => {
    expect(SCORE_LEVELS.map((item) => item.value)).toEqual([5, 4, 3, 2, 1])
  })

  it('resolves a known level and misses unknown scores', () => {
    expect(getScoreLevel(5)?.level).toBe('优秀')
    expect(getScoreLevel(1)?.level).toBe('不合格')
    expect(getScoreLevel(0)).toBeUndefined()
  })
})

describe('EVALUATION_DIMENSIONS', () => {
  it('has five frozen dimensions with a single observation item', () => {
    expect(EVALUATION_DIMENSIONS).toHaveLength(5)
    const observations = EVALUATION_DIMENSIONS.filter((item) => item.isObservation)
    expect(observations).toHaveLength(1)
    expect(observations[0].key).toBe('frontier')
    expect(observations[0].weight).toBe(0)
  })

  it('defines an anchor for every dimension and every score', () => {
    for (const dimension of EVALUATION_DIMENSIONS) {
      for (const level of SCORE_LEVELS) {
        expect(getDimensionAnchor(dimension.key, level.value)).not.toBe('')
      }
    }
  })
})

describe('getSessionStatusMeta', () => {
  it('marks evaluated sessions as success', () => {
    expect(getSessionStatusMeta('evaluated').tag).toBe('success')
    expect(getSessionStatusMeta('scheduled').tag).toBe('warning')
  })
})

describe('flagText', () => {
  // 后端 pkg/scoring 实际会产出的全部 flag（含 service 层追加的 sample_insufficient）。
  // 回归点：展示面此前漏了 sup_only / ai_only，页面会直接显示英文原始 key。
  const BACKEND_FLAGS = [
    'no_data',
    'sup_only',
    'ai_only',
    'disjoint',
    'sample_insufficient'
  ]

  it('maps every backend flag to Chinese copy', () => {
    for (const flag of BACKEND_FLAGS) {
      const text = flagText(flag)
      expect(text).not.toBe(flag)
      // 文案必须是中文，出现英文 key 说明字典漏配
      expect(text).toMatch(/[\u4e00-\u9fa5]/)
    }
  })

  it('falls back to the raw key for unknown flags', () => {
    // 未知 flag 原样返回，便于排障时发现前后端契约漂移
    expect(flagText('brand_new_flag')).toBe('brand_new_flag')
  })
})
