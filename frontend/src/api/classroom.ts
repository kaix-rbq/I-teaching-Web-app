import { request } from './http'
import type { ClassroomIssue, ClassroomIssueCreatePayload } from '@/types/classroom'

/** 课堂问题：教师看本人，督导/主任看全部（service 层裁剪）。 */
export function fetchClassroomIssuesApi(): Promise<ClassroomIssue[]> {
  return request<ClassroomIssue[]>({ url: '/classroom-issues', method: 'get' })
}

export function createClassroomIssueApi(payload: ClassroomIssueCreatePayload): Promise<ClassroomIssue> {
  return request<ClassroomIssue>({ url: '/classroom-issues', method: 'post', data: payload })
}

export function handleClassroomIssueApi(id: number, handleNote: string): Promise<ClassroomIssue> {
  return request<ClassroomIssue>({
    url: `/classroom-issues/${id}/handle`,
    method: 'post',
    data: { handleNote }
  })
}

export function confirmClassroomIssueApi(id: number, resolved: boolean): Promise<{ id: number; resolved: boolean }> {
  return request<{ id: number; resolved: boolean }>({
    url: `/classroom-issues/${id}/confirm`,
    method: 'post',
    data: { resolved }
  })
}
