<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import type { ScoreSummary } from '@/types/teacher'
import { scoreLevelLabel, scoreTone, scoreToneColor } from '@/utils/format'

/**
 * 双源罗盘 —— 全站「综合分」的统一签名图形（《前端设计-new》§4.4.1）。
 * 外弧 = 督导分（靛蓝），内弧 = AI 分（亮紫），中心 = 综合分 + 等级。
 * 红线：compositeScore === null 时显示灰环「暂无评价」，禁止画 0；
 * 样本量 / flags 必须露出，不得静默。
 */
const props = withDefaults(
  defineProps<{
    summary: ScoreSummary | null
    /** lg: 驾驶舱/画像主位；md: 页内卡片；sm: 缩略快照 */
    size?: 'lg' | 'md' | 'sm'
    loading?: boolean
    title?: string
  }>(),
  { size: 'md', loading: false, title: undefined }
)

const CENTER = 80
const R_HUMAN = 58
const R_AGENT = 44

function polar(deg: number, radius: number): { x: number; y: number } {
  const rad = (deg * Math.PI) / 180
  return { x: CENTER + radius * Math.cos(rad), y: CENTER + radius * Math.sin(rad) }
}

function arcPath(score: number | null, radius: number): string {
  if (score === null || score === undefined || !Number.isFinite(score) || score <= 0) return ''
  const deg = Math.min(((score / 100) * 360), 359.99)
  const start = polar(-90, radius)
  const end = polar(-90 + deg, radius)
  const largeArc = deg > 180 ? 1 : 0
  return `M ${start.x.toFixed(2)} ${start.y.toFixed(2)} A ${radius} ${radius} 0 ${largeArc} 1 ${end.x.toFixed(2)} ${end.y.toFixed(2)}`
}

const humanArc = computed(() => arcPath(props.summary?.supervisorScore ?? null, R_HUMAN))
const aiArc = computed(() => arcPath(props.summary?.agentScore ?? null, R_AGENT))

const composite = computed(() => props.summary?.compositeScore ?? null)
const tone = computed(() => scoreTone(composite.value))
const levelText = computed(() => scoreLevelLabel(tone.value))
const toneColor = computed(() => scoreToneColor(tone.value))
const singleSource = computed(() => {
  const s = props.summary
  if (!s) return false
  const hasHuman = s.supervisorScore !== null
  const hasAgent = s.agentScore !== null
  return hasHuman !== hasAgent
})
const alphaText = computed(() => {
  const w = props.summary?.weights
  return w && Number.isFinite(w.supervisor) ? w.supervisor.toFixed(2) : '0.50'
})

const ariaLabel = computed(() => {
  if (!props.summary) return '暂无评价'
  const s = props.summary
  return `综合分 ${s.compositeScore ?? '暂无'}，督导分 ${s.supervisorScore ?? '暂无'}，AI 分 ${s.agentScore ?? '暂无'}，已评 ${s.sample.evaluatedCount} / ${s.sample.sessionCount} 场`
})

/* 数字进场 count-up（600ms），reduced-motion 直接显示终值 */
const displayScore = ref<number | null>(null)
let rafId = 0

watch(
  composite,
  (target) => {
    if (typeof window === 'undefined') {
      displayScore.value = target
      return
    }
    window.cancelAnimationFrame(rafId)
    if (target === null || target === undefined) {
      displayScore.value = null
      return
    }
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      displayScore.value = target
      return
    }
    const start = performance.now()
    const duration = 600
    const tick = (now: number): void => {
      const t = Math.min(1, (now - start) / duration)
      const eased = 1 - Math.pow(1 - t, 3)
      displayScore.value = target * eased
      if (t < 1) rafId = requestAnimationFrame(tick)
    }
    rafId = requestAnimationFrame(tick)
  },
  { immediate: true }
)

onUnmounted(() => {
  if (typeof window !== 'undefined') window.cancelAnimationFrame(rafId)
})

const scoreText = computed(() => {
  if (composite.value === null) return '—'
  const shown = displayScore.value ?? composite.value
  return shown.toFixed(1)
})

function sideScore(value: number | null): string {
  return value === null || value === undefined ? '—' : value.toFixed(1)
}
</script>

