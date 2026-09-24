<script setup lang="ts">
import { computed } from 'vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import AiBadge from '@/components/common/AiBadge.vue'
import type { DimensionKey, EvaluationDTO } from '@/types/evaluation'

/**
 * 督导 × 智能体分维对照（《前端设计-new》§4.5 / §5.5）。
 * AI 侧统一走「AI 琥珀语法」：AiBadge 徽标 + 浅紫虚线卡 + 置信度可视化。
 * 红线继承：低置信（<0.4）标警示；阶段②未接入时显示预告卡，不得显示 0 分。
 */
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

const confidencePercent = computed(() =>
  Math.round((props.agent?.aiConfidence ?? 0) * 100)
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

    <!-- 阶段②未接入：预告卡（AI 语法淡出在场，但不用 0 分冒充数据） -->
    <template v-else-if="!hasAgent">
      <div class="evaluation-compare__placeholder ai-panel">
        <div class="evaluation-compare__placeholder-head">
          <AiBadge text="AI 参考" />
        </div>
        <p>智能体评价将在阶段②接入，届时此处展示与督导评分的分维度对比，并标注置信度与依据。</p>
      </div>
    </template>

    <template v-else>
      <div class="evaluation-compare__card ai-panel">
        <div class="evaluation-compare__head">
          <AiBadge text="AI 参考" />
          <span class="evaluation-compare__model">{{ agent?.aiModelVersion || '智能体' }}</span>
          <el-tag v-if="lowConfidence" type="warning" effect="light" round>低置信度</el-tag>
        </div>

        <div v-if="agent?.aiConfidence != null" class="evaluation-compare__confidence">
          <div class="evaluation-compare__confidence-track">
            <div
              class="evaluation-compare__confidence-fill"
              :style="{ width: `${confidencePercent}%` }"
              :class="{ 'evaluation-compare__confidence-fill--low': lowConfidence }"
            />
          </div>
          <span class="evaluation-compare__confidence-num tabular-nums">
            置信度 {{ confidencePercent }}%
          </span>
        </div>

        <ul class="evaluation-compare__list">
          <li
            v-for="dimension in EVALUATION_DIMENSIONS"
            :key="dimension.key"
            class="evaluation-compare__row"
          >
            <span class="evaluation-compare__name">{{ dimension.shortName }}</span>
            <span class="evaluation-compare__cell">
              <em>督导</em>
              <strong class="tabular-nums">{{ scoreOf(supervisor, dimension.key) }}</strong>
            </span>
            <span class="evaluation-compare__cell evaluation-compare__cell--ai">
              <em>AI</em>
              <strong class="tabular-nums">{{ scoreOf(agent, dimension.key) }}</strong>
            </span>
          </li>
        </ul>

        <p class="evaluation-compare__ethics">
          AI 评分仅作参考，最终以督导评分为准；低置信分不参与对照结论。
        </p>
      </div>
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

    &-head {
      display: flex;
    }
  }

  &__card {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
    padding: var(--spacing-4);
  }

  &__head {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
  }

  &__model {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__confidence {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;

    &-track {
      flex: 1;
      height: 6px;
      overflow: hidden;
      background-color: color-mix(in srgb, var(--color-ai) 12%, #ffffff);
      border-radius: 999px;
    }

    &-fill {
      height: 100%;
      background: var(--gradient-ai);
      border-radius: 999px;
      transition: width var(--duration-mid) var(--ease-out-soft);

      &--low {
        background: var(--color-warning);
      }
    }

    &-num {
      flex-shrink: 0;
      font-size: var(--font-size-xs);
      color: var(--color-text-secondary);
    }
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
    border-bottom: 1px dashed color-mix(in srgb, var(--color-ai) 24%, #ffffff);

    &:last-child {
      padding-bottom: 0;
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

    &--ai {
      em,
      strong {
        color: var(--color-ai-deep);
      }
    }
  }

  &__ethics {
    font-size: var(--font-size-xs);
    line-height: 1.6;
    color: var(--color-text-tertiary);
  }
}
</style>
