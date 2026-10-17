import { request } from './http'

export interface DemoSeedResult {
  recordings: number
  transcripts: number
  improvements: number
  reflections: number
  notifications: number
  message: string
}

/** POST /demo/seed：一键载入示例数据（仅本地/dev 开启，调用前需二次确认）。 */
export function seedDemoApi(): Promise<DemoSeedResult> {
  return request({ url: '/demo/seed', method: 'post' })
}
