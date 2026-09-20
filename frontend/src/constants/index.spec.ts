import { describe, expect, it } from 'vitest'
import {
  EVALUATION_DIMENSIONS,
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
