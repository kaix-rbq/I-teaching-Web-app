/**
 * 督导评估草稿类型（与后端 internal/dto/draft.go 对齐）。
 * 草稿为未生效的私人工作副本：维度分可为 null，不进入任何聚合。
 */
export interface DraftDTO {
  id: number
  sessionId: number
  courseId: number
  courseCode: string
  courseName: string
  teacherName: string
  sessionDate: string
  period: string
  topic: string
  objective: number | null
  content: number | null
  interaction: number | null
  organization: number | null
  frontier: number | null
  comment: string
  highlights: string
  improvements: string
  suggestions: string
  createdAt: string
  updatedAt: string
}

/** 保存草稿请求体（允许部分维度为空） */
export interface DraftUpsertPayload {
  objective: number | null
  content: number | null
  interaction: number | null
  organization: number | null
  frontier: number | null
  comment: string
  highlights: string
  improvements: string
  suggestions: string
}
