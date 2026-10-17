import { request } from './http'
import type { PageResult } from '@/types/api'
import type {
  ImprovementItem,
  ImprovementListQuery,
  ImprovementCreatePayload
} from '@/types/improvement'

/** 待改进清单分页（teacher 本人 / supervisor 全部） */
export function fetchImprovementsApi(
  params: ImprovementListQuery = {}
): Promise<PageResult<ImprovementItem>> {
  return request<PageResult<ImprovementItem>>({ url: '/improvement-items', method: 'get', params })
}

/** 教师创建改进项 */
export function createImprovementApi(payload: ImprovementCreatePayload): Promise<ImprovementItem> {
  return request<ImprovementItem>({ url: '/improvement-items', method: 'post', data: payload })
}

/** 教师修改改进项标题 */
export function updateImprovementApi(id: number, title: string): Promise<ImprovementItem> {
  return request<ImprovementItem>({ url: `/improvement-items/${id}`, method: 'put', data: { title } })
}

/** 教师自评完成：open → done_self */
export function completeImprovementApi(id: number, doneNote: string): Promise<ImprovementItem> {
  return request<ImprovementItem>({
    url: `/improvement-items/${id}/complete`,
    method: 'put',
    data: { doneNote }
  })
}

/** 督导确认关闭：done_self → closed */
export function closeImprovementApi(id: number, note = ''): Promise<ImprovementItem> {
  return request<ImprovementItem>({
    url: `/improvement-items/${id}/close`,
    method: 'put',
    data: { note }
  })
}
