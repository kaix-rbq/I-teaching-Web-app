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
