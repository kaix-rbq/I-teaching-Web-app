/**
 * 智能体接口层（Sprint 3 阶段③）。
 *
 * S3 流式对话已接真实 SSE；提优建议/趋势仍由 `src/mocks/teacherImprove.ts`
 * 驱动，待 P6 再切真数据（函数签名保持不变）。
 */
import { TOKEN_KEY } from '@/constants'
import type { AgentChatHandlers, AgentChatPayload, AgentSuggestion } from '@/types/agent'
import type { ScoreTrendSeries } from '@/types/teacher'
import { mockAgentSuggestions, mockScoreTrend } from '@/mocks/teacherImprove'

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

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

/**
 * 与智能体就课堂改进流式对话（真实 SSE）。
 * 使用 fetch + ReadableStream，axios 不支持流式响应。
 * @returns 取消函数（离开页面 / 重发时调用）
 */
export function streamAgentChatApi(
  payload: AgentChatPayload,
  handlers: AgentChatHandlers
): () => void {
  const controller = new AbortController()
  let reader: ReadableStreamDefaultReader<Uint8Array> | null = null
  let finished = false

  const finish = (callback: () => void): void => {
    if (finished) return
    finished = true
    callback()
  }

  const fail = (error: unknown): void => {
    if (finished) return
    finished = true
    handlers.onError(error)
  }

  const handleFrame = (frame: string): void => {
    if (finished) return
    const data = frame
      .split(/\r?\n/)
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).trim())
      .join('\n')
    if (!data) return
    if (data === '[DONE]') {
      finish(handlers.onDone)
      return
    }
    let event: { delta?: unknown; done?: boolean; error?: { message?: string } }
    try {
      event = JSON.parse(data)
    } catch {
      fail(new Error('智能体返回了无法解析的数据'))
      return
    }
    if (event.error) {
      fail(new Error(event.error.message || '智能体暂时不可用'))
      return
    }
    if (typeof event.delta === 'string' && event.delta) {
      handlers.onDelta(event.delta)
      return
    }
    if (event.done) {
      finish(handlers.onDone)
    }
  }

  void (async () => {
    try {
      const token = localStorage.getItem(TOKEN_KEY)
      const response = await fetch(`${API_BASE_URL}/agent/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'text/event-stream',
          ...(token ? { Authorization: `Bearer ${token}` } : {})
        },
        body: JSON.stringify(payload),
        signal: controller.signal
      })
      if (!response.ok) {
        let message = `智能体请求失败（${response.status}）`
        try {
          const body = (await response.json()) as {
            message?: string
            error?: { message?: string }
          }
          message = body.message || body.error?.message || message
        } catch {
          // 非 JSON 错误体保持状态码文案
        }
        fail(new Error(message))
        return
      }
      if (!response.body) {
        fail(new Error('智能体响应流不可用'))
        return
      }
      reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      while (!finished) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        let match: RegExpExecArray | null
        while ((match = /\r?\n\r?\n/.exec(buffer)) !== null) {
          const frame = buffer.slice(0, match.index)
          buffer = buffer.slice(match.index + match[0].length)
          handleFrame(frame)
          if (finished) return
        }
      }
      buffer += decoder.decode()
      if (!finished && buffer.trim()) handleFrame(buffer)
      if (!finished) finish(handlers.onDone)
    } catch (error) {
      if (controller.signal.aborted) return
      fail(error)
    }
  })()

  return () => {
    controller.abort()
    if (reader) {
      void reader.cancel().catch(() => undefined)
    }
  }
}
