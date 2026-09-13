import { describe, expect, it } from 'vitest'
import { formatBytes, formatDate, formatRate, percentFromCounts } from '@/utils/format'

describe('formatBytes', () => {
  it('returns 0 B for zero or invalid input', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(-1)).toBe('0 B')
    expect(formatBytes(Number.NaN)).toBe('0 B')
  })

  it('formats bytes with binary units', () => {
    expect(formatBytes(512)).toBe('512 B')
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatBytes(1024 * 1024)).toBe('1.0 MB')
    expect(formatBytes(1024 * 1024 * 1024 * 2)).toBe('2.0 GB')
  })
})

describe('formatDate', () => {
  it('returns dash for empty value', () => {
    expect(formatDate('')).toBe('-')
    expect(formatDate(undefined)).toBe('-')
  })

  it('formats valid dates', () => {
    expect(formatDate('2026-09-13T08:00:00.000Z', 'YYYY-MM-DD HH:mm')).toContain('2026-09-13')
    expect(formatDate('invalid')).toBe('-')
  })
})

describe('formatRate', () => {
  it('converts rate to percent string', () => {
    expect(formatRate(0)).toBe('0%')
    expect(formatRate(0.5)).toBe('50%')
    expect(formatRate(1)).toBe('100%')
    expect(formatRate(0.333, 1)).toBe('33.3%')
  })

  it('clamps out-of-range values', () => {
    expect(formatRate(-1)).toBe('0%')
    expect(formatRate(2)).toBe('100%')
    expect(formatRate(Number.NaN)).toBe('0%')
  })
})

describe('percentFromCounts', () => {
  it('computes ratio safely', () => {
    expect(percentFromCounts(3, 6)).toBe(0.5)
    expect(percentFromCounts(1, 0)).toBe(0)
  })
})
