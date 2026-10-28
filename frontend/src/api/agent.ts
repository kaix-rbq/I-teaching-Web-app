/**
 * 智能体接口层（Sprint 3 阶段③）。
 *
 * 提优建议、分数趋势、教师级 AI 报告、S3 流式对话均接真实后端；
 * 不再依赖前端演示数据。
 */
import { TOKEN_KEY } from '@/constants'
import { request } from './http'
import type {
  AgentChatHandlers,
  AgentChatPayload,
  AgentSuggestion,
  AiReportEnvelope
} from '@/types/agent'
import type { ScoreTrendSeries } from '@/types/teacher'

export interface ScoreTrendQuery {
  teacherId?: number | string
  courseId?: number | string
  semester?: string
  dimension?: string
}

/** 课程 AI 提优建议（按课次倒序，v1 数组）。 */
export function fetchAgentSuggestionsApi(
  courseId: number | string,
  semester?: string
): Promise<AgentSuggestion[]> {
  return request<AgentSuggestion[]>({
    url: `/courses/${courseId}/agent-suggestions`,
    method: 'get',
    params: semester ? { semester } : undefined
  })
}

/** 教师综合分 + 五维趋势。teacherId 必填。 */
export function fetchScoreTrendApi(query: ScoreTrendQuery = {}): Promise<ScoreTrendSeries[]> {
  const { teacherId, courseId, semester, dimension } = query
  if (teacherId === undefined || teacherId === null || teacherId === '') {
    return Promise.reject(new Error('缺少教师 ID'))
  }
  const params: Record<string, string | number> = {}
  if (semester) params.semester = semester
  if (courseId !== undefined && courseId !== null && courseId !== '' && Number(courseId) > 0) {
    params.courseId = Number(courseId)
  }
  if (dimension) params.dimension = dimension
  return request<ScoreTrendSeries[]>({
    url: `/teachers/${teacherId}/score-trend`,
    method: 'get',
    params
  })
}

/** 教师级 AI 综合报告（S2）。无报告时 report=null、status=pending。 */
export function fetchTeacherAiReportApi(
  teacherId: number | string,
  params: { semester?: string } = {}
): Promise<AiReportEnvelope> {
  return request<AiReportEnvelope>({
    url: `/teachers/${teacherId}/ai-report`,
    method: 'get',
    params: params.semester ? { semester: params.semester } : undefined
  })
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
