<script setup lang="ts">
import { computed } from 'vue'
import ScoreRadar from '@/components/evaluation/ScoreRadar.vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import type { ScoreSummary } from '@/types/teacher'

const props = withDefaults(
  defineProps<{
    title: string
    summary: ScoreSummary | null
    loading?: boolean
    /** 无评价时的文案 */
    emptyText?: string
    /** 是否展示「与上学期对比」占位说明（当前接口未提供对比数据） */
    showCompareHint?: boolean
  }>(),
  {
    loading: false,
    emptyText: '暂无评价',
    showCompareHint: false
  }
)

const FLAG_TEXT: Record<string, string> = {
  no_data: '暂无评价',
  sample_insufficient: '样本不足，当前分数代表性有限',
  disjoint: '督导与智能体评价尚未对齐，当前分数代表性有限',
  formula_mixed: '口径版本混杂，请谨慎解读'
}

const dimensions = computed(() => props.summary?.dimensions ?? [])

const radarItems = computed(() =>
  dimensions.value.map((dimension) => ({
    key: dimension.key,
    name: dimension.name,
    score: dimension.score,
    isObservation: dimension.isObservation
  }))
)

const dimensionRows = computed(() =>
  EVALUATION_DIMENSIONS.map((meta) => {
    const dimension = dimensions.value.find((item) => item.key === meta.key)
    const score = dimension?.score ?? null
    return {
      ...meta,
      score,
      ratio: score === null ? 0 : Math.max(0, Math.min(100, score))
    }
  })
)

const flags = computed(() => props.summary?.flags ?? [])

function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

function flagText(flag: string): string {
  return FLAG_TEXT[flag] ?? flag
}
</script>

<template>
  <section class="score-summary">
    <h2 class="score-summary__title">{{ title }}</h2>

    <el-skeleton v-if="loading" :rows="6" animated />

    <div v-else-if="!summary || summary.compositeScore === null" class="score-summary__empty">
      <el-empty :description="emptyText" :image-size="72" />
    </div>

    <template v-else>
      <div class="score-summary__overview">
        <div class="score-summary__composite">
          <span class="score-summary__composite-label">综合分</span>
          <strong class="score-summary__composite-value tabular-nums">
            {{ formatScore(summary.compositeScore) }}
          </strong>
          <el-tag
            v-if="!summary.sample.sampleSufficient"
            type="warning"
            effect="light"
            round
          >
            样本不足（n={{ summary.sample.evaluatedCount }}）
          </el-tag>
        </div>

        <dl class="score-summary__sides">
          <div>
            <dt>督导分</dt>
            <dd class="tabular-nums">{{ formatScore(summary.supervisorScore) }}</dd>
          </div>
          <div>
            <dt>智能体分</dt>
            <dd class="tabular-nums">{{ formatScore(summary.agentScore) }}</dd>
          </div>
          <div>
            <dt>融合权重</dt>
            <dd>
              督导 {{ Math.round(summary.weights.supervisor * 100) }}% / 智能体
              {{ Math.round(summary.weights.agent * 100) }}%
            </dd>
          </div>
          <div>
            <dt>样本场次</dt>
            <dd class="tabular-nums">
              已评价 {{ summary.sample.evaluatedCount }} / 共 {{ summary.sample.sessionCount }}
            </dd>
          </div>
        </dl>
      </div>

      <p v-if="showCompareHint" class="score-summary__hint">
        与上学期对比将在历史数据齐备后提供。
      </p>

      <div v-if="flags.length" class="score-summary__flags">
        <el-alert
          v-for="flag in flags"
          :key="flag"
          :title="flagText(flag)"
          type="warning"
          :closable="false"
          show-icon
        />
      </div>

      <div class="score-summary__dimensions">
        <div class="score-summary__radar">
          <ScoreRadar :items="radarItems" :max="100" />
        </div>
        <div class="score-summary__bars">
          <div v-for="row in dimensionRows" :key="row.key" class="score-summary__bar">
            <div class="score-summary__bar-head">
              <span class="score-summary__bar-name">
                {{ row.name }}
                <el-tag v-if="row.isObservation" type="warning" size="small" effect="light" round>
                  观测项
                </el-tag>
                <el-tag v-else type="info" size="small" effect="plain" round>
                  {{ Math.round(row.weight * 100) }}%
                </el-tag>
              </span>
              <span class="score-summary__bar-value tabular-nums">
                {{ formatScore(row.score) }}
              </span>
            </div>
            <div class="score-summary__bar-track">
              <div
                class="score-summary__bar-fill"
                :class="{ 'score-summary__bar-fill--observation': row.isObservation }"
                :style="{ width: `${row.ratio}%` }"
              />
            </div>
          </div>
        </div>
      </div>
    </template>
  </section>
</template>

<style scoped lang="scss">
.score-summary {
  &__title {
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__overview {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-6);
  }

  &__composite {
    display: flex;
    align-items: center;
    gap: var(--spacing-3);
  }

  &__composite-label {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__composite-value {
    font-size: var(--font-size-stat);
    font-weight: 600;
    color: var(--color-primary);
  }

  &__sides {
    display: grid;
    grid-template-columns: repeat(2, minmax(180px, 1fr));
    gap: var(--spacing-3) var(--spacing-6);

    dt {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    dd {
      font-size: var(--font-size-base);
      font-weight: 500;
      color: var(--color-text-primary);
    }
  }

  &__hint {
    margin-top: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__flags {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    margin-top: var(--spacing-4);
  }

  &__dimensions {
    display: grid;
    grid-template-columns: 300px 1fr;
    gap: var(--spacing-6);
    align-items: center;
    margin-top: var(--spacing-4);

    @media (max-width: 1280px) {
      grid-template-columns: 1fr;
    }
  }

  &__bar {
    margin-bottom: var(--spacing-3);

    &:last-child {
      margin-bottom: 0;
    }
  }

  &__bar-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-1);
  }

  &__bar-name {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__bar-value {
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__bar-track {
    height: 8px;
    overflow: hidden;
    background-color: var(--color-divider);
    border-radius: var(--radius-sm);
  }

  &__bar-fill {
    height: 100%;
    background-color: var(--color-primary);
    border-radius: var(--radius-sm);
    transition: width 0.3s ease;

    &--observation {
      background-color: var(--color-warning);
    }
  }

  &__empty {
    padding: var(--spacing-4) 0;
  }
}
</style>
