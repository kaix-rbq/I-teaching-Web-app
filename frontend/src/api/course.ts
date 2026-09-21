import { request } from './http'
import type { PageResult } from '@/types/api'
import type { Course, CourseQuery, CourseUpsertPayload } from '@/types/course'

export function fetchCoursesApi(query: CourseQuery): Promise<PageResult<Course>> {
  return request<PageResult<Course>>({
    url: '/courses',
    method: 'get',
    params: query
  })
}

export function fetchCourseDetailApi(id: number | string): Promise<Course> {
  return request<Course>({ url: `/courses/${id}`, method: 'get' })
}

export function fetchCourseEvaluationSummaryApi(id: number | string): Promise<import('@/types/course').CourseEvaluationSummary> {
  return request({ url: `/courses/${id}/evaluation-summary`, method: 'get' })
}

export function createCourseApi(payload: CourseUpsertPayload): Promise<Course> {
  return request<Course>({ url: '/courses', method: 'post', data: payload })
}

export function updateCourseApi(
  id: number | string,
  payload: CourseUpsertPayload
): Promise<Course> {
  return request<Course>({ url: `/courses/${id}`, method: 'put', data: payload })
}
