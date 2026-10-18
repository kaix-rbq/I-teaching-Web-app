/** 待改进清单（N4 简化版：标题 + 来源 + 完成说明 + 三态） */
export type ImprovementStatus = 'open' | 'done_self' | 'closed'

export interface ImprovementItem {
  id: number
  teacherId: number
  teacherName: string
  sessionId: number
  title: string
  source: string
  status: ImprovementStatus
  doneNote: string
  closedBy: number
  closedByName: string
  closedAt: string
  createdAt: string
  updatedAt: string
}

export interface ImprovementListQuery {
  teacherId?: number
  status?: ImprovementStatus
  page?: number
  pageSize?: number
}

export interface ImprovementCreatePayload {
  title: string
  sessionId?: number
  source?: 'ai' | 'supervisor' | 'self'
}

/** 教学反思（N5，教师私有） */
export interface Reflection {
  id: number
  teacherId: number
  sessionId: number
  sourceEvalId: number
  content: string
  plan: string
  nextAction: string
  createdAt: string
  updatedAt: string
}

export interface ReflectionPayload {
  sessionId?: number
  sourceEvalId?: number
  content?: string
  plan?: string
  nextAction?: string
}

/** 站内信（B12） */
export type NotificationType = 'supervisor_comment' | 'ai_issue'

export interface NotificationItem {
  id: number
  type: NotificationType
  refId: number
  title: string
  isRead: boolean
  createdAt: string
}

export interface NotificationList {
  list: NotificationItem[]
  total: number
  unread: number
  page: number
  pageSize: number
}
