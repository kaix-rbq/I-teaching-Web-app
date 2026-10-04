import { request } from './http'
import type { PageResult } from '@/types/api'
import type { EvaluationDTO } from '@/types/evaluation'
import type { DraftDTO, DraftUpsertPayload } from '@/types/draft'

export interface DraftListQuery {
  keyword?: string
  page?: number
  pageSize?: number
}

/** 草稿列表（GET /drafts，仅督导；可按课程名查询） */
export function fetchDraftsApi(query: DraftListQuery = {}): Promise<PageResult<DraftDTO>> {
  return request<PageResult<DraftDTO>>({ url: '/drafts', method: 'get', params: query })
}

/** 读取某场次的草稿（GET /sessions/:id/draft，无草稿返回 null） */
export function fetchSessionDraftApi(sessionId: number | string): Promise<DraftDTO | null> {
  return request<DraftDTO | null>({ url: `/sessions/${sessionId}/draft`, method: 'get' })
}

/** 保存/覆盖某场次的草稿（PUT /sessions/:id/draft） */
export function saveSessionDraftApi(
  sessionId: number | string,
  payload: DraftUpsertPayload
): Promise<DraftDTO> {
  return request<DraftDTO>({
    url: `/sessions/${sessionId}/draft`,
    method: 'put',
    data: payload
  })
}

/** 提交草稿为正式评价（POST /drafts/:id/submit，五维必须齐全） */
export function submitDraftApi(id: number | string): Promise<EvaluationDTO> {
  return request<EvaluationDTO>({ url: `/drafts/${id}/submit`, method: 'post' })
}

/** 删除草稿（DELETE /drafts/:id） */
export function deleteDraftApi(id: number | string): Promise<{ deleted: boolean }> {
  return request<{ deleted: boolean }>({ url: `/drafts/${id}`, method: 'delete' })
}
