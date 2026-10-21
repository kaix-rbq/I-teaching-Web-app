<script setup lang="ts">
import { computed } from 'vue'

export interface RadarItem {
  key: string
  name: string
  /** 原始分（默认 1-5）；null = 该维度无可展示分数，该轴断开不补 0 */
  score: number | null
  /** 画像链路：AI 分（null = 智能体未评此维度）；携带时图例以「AI」显形 */
  agentScore?: number | null
  /** 观测项：仅作亮点，不计入加权 */
  isObservation?: boolean
}

const props = withDefaults(
  defineProps<{
    items: RadarItem[]
    /** 分值上限，评分表单为 5 */
    max?: number
    loading?: boolean
    /** 图形语义色：画像链路为 AI（紫），评分表单为主色（蓝） */
    tone?: 'primary' | 'ai'
  }>(),
  {
    max: 5,
    loading: false,
    tone: 'primary'
  }
)

const SIZE = 240
const CENTER = SIZE / 2
const RADIUS = 78
const LABEL_RADIUS = RADIUS + 24

const axes = computed(() => {
  const count = props.items.length || 1
  return props.items.map((item, index) => {
    const angle = (-90 + (360 / count) * index) * (Math.PI / 180)
    return { item, angle }
  })
})

function pointAt(angle: number, radius: number): { x: number; y: number } {
  return {
    x: CENTER + radius * Math.cos(angle),
    y: CENTER + radius * Math.sin(angle)
  }
}

const rings = [0.25, 0.5, 0.75, 1].map((ratio) => RADIUS * ratio)

function ringPoints(radius: number): string {
  return axes.value
    .map((axis) => {
      const { x, y } = pointAt(axis.angle, radius)
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')
}

interface Vertex {
  item: RadarItem
  value: { x: number; y: number } | null
  label: { x: number; y: number }
  outer: { x: number; y: number }
}

function ratioOf(value: number | null | undefined): number | null {
  if (value === null || value === undefined || !Number.isFinite(value)) return null
  return Math.max(0, Math.min(1, value / props.max))
}

const vertices = computed<Vertex[]>(() =>
  axes.value.map((axis) => {
    const ratio = ratioOf(axis.item.score)
    return {
      item: axis.item,
      value: ratio === null ? null : pointAt(axis.angle, RADIUS * ratio),
      label: pointAt(axis.angle, LABEL_RADIUS),
      outer: pointAt(axis.angle, RADIUS)
    }
  })
)

const polygon = computed(() => {
  const filled = vertices.value.filter((vertex) => vertex.value !== null)
  return {
    points: filled
      .map((vertex) => `${vertex.value!.x.toFixed(1)},${vertex.value!.y.toFixed(1)}`)
      .join(' '),
    canFill: filled.length >= 3
  }
})

const hasScore = computed(() => vertices.value.some((vertex) => vertex.value !== null))

/** 仅画像链路（tone=ai）展示「AI」图例；评分表单 / 课程提优图不显示图例 */
const hasAgentLegend = computed(() => props.tone === 'ai')
</script>

<template>
  <div class="score-radar" :class="{ 'score-radar--ai': tone === 'ai' }">
    <el-skeleton v-if="loading" :rows="5" animated />

    <div v-else-if="!hasScore" class="score-radar__empty">
      <p>暂无评分数据</p>
    </div>

    <template v-else>
      <svg class="score-radar__svg" :viewBox="`0 0 ${SIZE} ${SIZE}`">
        <polygon
          v-for="(radius, index) in rings"
          :key="index"
          :points="ringPoints(radius)"
          class="score-radar__ring"
        />
        <line
          v-for="vertex in vertices"
          :key="`axis-${vertex.item.key}`"
          :x1="CENTER"
          :y1="CENTER"
          :x2="vertex.outer.x"
          :y2="vertex.outer.y"
          class="score-radar__axis"
        />
        <polygon v-if="polygon.canFill" :points="polygon.points" class="score-radar__area" />
        <template v-for="vertex in vertices" :key="`dot-${vertex.item.key}`">
          <circle
            v-if="vertex.value"
            :cx="vertex.value.x"
            :cy="vertex.value.y"
            :r="vertex.item.isObservation ? 4 : 3"
            class="score-radar__dot"
            :class="{ 'score-radar__dot--observation': vertex.item.isObservation }"
          />
        </template>
        <text
          v-for="vertex in vertices"
          :key="`label-${vertex.item.key}`"
          :x="vertex.label.x"
          :y="vertex.label.y"
          class="score-radar__label"
          text-anchor="middle"
          dominant-baseline="middle"
        >
          {{ vertex.item.name }}
        </text>
        <text
          v-for="vertex in vertices"
          :key="`value-${vertex.item.key}`"
          :x="vertex.label.x"
          :y="vertex.label.y + 13"
          class="score-radar__value tabular-nums"
          text-anchor="middle"
          dominant-baseline="middle"
        >
          {{ vertex.item.score === null ? '—' : vertex.item.score }}
        </text>
      </svg>

      <div v-if="hasAgentLegend" class="score-radar__legend">
        <span class="score-radar__legend-item">
          <i class="score-radar__swatch" />AI 评价
        </span>
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.score-radar {
  --radar-ink: var(--color-primary);

  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 220px;

  &--ai {
    --radar-ink: var(--color-ai-bright);
  }

  &__svg {
    width: 100%;
    max-width: 300px;
    height: auto;
  }

  &__ring {
    fill: none;
    stroke: var(--color-divider);
    stroke-width: 1;
  }

  &__axis {
    stroke: var(--color-divider);
    stroke-width: 1;
  }

  &__area {
    fill: color-mix(in srgb, var(--radar-ink) 16%, transparent);
    stroke: var(--radar-ink);
    stroke-width: 2;
    stroke-linejoin: round;
  }

  &__dot {
    fill: var(--radar-ink);

    &--observation {
      fill: var(--color-warning);
    }
  }

  &__label {
    font-size: 11px;
    fill: var(--color-text-secondary);
  }

  &__value {
    font-size: 11px;
    font-weight: 600;
    fill: var(--color-text-primary);
  }

  &__legend {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-3);
    align-items: center;
    justify-content: center;
    margin-top: var(--spacing-2);
    font-size: var(--font-size-xs);
    color: var(--color-text-secondary);

    &-item {
      display: inline-flex;
      align-items: center;
      gap: 6px;
    }
  }

  &__swatch {
    display: inline-block;
    width: 18px;
    height: 0;
    border-top: 2px solid var(--radar-ink);
  }

  &__empty {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 220px;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
