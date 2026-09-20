<script setup lang="ts">
import { computed } from 'vue'

export interface RadarItem {
  key: string
  name: string
  /** 原始分（默认 1-5）；null = 该评分源无法评价此维度 */
  score: number | null
  /** 观测项：仅作亮点，不计入加权 */
  isObservation?: boolean
}

const props = withDefaults(
  defineProps<{
    items: RadarItem[]
    /** 分值上限，评分表单为 5 */
    max?: number
    loading?: boolean
  }>(),
  {
    max: 5,
    loading: false
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

const vertices = computed(() =>
  axes.value.map((axis) => {
    const raw = axis.item.score
    const ratio = raw === null || raw === undefined ? null : Math.max(0, Math.min(1, raw / props.max))
    const outer = pointAt(axis.angle, RADIUS)
    const label = pointAt(axis.angle, LABEL_RADIUS)
    const value = ratio === null ? null : pointAt(axis.angle, RADIUS * ratio)
    return { item: axis.item, ratio, value, label, outer }
  })
)

const hasScore = computed(() => vertices.value.some((vertex) => vertex.value !== null))

const polygon = computed(() =>
  vertices.value
    .filter((vertex) => vertex.value !== null)
    .map((vertex) => `${vertex.value!.x.toFixed(1)},${vertex.value!.y.toFixed(1)}`)
    .join(' ')
)

const canFill = computed(
  () => vertices.value.filter((vertex) => vertex.value !== null).length >= 3
)
</script>

<template>
  <div class="score-radar">
    <el-skeleton v-if="loading" :rows="5" animated />

    <div v-else-if="!hasScore" class="score-radar__empty">
      <p>暂无评分数据</p>
    </div>

    <svg v-else class="score-radar__svg" :viewBox="`0 0 ${SIZE} ${SIZE}`">
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
      <polygon
        v-if="canFill"
        :points="polygon"
        class="score-radar__area"
      />
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
  </div>
</template>

<style scoped lang="scss">
.score-radar {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 220px;

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
    fill: color-mix(in srgb, var(--color-primary) 18%, transparent);
    stroke: var(--color-primary);
    stroke-width: 2;
    stroke-linejoin: round;
  }

  &__dot {
    fill: var(--color-primary);

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
