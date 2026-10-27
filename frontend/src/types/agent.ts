/**
 * 智能体域类型（Sprint 3 阶段③）。
 *
 * S3 流式对话已接真实 SSE；提优建议/趋势仍由前端 mock 驱动（P6 再切换）。
 * 本文件字段与后端 DTO 保持一致。
 */

/** 智能体针对某次课堂记录给出的提优建议（对应 evaluations 的 agent 行 suggestions 字段） */
export interface AgentSuggestion {
  id: number
  courseId: number
  sessionId: number
  /** YYYY-MM-DD */
  sessionDate: string
  period: string
  topic: string
  /** 提优建议正文（按时间倒序展示的排序依据为 sessionDate） */
  summary: string
  highlights: string
  improvements: string
  /** 建议依据的转写片段（已脱敏） */
  evidence: string
  modelVersion: string
  /** 模型自评置信度 0-1 */
  confidence: number
  createdAt: string
}

export type AgentChatRole = 'user' | 'agent'

export interface AgentChatHistoryItem {
  role: AgentChatRole
  content: string
}

export interface AgentChatMessage {
  id: number
  role: AgentChatRole
  content: string
  createdAt: string
  /** 正在流式输出中（用于显示光标 / 禁用输入） */
  pending?: boolean
}

export interface AgentChatPayload {
  courseId: number
  question: string
  /** 最近对话历史；后端会截断为最近 6 条 */
  history?: AgentChatHistoryItem[]
}

/** 流式对话回调；真实实现为 SSE */
export interface AgentChatHandlers {
  onDelta: (chunk: string) => void
  onDone: () => void
  onError: (error: unknown) => void
}
