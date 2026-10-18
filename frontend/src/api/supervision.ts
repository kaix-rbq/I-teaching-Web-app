import { request } from './http'
import type { PageResult } from '@/types/api'
import type {
  CoverageStat,
  SamplingTask,
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

/** 督导抽样任务队列（GET /supervision/ai-tasks，R1.2）：pending=1 只看待人工复评 */
export function fetchSamplingTasksApi(
  params: { semester?: string; pending?: boolean; page?: number; pageSize?: number } = {}
): Promise<PageResult<SamplingTask>> {
  return request<PageResult<SamplingTask>>({
    url: '/supervision/ai-tasks',
    method: 'get',
    params: { ...params, pending: params.pending ? 1 : undefined }
  })
}
