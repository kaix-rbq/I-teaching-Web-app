import { request } from './http'
import type { DashboardData } from '@/types/supervision'

export function fetchDashboardApi(): Promise<DashboardData> {
  return request<DashboardData>({ url: '/dashboard', method: 'get' })
}
