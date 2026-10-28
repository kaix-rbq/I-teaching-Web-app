import { onUnmounted, ref } from 'vue'
import { streamAgentChatApi } from '@/api/agent'
import type { AgentChatMessage, AgentChatPayload } from '@/types/agent'

/**
 * 与智能体流式对话的状态与副作用（阶段③）。
 * 数据请求封装在 composable 内，组件只消费 `messages` / `streaming` 并 emit 事件。
 */
export function useAgentChat() {
  const messages = ref<AgentChatMessage[]>([])
  const streaming = ref(false)

  let cancel: (() => void) | null = null
  let seq = 0

  function stop(): void {
    if (cancel) {
      cancel()
      cancel = null
    }
    streaming.value = false
    const last = messages.value[messages.value.length - 1]
    if (last?.pending) last.pending = false
  }

  function send(payload: AgentChatPayload, question: string): void {
    const text = question.trim()
    if (!text || streaming.value) return

    stop()
    // 当前问题单独放在 payload.question；history 只带此前最近 6 轮已完成消息，
    // 不含本次刚 push 的用户消息与待填充的 agent 占位。
    // 后端契约要求 history 最后一条必须是 user，因此去掉末尾已完成的 agent 回答。
    const recent = messages.value.filter((message) => !message.pending).slice(-12)
    while (recent.length > 0 && recent[recent.length - 1]?.role === 'agent') recent.pop()
    const history = recent.map((message) => ({ role: message.role, content: message.content }))
    const now = new Date().toISOString()
    messages.value.push(
      { id: (seq += 1), role: 'user', content: text, createdAt: now },
      { id: (seq += 1), role: 'agent', content: '', createdAt: now, pending: true }
    )
    const agentIndex = messages.value.length - 1
    streaming.value = true

    cancel = streamAgentChatApi({ ...payload, history }, {
      onDelta(chunk) {
        const message = messages.value[agentIndex]
        if (message) message.content += chunk
      },
      onDone() {
        const message = messages.value[agentIndex]
        if (message) message.pending = false
        streaming.value = false
        cancel = null
      },
      onError() {
        const message = messages.value[agentIndex]
        if (message) {
          message.pending = false
          if (!message.content) message.content = '智能体暂时无法响应，请稍后重试。'
        }
        streaming.value = false
        cancel = null
      }
    })
  }

  function reset(): void {
    stop()
    messages.value = []
  }

  onUnmounted(stop)

  return { messages, streaming, send, stop, reset }
}
