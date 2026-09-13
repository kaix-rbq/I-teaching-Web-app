import { request } from './http'
import type { PageResult } from '@/types/api'
import type {
  CoverageStat,
  SupervisionPlan,
  SupervisionQuery
} from '@/types/supervision'

export function fetchCoverageApi(): Promise<CoverageStat> {
  return request<CoverageStat>({ url: '/supervision/coverage', method: 'get' })
}

export function fetchPlansApi(query: SupervisionQuery): Promise<PageResult<SupervisionPlan>> {
  // 前端用 date（起始日期）表达筛选，后端契约为 dateFrom
  const { date, ...rest } = query
  return request<PageResult<SupervisionPlan>>({
    url: '/supervision/plans',
    method: 'get',
    params: { ...rest, dateFrom: date || undefined }
  })
}
