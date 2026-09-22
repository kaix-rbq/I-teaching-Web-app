import { request } from './http'
import type { PageResult } from '@/types/api'
import type { TeacherScoreItem, TeacherSummary, TeacherTimelineItem } from '@/types/teacher'

export function fetchTeacherScoresApi(params: Record<string, unknown> = {}): Promise<PageResult<TeacherScoreItem>> { return request({ url: '/teacher-scores', method: 'get', params }) }
export function fetchTeacherSummaryApi(id: number | string, params: Record<string, unknown> = {}): Promise<TeacherSummary> { return request({ url: `/teachers/${id}/evaluation-summary`, method: 'get', params }) }
export function fetchTeacherEvaluationsApi(id: number | string, params: Record<string, unknown> = {}): Promise<PageResult<TeacherTimelineItem>> { return request({ url: `/teachers/${id}/evaluations`, method: 'get', params }) }
