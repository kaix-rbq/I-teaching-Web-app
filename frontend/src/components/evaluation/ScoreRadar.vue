<script setup lang="ts">
import { computed } from 'vue'

export interface RadarItem {
  key: string
  name: string
  /** 原始分（默认 1-5）；null = 该评分源无法评价此维度 */
  score: number | null
  /** 双源叠加模式：督导分（null = 督导未评此维度，该轴断开不补 0） */
  supervisorScore?: number | null
  /** 双源叠加模式：AI 分（null = 智能体未评此维度） */
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

/** 双源叠加模式：任一维度携带 supervisor/agent 分即开启（教师画像/质量档案） */
const dualMode = computed(() =>
  props.items.some((item) => item.supervisorScore !== undefined || item.agentScore !== undefined)
)

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

function verticesFor(pick: (item: RadarItem) => number | null | undefined): Vertex[] {
  return axes.value.map((axis) => {
    const ratio = ratioOf(pick(axis.item))
    return {
      item: axis.item,
      value: ratio === null ? null : pointAt(axis.angle, RADIUS * ratio),
      label: pointAt(axis.angle, LABEL_RADIUS),
      outer: pointAt(axis.angle, RADIUS)
    }
  })
}

const vertices = computed(() => verticesFor((item) => item.score))
const supervisorVertices = computed(() => verticesFor((item) => item.supervisorScore ?? null))
const agentVertices = computed(() => verticesFor((item) => item.agentScore ?? null))

function polygonOf(list: Vertex[]): { points: string; canFill: boolean } {
  const filled = list.filter((vertex) => vertex.value !== null)
  return {
    points: filled.map((vertex) => `${vertex.value!.x.toFixed(1)},${vertex.value!.y.toFixed(1)}`).join(' '),
    canFill: filled.length >= 3
  }
}

const polygon = computed(() => polygonOf(vertices.value))
const supervisorPolygon = computed(() => polygonOf(supervisorVertices.value))
const agentPolygon = computed(() => polygonOf(agentVertices.value))

const hasScore = computed(() => {
  if (dualMode.value) {
    return [...supervisorVertices.value, ...agentVertices.value].some(
      (vertex) => vertex.value !== null
    )
  }
  return vertices.value.some((vertex) => vertex.value !== null)
})
</script>

<template>
  <div class="score-radar">
    <el-skeleton v-if="loading" :rows="5" animated />

    <div v-else-if="!hasScore" class="score-radar__empty">
      <p>暂无评分数据</p>
    </div>

    <template v-else>
      <svg v-if="!dualMode" class="score-radar__svg" :viewBox="`0 0 ${SIZE} ${SIZE}`">
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

      <!-- 双源叠加：督导 = 靛蓝实线，AI = 亮紫虚线（AI 视觉语言 §4.2.2） -->
      <svg v-else class="score-radar__svg" :viewBox="`0 0 ${SIZE} ${SIZE}`">
        <polygon
          v-for="(radius, index) in rings"
          :key="index"
          :points="ringPoints(radius)"
          class="score-radar__ring"
        />
        <line
          v-for="vertex in supervisorVertices"
          :key="`axis-${vertex.item.key}`"
          :x1="CENTER"
          :y1="CENTER"
          :x2="vertex.outer.x"
          :y2="vertex.outer.y"
          class="score-radar__axis"
        />
        <polygon
          v-if="agentPolygon.canFill"
          :points="agentPolygon.points"
          class="score-radar__area score-radar__area--ai"
        />
        <polygon
          v-if="supervisorPolygon.canFill"
          :points="supervisorPolygon.points"
          class="score-radar__area"
        />
        <template v-for="vertex in agentVertices" :key="`ai-dot-${vertex.item.key}`">
          <circle
            v-if="vertex.value"
            :cx="vertex.value.x"
            :cy="vertex.value.y"
            :r="vertex.item.isObservation ? 3.5 : 2.5"
            class="score-radar__dot score-radar__dot--ai"
          />
        </template>
        <template v-for="vertex in supervisorVertices" :key="`dot-${vertex.item.key}`">
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
          v-for="vertex in supervisorVertices"
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

      <div v-if="dualMode" class="score-radar__legend">
        <span class="score-radar__legend-item">
          <i class="score-radar__swatch score-radar__swatch--human" />督导
        </span>
        <span class="score-radar__legend-item">
          <i class="score-radar__swatch score-radar__swatch--ai" />AI
        </span>
        <span class="score-radar__legend-hint">中心数值为双源融合分</span>
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.score-radar {
  display: flex;
  flex-direction: column;
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

    &--ai {
      fill: color-mix(in srgb, var(--color-ai-bright) 12%, transparent);
      stroke: var(--color-ai-bright);
      stroke-dasharray: 4 3;
    }
  }

  &__dot {
    fill: var(--color-primary);

    &--ai {
      fill: var(--color-ai-bright);
    }

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

    &-hint {
      color: var(--color-text-tertiary);
    }
  }

  &__swatch {
    display: inline-block;
    width: 18px;
    height: 0;
    border-top: 2px solid var(--color-primary);

    &--ai {
      border-top: 2px dashed var(--color-ai-bright);
    }
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
