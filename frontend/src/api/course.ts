import { request } from './http'
import type { PageResult } from '@/types/api'
import type { Course, CourseFormModel, CourseQuery } from '@/types/course'

export function fetchCoursesApi(query: CourseQuery): Promise<PageResult<Course>> {
  return request<PageResult<Course>>({
    url: '/courses',
    method: 'get',
    params: query
  })
}

export function fetchCourseDetailApi(id: string): Promise<Course> {
  return request<Course>({ url: `/courses/${id}`, method: 'get' })
}

export function createCourseApi(payload: Omit<CourseFormModel, 'id'>): Promise<Course> {
  return request<Course>({ url: '/courses', method: 'post', data: payload })
}

export function updateCourseApi(
  id: string,
  payload: Omit<CourseFormModel, 'id'>
): Promise<Course> {
  return request<Course>({ url: `/courses/${id}`, method: 'put', data: payload })
}
