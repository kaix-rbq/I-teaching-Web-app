<script setup lang="ts">
import { computed } from 'vue'
import type { ScoreTrendPoint } from '@/types/teacher'

const props = withDefaults(
  defineProps<{
    points: ScoreTrendPoint[]
    /** 当前维度名称，用于无障碍描述与空态文案 */
    dimensionName?: string
    /** 分值上限（聚合接口为 0-100 制） */
    max?: number
    loading?: boolean
  }>(),
  {
    dimensionName: '',
    max: 100,
    loading: false
  }
)

const WIDTH = 640
const HEIGHT = 240
const PAD_LEFT = 40
const PAD_RIGHT = 16
const PAD_TOP = 16
const PAD_BOTTOM = 36
const PLOT_W = WIDTH - PAD_LEFT - PAD_RIGHT
const PLOT_H = HEIGHT - PAD_TOP - PAD_BOTTOM

const count = computed(() => props.points.length)

function xFor(index: number): number {
  if (count.value <= 1) return PAD_LEFT + PLOT_W / 2
  return PAD_LEFT + (PLOT_W * index) / (count.value - 1)
}

function yFor(value: number): number {
  const ratio = Math.max(0, Math.min(1, value / props.max))
  return PAD_TOP + PLOT_H * (1 - ratio)
}

const gridLines = [0, 0.25, 0.5, 0.75, 1].map((ratio) => ({
  y: PAD_TOP + PLOT_H * (1 - ratio),
  label: String(Math.round(props.max * ratio))
}))

const vertices = computed(() =>
  props.points.map((point, index) => ({
    point,
    x: xFor(index),
    y: point.value === null ? null : yFor(point.value)
  }))
)

const linePath = computed(() => {
  const scored = vertices.value.filter((vertex) => vertex.y !== null)
  if (scored.length === 0) return ''
  return scored
    .map((vertex, index) => `${index === 0 ? 'M' : 'L'} ${vertex.x.toFixed(1)} ${vertex.y!.toFixed(1)}`)
    .join(' ')
})

const hasScore = computed(() => vertices.value.some((vertex) => vertex.y !== null))
</script>

<template>
  <div class="trend-chart">
    <el-skeleton v-if="loading" :rows="5" animated />

    <div v-else-if="!hasScore" class="trend-chart__empty">
      <p>「{{ dimensionName || '该维度' }}」暂无历史评分数据</p>
    </div>

    <svg
      v-else
      class="trend-chart__svg"
      :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
      role="img"
      :aria-label="`${dimensionName}历史趋势折线`"
    >
      <line
        v-for="line in gridLines"
        :key="`grid-${line.label}`"
        :x1="PAD_LEFT"
        :y1="line.y"
        :x2="WIDTH - PAD_RIGHT"
        :y2="line.y"
        class="trend-chart__grid"
      />
      <text
        v-for="line in gridLines"
        :key="`axis-${line.label}`"
        :x="PAD_LEFT - 8"
        :y="line.y + 3"
        class="trend-chart__axis-label"
        text-anchor="end"
      >
        {{ line.label }}
      </text>

      <path v-if="linePath" :d="linePath" class="trend-chart__line" />

      <template v-for="vertex in vertices" :key="`point-${vertex.point.label}`">
        <g v-if="vertex.y !== null">
          <circle :cx="vertex.x" :cy="vertex.y" r="4" class="trend-chart__dot">
            <title>{{ vertex.point.date }} · {{ vertex.point.value }}</title>
          </circle>
          <text
            :x="vertex.x"
            :y="vertex.y! - 10"
            class="trend-chart__value tabular-nums"
            text-anchor="middle"
          >
            {{ vertex.point.value }}
          </text>
        </g>
        <text
          :x="vertex.x"
          :y="HEIGHT - 14"
          class="trend-chart__x-label"
          text-anchor="middle"
        >
          {{ vertex.point.label }}
        </text>
      </template>
    </svg>
  </div>
</template>

<style scoped lang="scss">
.trend-chart {
  min-height: 200px;

  &__svg {
    width: 100%;
    height: auto;
  }

  &__grid {
    stroke: var(--color-divider);
    stroke-width: 1;
  }

  &__axis-label {
    font-size: 11px;
    fill: var(--color-text-tertiary);
  }

  &__line {
    fill: none;
    stroke: var(--color-primary);
    stroke-width: 2;
    stroke-linejoin: round;
    stroke-linecap: round;
  }

  &__dot {
    fill: var(--color-bg-card);
    stroke: var(--color-primary);
    stroke-width: 2;
  }

  &__value {
    font-size: 12px;
    font-weight: 600;
    fill: var(--color-text-primary);
  }

  &__x-label {
    font-size: 12px;
    fill: var(--color-text-secondary);
  }

  &__empty {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 200px;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
