/**
 * 智能体接口层（Sprint 3 阶段③）。
 *
 * 🔴 **真实智能体接口本次留空**：阶段②/③后端就绪前，以下函数不发起任何网络请求，
 * 直接返回 `src/mocks/teacherImprove.ts` 的前端临时演示数据。函数签名与未来后端
 * 保持一致，后端就绪后仅替换函数体（用 `request(...)`），页面与组件无需改动。
 *
 * 未来对接的真实接口（见开发计划 §4.2 / §4.4）：
 *   - `GET  /teachers/:id/score-trend?semester=&dimension=`
 *   - `POST /agent/chat`（SSE 流式）
 */
import type { AgentChatHandlers, AgentChatPayload, AgentSuggestion } from '@/types/agent'
import type { ScoreTrendSeries } from '@/types/teacher'
import { mockAgentChatReply, mockAgentSuggestions, mockScoreTrend } from '@/mocks/teacherImprove'

export interface ScoreTrendQuery {
  teacherId?: number | string
  courseId?: number | string
  semester?: string
  dimension?: string
}

/** 智能体提优建议（按时间倒序）。真实接口待后端就绪。 */
export function fetchAgentSuggestionsApi(
  courseId: number | string
): Promise<AgentSuggestion[]> {
  return Promise.resolve(mockAgentSuggestions(Number(courseId)))
}

/** 各维度历史趋势。真实接口 `GET /teachers/:id/score-trend` 待后端就绪。 */
export function fetchScoreTrendApi(query: ScoreTrendQuery = {}): Promise<ScoreTrendSeries[]> {
  void query
  return Promise.resolve(mockScoreTrend())
}

const CHUNK_SIZE = 3
const CHUNK_DELAY = 40

/**
 * 与智能体就课堂改进流式对话。
 * 真实实现为 SSE（断流可重连）；当前用定时分包模拟流式输出。
 * @returns 取消函数（离开页面 / 重发时调用）
 */
export function streamAgentChatApi(
  payload: AgentChatPayload,
  handlers: AgentChatHandlers
): () => void {
  const reply = mockAgentChatReply(payload.question)
  const chunks = Array.from({ length: Math.ceil(reply.length / CHUNK_SIZE) }, (_, index) =>
    reply.slice(index * CHUNK_SIZE, (index + 1) * CHUNK_SIZE)
  )

  let index = 0
  let cancelled = false
  let timer: ReturnType<typeof setTimeout> | null = null

  function tick(): void {
    if (cancelled) return
    if (index >= chunks.length) {
      handlers.onDone()
      return
    }
    handlers.onDelta(chunks[index])
    index += 1
    timer = setTimeout(tick, CHUNK_DELAY)
  }

  timer = setTimeout(tick, CHUNK_DELAY)

  return () => {
    cancelled = true
    if (timer) clearTimeout(timer)
  }
}
