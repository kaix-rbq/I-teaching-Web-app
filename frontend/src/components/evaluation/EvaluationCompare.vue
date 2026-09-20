<script setup lang="ts">
import { computed } from 'vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import type { DimensionKey, EvaluationDTO } from '@/types/evaluation'

const props = withDefaults(
  defineProps<{
    supervisor?: EvaluationDTO | null
    agent?: EvaluationDTO | null
    loading?: boolean
  }>(),
  {
    supervisor: null,
    agent: null,
    loading: false
  }
)

const LOW_CONFIDENCE = 0.4

const hasAgent = computed(() => props.agent !== null && props.agent !== undefined)

const lowConfidence = computed(
  () => hasAgent.value && (props.agent?.aiConfidence ?? 1) < LOW_CONFIDENCE
)

function scoreOf(evaluation: EvaluationDTO | null | undefined, key: DimensionKey): string {
  if (!evaluation) return '—'
  const value = evaluation[key]
  return value === null || value === undefined ? '未评' : String(value)
}
</script>

<template>
  <div class="evaluation-compare">
    <el-skeleton v-if="loading" :rows="4" animated />

    <template v-else-if="!hasAgent">
      <div class="evaluation-compare__placeholder">
        <el-tag type="info" effect="plain" round>AI 参考</el-tag>
        <p>智能体评价将在阶段②接入，届时此处展示与督导评分的分维度对比。</p>
      </div>
    </template>

    <template v-else>
      <div class="evaluation-compare__head">
        <el-tag type="info" effect="plain" round>AI 参考</el-tag>
        <span class="evaluation-compare__model">{{ agent?.aiModelVersion || '智能体' }}</span>
        <el-tag v-if="lowConfidence" type="warning" effect="light" round>低置信度</el-tag>
      </div>

      <p v-if="agent?.aiConfidence != null" class="evaluation-compare__confidence">
        模型自评置信度：{{ Math.round((agent.aiConfidence ?? 0) * 100) }}%
      </p>

      <ul class="evaluation-compare__list">
        <li
          v-for="dimension in EVALUATION_DIMENSIONS"
          :key="dimension.key"
          class="evaluation-compare__row"
        >
          <span class="evaluation-compare__name">{{ dimension.name }}</span>
          <span class="evaluation-compare__cell">
            <em>督导</em>
            <strong class="tabular-nums">{{ scoreOf(supervisor, dimension.key) }}</strong>
          </span>
          <span class="evaluation-compare__cell">
            <em>智能体</em>
            <strong class="tabular-nums">{{ scoreOf(agent, dimension.key) }}</strong>
          </span>
        </li>
      </ul>
    </template>
  </div>
</template>

<style scoped lang="scss">
.evaluation-compare {
  &__placeholder {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    padding: var(--spacing-4);
    font-size: var(--font-size-sm);
    line-height: 1.6;
    color: var(--color-text-tertiary);
    background-color: var(--color-bg-page);
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-lg);
  }

  &__head {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    margin-bottom: var(--spacing-2);
  }

  &__model {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__confidence {
    margin-bottom: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
  }

  &__row {
    display: grid;
    grid-template-columns: 1fr auto auto;
    gap: var(--spacing-3);
    align-items: center;
    padding-bottom: var(--spacing-2);
    font-size: var(--font-size-sm);
    border-bottom: 1px dashed var(--color-divider);

    &:last-child {
      border-bottom: none;
    }
  }

  &__name {
    color: var(--color-text-secondary);
  }

  &__cell {
    display: flex;
    align-items: baseline;
    gap: 4px;

    em {
      font-size: var(--font-size-xs);
      font-style: normal;
      color: var(--color-text-tertiary);
    }

    strong {
      color: var(--color-text-primary);
    }
  }
}
</style>
