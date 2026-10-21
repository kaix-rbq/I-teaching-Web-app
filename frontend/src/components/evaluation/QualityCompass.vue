<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import type { ScoreSummary } from '@/types/teacher'
import { scoreLevelLabel, scoreTone, scoreToneColor } from '@/utils/format'

/**
 * AI 单源罗盘 —— 画像链路「综合分」的统一签名图形（docs/frontend_AGENTS.md §9.0 / §9.1）。
 * 单一主环 = AI 评价分（亮紫），中心 = AI 综合分 + 等级。
 * 红线：主分缺失时显示单灰环「暂无 AI 评价」，禁止画 0；
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
/** 唯一主环半径：原双环外环已随「AI 单源」决策移除，内环升级为主环并加粗到 10px */
const R_RING = 52

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

/**
 * 主分 = AI 分。画像页不再展示综合口径，因此不再以 compositeScore 为主，
 * 仅当 AI 尚未评价（agentScore 为 null）时回退到后端综合分，避免已有评价却渲染成空环。
 */
const mainScore = computed(() => props.summary?.agentScore ?? props.summary?.compositeScore ?? null)

const aiArc = computed(() => arcPath(mainScore.value, R_RING))

const tone = computed(() => scoreTone(mainScore.value))
/** 主分缺失时统一措辞为「暂无 AI 评价」（沿用等级文案会退化成泛化的「暂无评价」） */
const levelText = computed(() =>
  tone.value === null ? '暂无 AI 评价' : scoreLevelLabel(tone.value)
)
const toneColor = computed(() => scoreToneColor(tone.value))

const ariaLabel = computed(() => {
  if (!props.summary) return '暂无 AI 评价'
  const s = props.summary
  return `AI 综合分 ${mainScore.value ?? '暂无'}，已评 ${s.sample.evaluatedCount} / ${s.sample.sessionCount} 场`
})

/* 数字进场 count-up（600ms），reduced-motion 直接显示终值 */
const displayScore = ref<number | null>(null)
let rafId = 0

watch(
  mainScore,
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
  if (mainScore.value === null) return '—'
  const shown = displayScore.value ?? mainScore.value
  return shown.toFixed(2)
})

/** 环内展示的是 AI 分：AI 未评（回退展示综合分）时该读数显式留空，不误导为 AI 分 */
const aiScoreText = computed(() =>
  props.summary?.agentScore === null || props.summary?.agentScore === undefined
    ? '—'
    : props.summary.agentScore.toFixed(2)
)
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
          <circle class="quality-compass__track" :cx="CENTER" :cy="CENTER" :r="R_RING" />
          <path class="quality-compass__arc" :d="aiArc" />
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
          <span class="quality-compass__dot" />AI {{ aiScoreText }}
        </span>
      </div>

      <div class="quality-compass__sample">
        <span>已评 {{ summary.sample.evaluatedCount }} / {{ summary.sample.sessionCount }} 场</span>
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
      <div class="quality-compass__dial" role="img" aria-label="暂无 AI 评价">
        <svg viewBox="0 0 160 160" class="quality-compass__svg">
          <circle class="quality-compass__track" :cx="CENTER" :cy="CENTER" :r="R_RING" />
        </svg>
        <div class="quality-compass__center">
          <span class="quality-compass__score quality-compass__score--void score-num">—</span>
          <span class="quality-compass__level">暂无 AI 评价</span>
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
  }

  &__track {
    fill: none;
    stroke: color-mix(in srgb, var(--color-score-void) 22%, #ffffff);
    stroke-width: 10;
  }

  /* 唯一主环 = AI 评价分 */
  &__arc {
    fill: none;
    stroke: var(--color-ai-bright);
    stroke-width: 10;
    stroke-linecap: round;
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
  }

  &__dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background-color: var(--color-ai-bright);
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
