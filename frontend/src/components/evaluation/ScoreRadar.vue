<script setup lang="ts">
import { computed } from 'vue'

export interface RadarItem {
  key: string
  name: string
  /** AI 主序列原始分（默认 1-5）；null = 该维度无 AI 分数，该轴断开不补 0 */
  score: number | null
  /** 画像链路：AI 分（null = 智能体未评此维度）；携带时图例以「AI」显形 */
  agentScore?: number | null
  /** 督导评分（1-5）；携带该字段即启用督导辅助序列，null = 该维度无督导分 */
  supervisorScore?: number | null
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

interface SeriesVertex {
  item: RadarItem
  value: { x: number; y: number } | null
}

function ratioOf(value: number | null | undefined): number | null {
  if (value === null || value === undefined || !Number.isFinite(value)) return null
  return Math.max(0, Math.min(1, value / props.max))
}

function verticesOf(series: 'ai' | 'supervisor'): SeriesVertex[] {
  return axes.value.map((axis) => {
    const raw = series === 'ai' ? axis.item.score : axis.item.supervisorScore
    const ratio = ratioOf(raw)
    return {
      item: axis.item,
      value: ratio === null ? null : pointAt(axis.angle, RADIUS * ratio)
    }
  })
}

function polygonOf(vertices: SeriesVertex[]): { points: string; canFill: boolean } {
  const filled = vertices.filter((vertex) => vertex.value !== null)
  return {
    points: filled
      .map((vertex) => `${vertex.value!.x.toFixed(1)},${vertex.value!.y.toFixed(1)}`)
      .join(' '),
    canFill: filled.length >= 3
  }
}

const aiVertices = computed(() => verticesOf('ai'))
const supervisorVertices = computed(() => verticesOf('supervisor'))
const aiPolygon = computed(() => polygonOf(aiVertices.value))
const supervisorPolygon = computed(() => polygonOf(supervisorVertices.value))

const labels = computed(() =>
  axes.value.map((axis) => ({
    item: axis.item,
    label: pointAt(axis.angle, LABEL_RADIUS),
    outer: pointAt(axis.angle, RADIUS)
  }))
)

const hasSupervisorSeries = computed(() =>
  props.items.some((item) => item.supervisorScore !== undefined)
)

/** AI 缺失但督导有分时，督导多边形仍可显示。 */
const hasScore = computed(
  () =>
    aiVertices.value.some((vertex) => vertex.value !== null) ||
    supervisorVertices.value.some((vertex) => vertex.value !== null)
)

const showLegend = computed(() => props.tone === 'ai' || hasSupervisorSeries.value)

/** 双轨时按「AI / 督导」顺序同时显示数值，单轨保持原样。 */
function valueText(item: RadarItem): string {
  if (hasSupervisorSeries.value) {
    return `${item.score ?? '—'} / ${item.supervisorScore ?? '—'}`
  }
  return item.score === null ? '—' : String(item.score)
}
</script>

<template>
  <div class="score-radar" :class="{ 'score-radar--ai': tone === 'ai' }">
    <el-skeleton v-if="loading" :rows="5" animated />

    <div v-else-if="!hasScore" class="score-radar__empty">
      <p>暂无评分数据</p>
    </div>

    <template v-else>
      <svg
        class="score-radar__svg"
        :viewBox="`0 0 ${SIZE} ${SIZE}`"
        role="img"
        aria-label="五维评分雷达图"
      >
        <polygon
          v-for="(radius, index) in rings"
          :key="index"
          :points="ringPoints(radius)"
          class="score-radar__ring"
        />
        <line
          v-for="vertex in labels"
          :key="`axis-${vertex.item.key}`"
          :x1="CENTER"
          :y1="CENTER"
          :x2="vertex.outer.x"
          :y2="vertex.outer.y"
          class="score-radar__axis"
        />

        <!-- 督导辅助序列在下层，AI 主序列在上层，避免遮挡 -->
        <polygon
          v-if="supervisorPolygon.canFill"
          :points="supervisorPolygon.points"
          class="score-radar__area score-radar__area--supervisor"
        />
        <polygon v-if="aiPolygon.canFill" :points="aiPolygon.points" class="score-radar__area" />

        <template v-for="vertex in supervisorVertices" :key="`sup-dot-${vertex.item.key}`">
          <circle
            v-if="vertex.value"
            :cx="vertex.value.x"
            :cy="vertex.value.y"
            :r="vertex.item.isObservation ? 4 : 3"
            class="score-radar__dot score-radar__dot--supervisor"
            :class="{ 'score-radar__dot--observation': vertex.item.isObservation }"
          />
        </template>
        <template v-for="vertex in aiVertices" :key="`dot-${vertex.item.key}`">
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
          v-for="vertex in labels"
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
          v-for="vertex in labels"
          :key="`value-${vertex.item.key}`"
          :x="vertex.label.x"
          :y="vertex.label.y + 13"
          class="score-radar__value tabular-nums"
          text-anchor="middle"
          dominant-baseline="middle"
        >
          {{ valueText(vertex.item) }}
        </text>
      </svg>

      <div v-if="showLegend" class="score-radar__legend">
        <span class="score-radar__legend-item">
          <i class="score-radar__swatch" />AI 评价（紫）
        </span>
        <span v-if="hasSupervisorSeries" class="score-radar__legend-item">
          <i class="score-radar__swatch score-radar__swatch--supervisor" />督导评分（蓝/青）
        </span>
        <span v-if="hasSupervisorSeries" class="score-radar__legend-hint">数值：AI / 督导</span>
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.score-radar {
  --radar-ink: var(--color-primary);
  --radar-supervisor: var(--color-teacher);

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

    &--supervisor {
      fill: color-mix(in srgb, var(--radar-supervisor) 12%, transparent);
      stroke: var(--radar-supervisor);
      stroke-dasharray: 5 3;
    }
  }

  &__dot {
    fill: var(--radar-ink);

    &--observation {
      fill: var(--color-warning);
    }

    &--supervisor {
      fill: var(--radar-supervisor);
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
    gap: var(--spacing-2) var(--spacing-3);
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
    border-top: 2px solid var(--radar-ink);

    &--supervisor {
      border-top-color: var(--radar-supervisor);
      border-top-style: dashed;
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
