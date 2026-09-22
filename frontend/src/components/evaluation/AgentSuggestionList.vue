<script setup lang="ts">
import { formatDate } from '@/utils/format'
import type { AgentSuggestion } from '@/types/agent'

withDefaults(
  defineProps<{
    items: AgentSuggestion[]
    loading?: boolean
  }>(),
  {
    loading: false
  }
)

function confidenceText(confidence: number): string {
  return `${Math.round(Math.max(0, Math.min(1, confidence)) * 100)}%`
}
</script>

<template>
  <div class="agent-suggestions">
    <el-skeleton v-if="loading" :rows="4" animated />

    <template v-else>
      <p class="agent-suggestions__notice">
        <el-tag type="info" effect="plain" round>AI 参考</el-tag>
        以下提优建议由智能体依据课堂转写生成，仅用于教学改进参考，不代表考核结论。
      </p>

      <article
        v-for="item in items"
        :key="item.id"
        class="agent-suggestions__item"
      >
        <header class="agent-suggestions__head">
          <div class="agent-suggestions__when">
            <span class="agent-suggestions__date">{{ formatDate(item.sessionDate) }}</span>
            <span class="agent-suggestions__meta">
              {{ item.period || '—' }}
              <template v-if="item.topic"> · {{ item.topic }}</template>
            </span>
          </div>
          <div class="agent-suggestions__tags">
            <el-tag size="small" effect="light" round>{{ item.modelVersion }}</el-tag>
            <span class="agent-suggestions__confidence">
              置信度 {{ confidenceText(item.confidence) }}
            </span>
          </div>
        </header>

        <p class="agent-suggestions__summary">{{ item.summary }}</p>

        <dl v-if="item.highlights" class="agent-suggestions__row">
          <dt>亮点</dt>
          <dd>{{ item.highlights }}</dd>
        </dl>
        <dl v-if="item.improvements" class="agent-suggestions__row">
          <dt>待改进</dt>
          <dd>{{ item.improvements }}</dd>
        </dl>
        <dl v-if="item.evidence" class="agent-suggestions__row agent-suggestions__row--evidence">
          <dt>依据</dt>
          <dd>{{ item.evidence }}</dd>
        </dl>
      </article>

      <el-empty v-if="items.length === 0" description="暂无智能体提优建议" :image-size="72" />
    </template>
  </div>
</template>

<style scoped lang="scss">
.agent-suggestions {
  &__notice {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__item {
    padding: var(--spacing-4);
    margin-bottom: var(--spacing-3);
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);

    &:last-of-type {
      margin-bottom: 0;
    }
  }

  &__head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-2);
    margin-bottom: var(--spacing-2);
  }

  &__when {
    display: flex;
    align-items: baseline;
    gap: var(--spacing-2);
  }

  &__date {
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__meta {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__tags {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
  }

  &__confidence {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__summary {
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-secondary);
  }

  &__row {
    display: flex;
    gap: var(--spacing-2);
    margin-top: var(--spacing-2);
    font-size: var(--font-size-sm);

    dt {
      flex-shrink: 0;
      width: 52px;
      color: var(--color-text-tertiary);
    }

    dd {
      flex: 1;
      line-height: 1.6;
      color: var(--color-text-secondary);
    }

    &--evidence dd {
      color: var(--color-text-tertiary);
    }
  }
}
</style>
