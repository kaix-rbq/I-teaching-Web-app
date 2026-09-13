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
  return request<PageResult<SupervisionPlan>>({
    url: '/supervision/plans',
    method: 'get',
    params: query
  })
}
