import type { Role } from '@/types/user'
import type { CourseStatus } from '@/types/course'
import type { ResourceType } from '@/types/resource'
import type { SupervisionStatus } from '@/types/supervision'

export const TOKEN_KEY = 'aijiaoxue_token'
export const USER_KEY = 'aijiaoxue_user'

export interface RoleMeta {
  value: Role
  label: string
  shortLabel: string
  color: string
}

export const ROLES: RoleMeta[] = [
  { value: 'director', label: '教研室主任', shortLabel: '主任', color: '#2f54eb' },
  { value: 'teacher', label: '教师', shortLabel: '教师', color: '#13c2c2' },
  { value: 'supervisor', label: '教学督导', shortLabel: '督导', color: '#722ed1' }
]

export function getRoleMeta(role?: Role): RoleMeta | undefined {
  return ROLES.find((item) => item.value === role)
}

export interface OptionMeta<T extends string> {
  value: T
  label: string
}

export const COURSE_STATUS: OptionMeta<CourseStatus>[] = [
  { value: 'open', label: '开课中' },
  { value: 'draft', label: '草稿' },
  { value: 'closed', label: '已结课' }
]

export const SUPERVISION_STATUS: OptionMeta<SupervisionStatus>[] = [
  { value: 'planned', label: '计划中' },
  { value: 'completed', label: '已完成' }
]

export type TagType = 'success' | 'warning' | 'info' | 'primary' | 'danger'

export function getCourseStatusMeta(status: CourseStatus): {
  label: string
  tag: TagType
} {
  const map: Record<CourseStatus, { label: string; tag: TagType }> = {
    open: { label: '开课中', tag: 'success' },
    draft: { label: '草稿', tag: 'warning' },
    closed: { label: '已结课', tag: 'info' }
  }
  return map[status]
}

export function getSupervisionStatusMeta(status: SupervisionStatus): {
  label: string
  tag: TagType
} {
  const map: Record<SupervisionStatus, { label: string; tag: TagType }> = {
    planned: { label: '计划中', tag: 'warning' },
    completed: { label: '已完成', tag: 'success' }
  }
  return map[status]
}

export const RESOURCE_ACCEPT = '.pdf,.doc,.docx,.ppt,.pptx,.mp4,.zip'
export const MAX_RESOURCE_SIZE = 100 * 1024 * 1024

export const RESOURCE_TYPE_LABELS: Record<ResourceType, string> = {
  pdf: 'PDF',
  doc: '文档',
  ppt: '演示文稿',
  video: '视频',
  zip: '压缩包',
  other: '其他'
}

export const PAGE_SIZES = [10, 20, 50]

/** 学期字典：Sprint 1 由前端常量维护，与后端 internal/service/consts.go 的 Semesters 同源 */
export const SEMESTERS = ['2026-2027-1', '2025-2026-2', '2025-2026-1']

/** 当前学期（工作台/覆盖率口径），与后端 CurrentSemester 一致 */
export const CURRENT_SEMESTER = SEMESTERS[0]

export function resolveResourceType(fileName: string): ResourceType {
  const ext = fileName.slice(fileName.lastIndexOf('.') + 1).toLowerCase()
  if (ext === 'pdf') return 'pdf'
  if (ext === 'doc' || ext === 'docx') return 'doc'
  if (ext === 'ppt' || ext === 'pptx') return 'ppt'
  if (ext === 'mp4' || ext === 'avi' || ext === 'mov') return 'video'
  if (ext === 'zip' || ext === 'rar' || ext === '7z') return 'zip'
  return 'other'
}
