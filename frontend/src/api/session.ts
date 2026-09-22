import { request } from './http'
import type { PageResult } from '@/types/api'
import type {
  EvaluationDTO,
  SessionCreatePayload,
  SessionDetail,
  SessionEvaluation,
  SessionListItem,
  SupervisorEvaluationPayload
} from '@/types/evaluation'
import type { RecordingDTO, TranscriptDTO } from '@/types/evaluation'

export interface SessionListQuery {
  semester?: string
  page?: number
  pageSize?: number
}

/** 课程历史授课记录（GET /courses/:id/sessions），含每场双侧评分摘要 */
export function fetchCourseSessionsApi(
  courseId: number | string,
  query: SessionListQuery = {}
): Promise<PageResult<SessionListItem>> {
  return request<PageResult<SessionListItem>>({
    url: `/courses/${courseId}/sessions`,
    method: 'get',
    params: query
  })
}

/** 创建授课记录（POST /sessions，仅督导） */
export function createSessionApi(payload: SessionCreatePayload): Promise<SessionDetail> {
  return request<SessionDetail>({ url: '/sessions', method: 'post', data: payload })
}

/** 单场次基本信息（GET /sessions/:id） */
export function fetchSessionDetailApi(id: number | string): Promise<SessionDetail> {
  return request<SessionDetail>({ url: `/sessions/${id}`, method: 'get' })
}

/** 当堂课评估页聚合：场次 + 督导评分 + 智能体参考 + 评语（GET /sessions/:id/evaluation） */
export function fetchSessionEvaluationApi(id: number | string): Promise<SessionEvaluation> {
  return request<SessionEvaluation>({ url: `/sessions/${id}/evaluation`, method: 'get' })
}

/** 提交/覆盖督导评分与评语（PUT 幂等，五维必填） */
export function submitSupervisorEvaluationApi(
  id: number | string,
  payload: SupervisorEvaluationPayload
): Promise<EvaluationDTO> {
  return request<EvaluationDTO>({
    url: `/sessions/${id}/supervisor-evaluation`,
    method: 'put',
    data: payload
  })
}

export function uploadSessionRecordingApi(id: number | string, file: File, onProgress?: (percent: number) => void): Promise<RecordingDTO> {
  const form = new FormData()
  form.append('file', file)
  return request<RecordingDTO>({ url: `/sessions/${id}/recording`, method: 'post', data: form, headers: { 'Content-Type': 'multipart/form-data' }, onUploadProgress: (event) => { if (event.total && onProgress) onProgress(Math.round(event.loaded * 100 / event.total)) } })
}

export interface SessionMedia { recording: RecordingDTO | null; transcript: TranscriptDTO | null }
export function fetchSessionTranscriptApi(id: number | string): Promise<SessionMedia> {
  return request<SessionMedia>({ url: `/sessions/${id}/transcript`, method: 'get' })
}
export function retrySessionTranscriptApi(id: number | string): Promise<{ status: string }> {
  return request<{ status: string }>({ url: `/sessions/${id}/transcript/retry`, method: 'post' })
}
