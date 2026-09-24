import dayjs from 'dayjs'

const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB']

export function formatBytes(bytes: number, fractionDigits = 1): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '0 B'
  if (bytes === 0) return '0 B'

  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), BYTE_UNITS.length - 1)
  const value = bytes / Math.pow(1024, index)
  const digits = index === 0 ? 0 : fractionDigits
  return `${value.toFixed(digits)} ${BYTE_UNITS[index]}`
}

export function formatDate(value?: string | number | Date, pattern = 'YYYY-MM-DD'): string {
  if (!value) return '-'
  const date = dayjs(value)
  return date.isValid() ? date.format(pattern) : '-'
}

export function formatDateTime(value?: string | number | Date): string {
  return formatDate(value, 'YYYY-MM-DD HH:mm')
}

export function formatRate(rate: number, fractionDigits = 0): string {
  if (!Number.isFinite(rate)) return '0%'
  const percent = Math.max(0, Math.min(1, rate)) * 100
  return `${percent.toFixed(fractionDigits)}%`
}

export function percentFromCounts(done: number, total: number): number {
  if (!total) return 0
  return done / total
}

/* ===== 《前端设计-new》评分色阶映射（区间阈值与锚点「3 分=合格」对齐：60 分即李克特 3 分） ===== */

export type ScoreTone = 1 | 2 | 3 | 4 | 5 | null

/** 连续分（0-100）→ 质量色阶档位；null/NaN → 无分数（禁止以 0 充当） */
export function scoreTone(score?: number | null): ScoreTone {
  if (score === null || score === undefined || !Number.isFinite(score)) return null
  if (score < 40) return 1
  if (score < 60) return 2
  if (score < 75) return 3
  if (score < 90) return 4
  return 5
}

/** 离散原始分（1-5 李克特）→ 色阶档位 */
export function rawScoreTone(value?: number | null): ScoreTone {
  if (value === null || value === undefined || !Number.isFinite(value)) return null
  return Math.min(5, Math.max(1, Math.round(value))) as Exclude<ScoreTone, null>
}

/** 档位 → 等级文案（与通用锚点同源） */
export function scoreLevelLabel(tone: ScoreTone): string {
  switch (tone) {
    case 5:
      return '优秀'
    case 4:
      return '良好'
    case 3:
      return '合格'
    case 2:
      return '待改进'
    case 1:
      return '需干预'
    default:
      return '暂无评价'
  }
}

/** 档位 → CSS 色阶变量（无分数返回中性灰，与业务色值铁律兼容） */
export function scoreToneColor(tone: ScoreTone): string {
  if (tone === null) return 'var(--color-score-void)'
  return `var(--color-score-${tone})`
}
