<script setup lang="ts">
import { computed } from 'vue'
import ScoreRadar from '@/components/evaluation/ScoreRadar.vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { ScoreSummary } from '@/types/teacher'

/**
 * 五维评分卡（AI 单源雷达 + 维度条权重徽章，docs/frontend_AGENTS.md §8.4）。
 * 教师画像 / 我的质量档案共用：分数一律取 AI 评价，缺失分渲染 — 不补 0。
 * 布局：雷达在左、维度读数在右（宽屏），窄屏回落单列。
 */
const props = withDefaults(
  defineProps<{
    summary: ScoreSummary | null
    title?: string
  }>(),
  {
    title: '五维评分 · AI 评价'
  }
)

/** 该维度的展示分 = AI 分；AI 未评时回退到后端该维度分，避免已有评价却显示空轴 */
function dimensionScore(dimension: ScoreSummary['dimensions'][number]): number | null {
  return dimension.agentScore ?? dimension.score ?? null
}

const radarItems = computed(() =>
  (props.summary?.dimensions ?? []).map((dimension) => ({
    key: dimension.key,
    name: dimension.name,
    score: dimensionScore(dimension),
    agentScore: dimension.agentScore,
    isObservation: dimension.isObservation
  }))
)

const dimensionRows = computed(() =>
  EVALUATION_DIMENSIONS.map((meta) => {
    const dimension = props.summary?.dimensions.find((item) => item.key === meta.key)
    const score = dimension ? dimensionScore(dimension) : null
    return {
      ...meta,
      score,
      ratio: score === null ? 0 : Math.max(0, Math.min(100, score)),
      color: scoreToneColor(scoreTone(score))
    }
  })
)

/** 数字一致性铁律：直接展示后端值，不做前端四舍五入 */
function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}
</script>

<template>
  <div class="dimensions-card">
    <h2 class="dimensions-card__title">{{ title }}</h2>

    <div class="dimensions-card__body">
      <div class="dimensions-card__radar">
        <ScoreRadar :items="radarItems" :max="100" tone="ai" />
      </div>

      <ul class="dimensions-card__dims">
        <li v-for="(row, index) in dimensionRows" :key="row.key" class="dimensions-card__dim">
          <div class="dimensions-card__dim-head">
            <span class="dimensions-card__dim-name">
              {{ row.name }}
              <el-tag v-if="row.isObservation" type="warning" size="small" effect="light" round>
                观测项
              </el-tag>
              <el-tag v-else type="info" size="small" effect="plain" round>
                {{ Math.round(row.weight * 100) }}%
              </el-tag>
            </span>
            <span
              class="dimensions-card__dim-score score-num"
              :style="{ color: row.score === null ? undefined : row.color }"
            >
              {{ formatScore(row.score) }}
            </span>
          </div>
          <div class="dimensions-card__track">
            <i
              class="dimensions-card__fill anim-grow-x"
              :style="{
                width: `${row.ratio}%`,
                backgroundColor: row.color,
                '--stagger': `${index * 60}ms`
              }"
            />
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped lang="scss">
.dimensions-card {
  display: flex;
  flex-direction: column;

  &__title {
    margin-bottom: var(--spacing-4);
    padding-bottom: var(--spacing-2);
    font-size: var(--font-size-base);
    color: var(--color-text-primary);
    border-bottom: 1px solid var(--color-divider);
  }

  &__body {
    display: grid;
    grid-template-columns: minmax(220px, 280px) minmax(0, 1fr);
    gap: var(--spacing-8);
    align-items: center;
  }

  &__radar {
    display: flex;
    justify-content: center;
  }

  &__dims {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
    padding: 0;
    margin: 0;
    list-style: none;
  }

  &__dim-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-1);
  }

  &__dim-name {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__dim-score {
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__track {
    height: 8px;
    overflow: hidden;
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-sm);
  }

  &__fill {
    display: block;
    height: 100%;
    border-radius: var(--radius-sm);
  }
}

@media (max-width: 960px) {
  .dimensions-card__body {
    grid-template-columns: 1fr;
    gap: var(--spacing-4);
  }
}
</style>