<template>
  <div class="quality-compass" :class="`quality-compass--${size}`">
    <template v-if="loading">
      <div class="quality-compass__skeleton">
        <el-skeleton animated>
          <template #template>
            <el-skeleton-item variant="circle" style="width: 112px; height: 112px" />
            <el-skeleton-item variant="text" style="margin-top: 12px; width: 60%" />
          </template>
        </el-skeleton>
      </div>
    </template>

    <template v-else-if="summary">
      <div class="quality-compass__dial" role="img" :aria-label="ariaLabel">
        <svg viewBox="0 0 160 160" class="quality-compass__svg">
          <circle class="quality-compass__track" :cx="CENTER" :cy="CENTER" :r="R_HUMAN" />
          <circle class="quality-compass__track quality-compass__track--inner" :cx="CENTER" :cy="CENTER" :r="R_AGENT" />
          <path class="quality-compass__arc quality-compass__arc--human" :d="humanArc" />
          <path class="quality-compass__arc quality-compass__arc--ai" :d="aiArc" />
        </svg>
        <div class="quality-compass__center">
          <span class="quality-compass__score score-num" :style="{ color: toneColor }">
            {{ scoreText }}
          </span>
          <span class="quality-compass__level">{{ levelText }}</span>
        </div>
      </div>

      <div v-if="title" class="quality-compass__title">{{ title }}</div>

      <div class="quality-compass__meta">
        <span class="quality-compass__meta-item">
          <span class="quality-compass__dot quality-compass__dot--human" />督导
          {{ sideScore(summary.supervisorScore) }}
        </span>
        <span class="quality-compass__meta-item">
          <span class="quality-compass__dot quality-compass__dot--ai" />AI
          {{ sideScore(summary.agentScore) }}
        </span>
        <span class="quality-compass__meta-alpha">α={{ alphaText }}</span>
      </div>

      <div class="quality-compass__sample">
        <span>
          已评 {{ summary.sample.evaluatedCount }} / {{ summary.sample.sessionCount }} 场 · 对齐
          {{ summary.sample.alignedCount }}
        </span>
        <el-tag v-if="singleSource" size="small" type="info" effect="plain">单源</el-tag>
        <el-tag
          v-if="!summary.sample.sampleSufficient"
          size="small"
          type="warning"
          effect="plain"
        >
          样本不足
        </el-tag>
        <el-tooltip
          v-if="summary.flags.length"
          placement="top"
          :content="summary.flags.join('；')"
        >
          <el-tag size="small" type="warning" effect="plain" class="quality-compass__flags">
            口径提示 {{ summary.flags.length }}
          </el-tag>
        </el-tooltip>
      </div>
    </template>

    <div v-else class="quality-compass__empty">
      <div class="quality-compass__dial" role="img" aria-label="暂无评价">
        <svg viewBox="0 0 160 160" class="quality-compass__svg">
          <circle class="quality-compass__track" :cx="CENTER" :cy="CENTER" :r="R_HUMAN" />
          <circle class="quality-compass__track quality-compass__track--inner" :cx="CENTER" :cy="CENTER" :r="R_AGENT" />
        </svg>
        <div class="quality-compass__center">
          <span class="quality-compass__score quality-compass__score--void score-num">—</span>
          <span class="quality-compass__level">暂无评价</span>
        </div>
      </div>
      <div v-if="title" class="quality-compass__title">{{ title }}</div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.quality-compass {
  display: flex;
  flex-direction: column;
  align-items: center;

  --compass-size: 160px;
  --compass-score-size: var(--font-size-score-xl);

  &--lg {
    --compass-size: 176px;
  }

  &--md {
    --compass-size: 148px;
    --compass-score-size: var(--font-size-score-lg);
  }

  &--sm {
    --compass-size: 116px;
    --compass-score-size: var(--font-size-score-lg);
  }

  &__dial {
    position: relative;
    width: var(--compass-size);
    height: var(--compass-size);
  }

  &__svg {
    width: 100%;
    height: 100%;
    transform: rotate(0deg);
  }

  &__track {
    fill: none;
    stroke: color-mix(in srgb, var(--color-score-void) 26%, #ffffff);
    stroke-width: 10;

    &--inner {
      stroke-width: 8;
      stroke: color-mix(in srgb, var(--color-score-void) 18%, #ffffff);
    }
  }

  &__arc {
    fill: none;
    stroke-linecap: round;
    transition: opacity var(--duration-fast) ease;

    &--human {
      stroke: var(--color-primary);
      stroke-width: 10;
    }

    &--ai {
      stroke: var(--color-ai-bright);
      stroke-width: 8;
    }
  }

  &__center {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
  }

  &__score {
    font-size: var(--compass-score-size);
    line-height: 1;

    &--void {
      color: var(--color-score-void);
    }
  }

  &__level {
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__title {
    margin-top: var(--spacing-3);
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-3);
    justify-content: center;
    margin-top: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-secondary);

    &-item {
      display: inline-flex;
      align-items: center;
      gap: 4px;
    }

    &-alpha {
      color: var(--color-text-tertiary);
    }
  }

  &__dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;

    &--human {
      background-color: var(--color-primary);
    }

    &--ai {
      background-color: var(--color-ai-bright);
    }
  }

  &__sample {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
    justify-content: center;
    margin-top: var(--spacing-2);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__flags {
    cursor: help;
  }

  &__skeleton {
    width: 160px;
    padding: var(--spacing-4);
    text-align: center;
  }

  &__empty {
    display: flex;
    flex-direction: column;
    align-items: center;
  }
}
</style>
