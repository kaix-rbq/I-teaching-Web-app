export type ClassroomIssueType = 'equipment' | 'absence' | 'progress' | 'material' | 'other'
export type ClassroomIssueStatus = 'open' | 'handled'

/** 课堂问题登记（GET/POST /classroom-issues）。 */
export interface ClassroomIssue {
  id: number
  teacherId: number
  teacherName: string
  issueType: ClassroomIssueType
  content: string
  status: ClassroomIssueStatus
  handleNote: string
  handlerId: number
  handlerName: string
  handledAt: string
  createdAt: string
}

export interface ClassroomIssueCreatePayload {
  issueType: ClassroomIssueType
  content: string
}

export const CLASSROOM_ISSUE_TYPES: Array<{ value: ClassroomIssueType; label: string }> = [
  { value: 'equipment', label: '设备故障' },
  { value: 'absence', label: '学生缺勤' },
  { value: 'progress', label: '进度问题' },
  { value: 'material', label: '材料缺失' },
  { value: 'other', label: '其他' }
]
