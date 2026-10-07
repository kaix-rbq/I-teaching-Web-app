<script setup lang="ts">
import { computed } from 'vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import AiBadge from '@/components/common/AiBadge.vue'
import { formatOffset } from '@/utils/format'
import type { DimensionKey, EvaluationDTO } from '@/types/evaluation'

/**
 * 督导 × 智能体分维对照（docs/frontend_AGENTS.md §8.4 / §9.0）。
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

/**
 * 证据面板数据：只列出「有引用」的维度，并保留维度中文短名与全部引用。
 * 智能体无法观测的维度（notObservable）没有引用，单独提示而不混进引用列表。
 */
const evidenceRows = computed(() => {
  const evidence = props.agent?.evidence
  if (!evidence) return []

  return EVALUATION_DIMENSIONS.flatMap((dimension) => {
    const item = evidence.dimensions?.[dimension.key]
    if (!item || !item.quotes?.length) return []
    return [
      {
        key: dimension.key,
        name: dimension.shortName,
        confidence: item.confidence,
        quotes: item.quotes
      }
    ]
  })
})

/** 智能体显式声明「无法评价」的维度名称，用于向用户解释该维度为何没有分数。 */
const notObservableNames = computed(() => {
  const list = props.agent?.evidence?.notObservable ?? []
  return list
    .map((key) => EVALUATION_DIMENSIONS.find((item) => item.key === key)?.shortName ?? key)
    .filter(Boolean)
})

const citedChunks = computed(() => props.agent?.evidence?.citedChunks ?? [])
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

        <!-- 证据引用：让「AI 为什么给这个分」可追溯（无证据不采信） -->
        <section v-if="evidenceRows.length || notObservableNames.length" class="evaluation-compare__evidence">
          <h4 class="evaluation-compare__evidence-title">评分依据</h4>

          <ul class="evaluation-compare__evidence-list">
            <li v-for="row in evidenceRows" :key="row.key" class="evaluation-compare__evidence-item">
              <div class="evaluation-compare__evidence-head">
                <span class="evaluation-compare__evidence-dim">{{ row.name }}</span>
                <span class="evaluation-compare__evidence-conf tabular-nums">
                  置信度 {{ Math.round(row.confidence * 100) }}%
                </span>
              </div>
              <blockquote
                v-for="(quote, index) in row.quotes"
                :key="`${row.key}-${index}`"
                class="evaluation-compare__evidence-quote"
              >
                <span class="evaluation-compare__evidence-time tabular-nums">
                  {{ formatOffset(quote.start) }}
                </span>
                {{ quote.quote }}
              </blockquote>
            </li>
          </ul>

          <p v-if="notObservableNames.length" class="evaluation-compare__evidence-note">
            以下维度智能体无法从课堂音频中观测，未给出分数：{{ notObservableNames.join('、') }}
          </p>

          <p v-if="citedChunks.length" class="evaluation-compare__evidence-note">
            引用知识库片段：{{ citedChunks.join('、') }}
          </p>
        </section>

        <!-- 有 AI 评分但无证据：按红线标注「无证据不采信」，不得让分数看起来有依据 -->
        <el-alert
          v-else
          class="evaluation-compare__no-evidence"
          type="info"
          :closable="false"
          show-icon
          title="本次评价未附带转写引用，建议结合督导评分判断"
        />

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

  &__evidence {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    padding-top: var(--spacing-3);
    border-top: 1px dashed color-mix(in srgb, var(--color-ai) 32%, #ffffff);

    &-title {
      font-size: var(--font-size-xs);
      font-weight: 600;
      color: var(--color-ai-deep);
    }

    &-list {
      display: flex;
      flex-direction: column;
      gap: var(--spacing-3);
    }

    &-item {
      display: flex;
      flex-direction: column;
      gap: 4px;
    }

    &-head {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: var(--spacing-2);
    }

    &-dim {
      font-size: var(--font-size-xs);
      font-weight: 600;
      color: var(--color-text-secondary);
    }

    &-conf {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    &-quote {
      padding: var(--spacing-2);
      margin: 0;
      font-size: var(--font-size-xs);
      line-height: 1.7;
      color: var(--color-text-secondary);
      background-color: color-mix(in srgb, var(--color-ai) 6%, #ffffff);
      border-left: 2px solid color-mix(in srgb, var(--color-ai) 40%, #ffffff);
      border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
    }

    &-time {
      margin-right: 4px;
      color: var(--color-ai-deep);
    }

    &-note {
      font-size: var(--font-size-xs);
      line-height: 1.6;
      color: var(--color-text-tertiary);
    }
  }

  &__no-evidence {
    margin: 0;
  }
}
</style>
