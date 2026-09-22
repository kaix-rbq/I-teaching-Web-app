<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { AgentChatMessage } from '@/types/agent'

const props = withDefaults(
  defineProps<{
    messages: AgentChatMessage[]
    streaming?: boolean
    disabled?: boolean
  }>(),
  {
    streaming: false,
    disabled: false
  }
)

const emit = defineEmits<{
  (e: 'send', question: string): void
  (e: 'stop'): void
}>()

const PROMPTS = ['如何提升课堂互动？', '我的节奏控制得怎么样？', '内容深度如何改进？', '近几次课有进步吗？']

const draft = ref('')
const scroller = ref<HTMLElement | null>(null)

function send(): void {
  const text = draft.value.trim()
  if (!text || props.streaming || props.disabled) return
  emit('send', text)
  draft.value = ''
  void scrollToBottom()
}

function usePrompt(prompt: string): void {
  if (props.streaming || props.disabled) return
  draft.value = prompt
  send()
}

async function scrollToBottom(): Promise<void> {
  await nextTick()
  const el = scroller.value
  if (el) el.scrollTop = el.scrollHeight
}

watch(
  () => props.messages.map((message) => message.content).join(''),
  () => void scrollToBottom()
)
</script>

<template>
  <div class="agent-chat">
    <div ref="scroller" class="agent-chat__body">
      <div v-if="messages.length === 0" class="agent-chat__empty">
        <p class="agent-chat__empty-title">与智能体聊聊这节课怎么改进</p>
        <p class="agent-chat__empty-desc">
          可就互动、节奏、内容深度等提问，智能体将结合课堂转写给出建议。
        </p>
        <div class="agent-chat__prompts">
          <el-tag
            v-for="prompt in PROMPTS"
            :key="prompt"
            class="agent-chat__prompt"
            effect="plain"
            round
            @click="usePrompt(prompt)"
          >
            {{ prompt }}
          </el-tag>
        </div>
      </div>

      <div
        v-for="message in messages"
        :key="message.id"
        class="agent-chat__message"
        :class="`agent-chat__message--${message.role}`"
      >
        <span class="agent-chat__avatar">
          {{ message.role === 'user' ? '我' : 'AI' }}
        </span>
        <div class="agent-chat__bubble">
          <span>{{ message.content }}</span>
          <span v-if="message.pending" class="agent-chat__cursor" />
        </div>
      </div>
    </div>

    <div class="agent-chat__input">
      <el-input
        v-model="draft"
        type="textarea"
        :rows="2"
        resize="none"
        :disabled="disabled"
        placeholder="就课堂改进向智能体提问…（Enter 发送，Shift+Enter 换行）"
        @keydown.enter.exact.prevent="send"
      />
      <div class="agent-chat__actions">
        <el-button v-if="streaming" @click="emit('stop')">停止生成</el-button>
        <el-button type="primary" :disabled="disabled || !draft.trim()" @click="send">
          发送
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.agent-chat {
  &__body {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
    max-height: 360px;
    padding: var(--spacing-4);
    overflow-y: auto;
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
  }

  &__empty {
    padding: var(--spacing-4) 0;
    text-align: center;
  }

  &__empty-title {
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__empty-desc {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__prompts {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    justify-content: center;
    margin-top: var(--spacing-3);
  }

  &__prompt {
    cursor: pointer;

    &:hover {
      color: var(--color-primary);
      border-color: var(--color-primary);
    }
  }

  &__message {
    display: flex;
    gap: var(--spacing-2);
    align-items: flex-start;

    &--user {
      flex-direction: row-reverse;

      .agent-chat__bubble {
        color: var(--color-primary);
        background-color: var(--color-primary-bg);
        border-color: var(--color-primary-bg);
      }

      .agent-chat__avatar {
        color: var(--color-primary);
        background-color: var(--color-primary-bg);
      }
    }
  }

  &__avatar {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--color-text-secondary);
    background-color: var(--color-divider);
    border-radius: 50%;
  }

  &__bubble {
    max-width: 80%;
    padding: var(--spacing-2) var(--spacing-3);
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-secondary);
    white-space: pre-wrap;
    word-break: break-word;
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
  }

  &__cursor {
    display: inline-block;
    width: 6px;
    height: 14px;
    margin-left: 2px;
    vertical-align: text-bottom;
    background-color: var(--color-primary);
    animation: agent-chat-blink 1s steps(2, start) infinite;
  }

  &__input {
    margin-top: var(--spacing-3);
  }

  &__actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--spacing-2);
    margin-top: var(--spacing-2);
  }
}

@keyframes agent-chat-blink {
  to {
    visibility: hidden;
  }
}
</style>
