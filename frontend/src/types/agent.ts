/**
 * 智能体域类型（Sprint 3 阶段③）。
 *
 * ⚠️ 智能体后端接口当前未实现（阶段②/③排期）。本文件与 `api/agent.ts` 定义
 * 与未来后端保持一致的字段形状，页面先行使用 `src/mocks/teacherImprove.ts`
 * 的临时演示数据；后端就绪后仅需替换 `api/agent.ts` 的实现，页面无需改动。
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
}

/** 流式对话回调；real 实现为 SSE，当前由 mock 模拟分包输出 */
export interface AgentChatHandlers {
  onDelta: (chunk: string) => void
  onDone: () => void
  onError: (error: unknown) => void
}
