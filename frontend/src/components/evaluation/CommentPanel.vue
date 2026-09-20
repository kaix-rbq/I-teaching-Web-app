<script setup lang="ts">
import { EVALUATION_DIMENSIONS, getScoreLevel } from '@/constants'
import { formatDateTime } from '@/utils/format'
import type { DimensionKey, EvaluationDTO } from '@/types/evaluation'

withDefaults(
  defineProps<{
    items: EvaluationDTO[]
    loading?: boolean
  }>(),
  {
    loading: false
  }
)

const emit = defineEmits<{ (e: 'select', evaluation: EvaluationDTO): void }>()

function dimensionScore(evaluation: EvaluationDTO, key: DimensionKey): number | null {
  return evaluation[key]
}

function scoreText(score: number | null): string {
  return score === null || score === undefined ? '—' : String(score)
}

function totalText(score: number | null): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}
</script>

<template>
  <div class="comment-panel">
    <el-skeleton v-if="loading" :rows="4" animated />

    <template v-else>
      <div
        v-for="evaluation in items"
        :key="`${evaluation.evaluatorType}-${evaluation.evaluatorId}`"
        class="comment-panel__item"
        @click="emit('select', evaluation)"
      >
        <div class="comment-panel__head">
          <div class="comment-panel__who">
            <span class="comment-panel__name">{{ evaluation.evaluatorName }}</span>
            <el-tag size="small" effect="light" type="primary" round>督导</el-tag>
            <span class="comment-panel__time">{{ formatDateTime(evaluation.updatedAt) }}</span>
          </div>
          <span class="comment-panel__total tabular-nums">
            {{ totalText(evaluation.totalScore) }}
          </span>
        </div>

        <div class="comment-panel__scores">
          <div
            v-for="dimension in EVALUATION_DIMENSIONS"
            :key="dimension.key"
            class="comment-panel__score"
            :class="{ 'comment-panel__score--observation': dimension.isObservation }"
          >
            <span class="comment-panel__score-name">{{ dimension.shortName }}</span>
            <span class="comment-panel__score-value tabular-nums">
              {{ scoreText(dimensionScore(evaluation, dimension.key)) }}
            </span>
            <span class="comment-panel__score-level">
              {{
                evaluation[dimension.key] === null
                  ? ''
                  : getScoreLevel(evaluation[dimension.key] as number)?.level ?? ''
              }}
            </span>
          </div>
        </div>

        <dl v-if="evaluation.comment" class="comment-panel__block">
          <dt>总体评语</dt>
          <dd>{{ evaluation.comment }}</dd>
        </dl>
        <dl v-if="evaluation.highlights" class="comment-panel__block">
          <dt>亮点</dt>
          <dd>{{ evaluation.highlights }}</dd>
        </dl>
        <dl v-if="evaluation.improvements" class="comment-panel__block">
          <dt>待改进</dt>
          <dd>{{ evaluation.improvements }}</dd>
        </dl>
        <dl v-if="evaluation.suggestions" class="comment-panel__block">
          <dt>建议</dt>
          <dd>{{ evaluation.suggestions }}</dd>
        </dl>
      </div>

      <el-empty v-if="items.length === 0" description="暂无督导评语" :image-size="72" />
    </template>
  </div>
</template>

<style scoped lang="scss">
.comment-panel {
  &__item {
    padding: var(--spacing-4);
    margin-bottom: var(--spacing-3);
    cursor: pointer;
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    transition: border-color 0.2s ease, box-shadow 0.2s ease;

    &:hover {
      border-color: var(--color-primary);
      box-shadow: var(--shadow-card-hover);
    }

    &:last-child {
      margin-bottom: 0;
    }
  }

  &__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-3);
  }

  &__who {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
  }

  &__name {
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__time {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__total {
    font-size: var(--font-size-xl);
    font-weight: 600;
    color: var(--color-primary);
  }

  &__scores {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2) var(--spacing-4);
    padding: var(--spacing-2) var(--spacing-3);
    margin-bottom: var(--spacing-3);
    background-color: var(--color-bg-card);
    border-radius: var(--radius-md);
  }

  &__score {
    display: flex;
    align-items: baseline;
    gap: 4px;
    font-size: var(--font-size-sm);

    &--observation .comment-panel__score-value {
      color: var(--color-warning);
    }
  }

  &__score-name {
    color: var(--color-text-tertiary);
  }

  &__score-value {
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__score-level {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__block {
    display: flex;
    gap: var(--spacing-2);
    margin-top: var(--spacing-2);
    font-size: var(--font-size-sm);

    dt {
      flex-shrink: 0;
      width: 56px;
      color: var(--color-text-tertiary);
    }

    dd {
      flex: 1;
      line-height: 1.6;
      color: var(--color-text-secondary);
    }
  }
}
</style>
