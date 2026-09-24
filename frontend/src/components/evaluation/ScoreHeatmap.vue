<script setup lang="ts">
import { computed } from 'vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import type { TeacherScoreItem } from '@/types/teacher'
import EmptyState from '@/components/common/EmptyState.vue'
import QualityBadge from '@/components/common/QualityBadge.vue'
import { scoreTone } from '@/utils/format'

/**
 * 教师 × 五维质量热力（《前端设计-new》§5.3 主任驾驶舱）。
 * 色阶深浅 = 维度分；缺分灰格；行尾综合分徽章；点行下钻教师画像。
 * 数据源：GET /teacher-scores（本室），无新增接口。
 */
const props = withDefaults(
  defineProps<{
    teachers: TeacherScoreItem[]
    loading?: boolean
  }>(),
  { loading: false }
)

const emit = defineEmits<{
  select: [teacherId: number]
}>()

const dimensions = computed(() => EVALUATION_DIMENSIONS.filter((item) => !item.isObservation))
const observation = computed(() => EVALUATION_DIMENSIONS.find((item) => item.isObservation))

const TONE_BOUNDS: Record<number, [number, number]> = {
  1: [0, 40],
  2: [40, 60],
  3: [60, 75],
  4: [75, 90],
  5: [90, 100.001]
}

/** 格子着色：档内相对位置决定色阶深浅（色阶只用于分数派生物） */
function cellStyle(score: number | null | undefined): Record<string, string> {
  if (score === null || score === undefined) {
    return {
      backgroundColor: 'color-mix(in srgb, var(--color-score-void) 16%, #ffffff)',
      color: 'var(--color-text-disabled)'
    }
  }
  const tone = scoreTone(score)
  if (tone === null) {
    return { backgroundColor: 'var(--color-bg-page)', color: 'var(--color-text-disabled)' }
  }
  const [lo, hi] = TONE_BOUNDS[tone]
  const pos = Math.min(1, Math.max(0, (score - lo) / (hi - lo)))
  const mix = Math.round(16 + pos * 58)
  return {
    backgroundColor: `color-mix(in srgb, var(--color-score-${tone}) ${mix}%, #ffffff)`,
    color: pos > 0.4 ? '#ffffff' : 'var(--color-text-primary)'
  }
}

function dimensionScore(teacher: TeacherScoreItem, key: string): number | null {
  return teacher.dimensions.find((item) => item.key === key)?.score ?? null
}

function observationScore(teacher: TeacherScoreItem): number | null {
  const key = observation.value?.key
  if (!key) return null
  return teacher.dimensions.find((item) => item.key === key)?.score ?? null
}

const sortedTeachers = computed(() => [...props.teachers].sort((a, b) => {
  const av = a.compositeScore
  const bv = b.compositeScore
  if (av === null) return 1
  if (bv === null) return -1
  return bv - av
}))
</script>

<template>
  <div class="score-heatmap">
    <el-skeleton v-if="loading" :rows="5" animated />

    <template v-else-if="sortedTeachers.length">
      <div
        class="score-heatmap__grid"
        role="table"
        :aria-label="`教师五维质量热力，共 ${sortedTeachers.length} 位教师`"
      >
        <div class="score-heatmap__head score-heatmap__row" role="row">
          <span class="score-heatmap__cell score-heatmap__cell--name" role="columnheader">教师</span>
          <span
            v-for="dim in dimensions"
            :key="dim.key"
            class="score-heatmap__cell score-heatmap__cell--dim"
            role="columnheader"
            :title="`${dim.name} · 权重 ${dim.weight}%`"
          >
            {{ dim.shortName }}<i class="score-heatmap__weight">{{ dim.weight }}%</i>
          </span>
          <span
            v-if="observation"
            class="score-heatmap__cell score-heatmap__cell--dim score-heatmap__cell--obs"
            role="columnheader"
            :title="`${observation.name} · 观测项不计分`"
          >
            {{ observation.shortName }}<i class="score-heatmap__weight">观测</i>
          </span>
          <span class="score-heatmap__cell score-heatmap__cell--total" role="columnheader">
            综合分
          </span>
        </div>

        <button
          v-for="teacher in sortedTeachers"
          :key="teacher.teacherId"
          type="button"
          class="score-heatmap__row"
          role="row"
          @click="emit('select', teacher.teacherId)"
        >
          <span class="score-heatmap__cell score-heatmap__cell--name" role="cell">
            {{ teacher.teacherName }}
          </span>
          <span
            v-for="dim in dimensions"
            :key="dim.key"
            class="score-heatmap__cell score-heatmap__cell--dim"
            role="cell"
            :style="cellStyle(dimensionScore(teacher, dim.key))"
          >
            {{ dimensionScore(teacher, dim.key)?.toFixed(0) ?? '—' }}
          </span>
          <span
            v-if="observation"
            class="score-heatmap__cell score-heatmap__cell--dim score-heatmap__cell--obs"
            role="cell"
            :style="cellStyle(observationScore(teacher))"
          >
            {{ observationScore(teacher)?.toFixed(0) ?? '—' }}
          </span>
          <span class="score-heatmap__cell score-heatmap__cell--total" role="cell">
            <QualityBadge
              :score="teacher.compositeScore"
              :level="teacher.compositeScore === null ? '暂无' : undefined"
            />
          </span>
        </button>
      </div>

      <p class="score-heatmap__legend">
        色阶由红至绿对应「需干预 → 优秀」；灰格为暂无评价。点击行查看教师画像。
      </p>
    </template>

    <EmptyState v-else description="本教研室暂无教师评分数据" />
  </div>
</template>

<style scoped lang="scss">
.score-heatmap {
  width: 100%;

  &__grid {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  &__row {
    display: grid;
    grid-template-columns: minmax(88px, 1.2fr) repeat(6, minmax(52px, 1fr)) minmax(72px, 0.9fr);
    gap: 4px;
    align-items: stretch;
    width: 100%;
    padding: 0;
    font: inherit;
    color: inherit;
    text-align: center;
    background: transparent;
    border: none;
    cursor: pointer;
  }

  &__head {
    cursor: default;
  }

  &__row:not(&__head):hover {
    .score-heatmap__cell {
      box-shadow: inset 0 0 0 1.5px var(--color-primary);
    }
  }

  &__cell {
    display: flex;
    flex-direction: column;
    gap: 2px;
    align-items: center;
    justify-content: center;
    min-height: 44px;
    padding: var(--spacing-2) 4px;
    font-size: var(--font-size-sm);
    line-height: 1.3;
    border-radius: var(--radius-md);

    &--name {
      align-items: flex-start;
      font-weight: 600;
      color: var(--color-text-primary);
      text-align: left;
      background-color: var(--color-bg-page);
    }

    &--dim {
      font-variant-numeric: tabular-nums;
    }

    &--obs {
      font-style: italic;
    }

    &--total {
      background-color: var(--color-bg-page);
    }
  }

  &__head &__cell {
    min-height: 32px;
    font-size: var(--font-size-xs);
    color: var(--color-text-secondary);
    background-color: transparent;
  }

  &__weight {
    font-size: 10px;
    font-style: normal;
    color: var(--color-text-tertiary);
  }

  &__legend {
    margin-top: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    text-align: center;
  }
}
</style>
