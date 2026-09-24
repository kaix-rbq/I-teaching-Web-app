<script setup lang="ts">
import { computed } from 'vue'
import { DIMENSION_ANCHORS, getDimensionAnchor, getScoreLevel, SCORE_LEVELS } from '@/constants'
import type { DimensionKey } from '@/types/evaluation'
import { rawScoreTone, scoreToneColor } from '@/utils/format'

/**
 * 锚点刻度条（《前端设计-new》§4.4.3）——把 1-5 李克特量表外化为「仪器刻度」。
 * 五档刻度联动质量色阶，悬停逐档显示行为锚点（DIMENSION_ANCHORS 已冻结）。
 * 替代 EvaluationForm 原有的 Radio 组；键盘可达（原生 radio）。
 */
const props = withDefaults(
  defineProps<{
    dimensionKey: DimensionKey
    dimensionName: string
    modelValue: number | null
    readonly?: boolean
    /** 隐藏下方锚点速览（紧凑模式） */
    compact?: boolean
  }>(),
  { readonly: false, compact: false }
)

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

/** 升序 1→5 渲染（仪器刻度从左至右） */
const levels = computed(() => [...SCORE_LEVELS].sort((a, b) => a.value - b.value))

function toneColor(value: number): string {
  return scoreToneColor(rawScoreTone(value))
}

function anchorContent(value: number): string {
  const level = getScoreLevel(value)
  return `${value} 分 · ${level?.level ?? ''}：${getDimensionAnchor(props.dimensionKey, value)}`
}

const hasAnchor = computed(() => Boolean(DIMENSION_ANCHORS[props.dimensionKey]))
</script>

<template>
  <div class="anchor-scale" role="radiogroup" :aria-label="`${dimensionName} 评分刻度`">
    <div class="anchor-scale__track">
      <label
        v-for="level in levels"
        :key="level.value"
        class="anchor-scale__option"
        :class="{
          'anchor-scale__option--active': modelValue === level.value,
          'anchor-scale__option--readonly': readonly
        }"
        :style="{ '--slot-tone': toneColor(level.value) }"
      >
        <input
          type="radio"
          class="anchor-scale__input"
          :name="`anchor-${dimensionKey}`"
          :value="level.value"
          :checked="modelValue === level.value"
          :disabled="readonly"
          @change="emit('update:modelValue', level.value)"
        >
        <span class="anchor-scale__node" aria-hidden="true" />
        <span class="anchor-scale__num">{{ level.value }}</span>
        <span class="anchor-scale__level">{{ level.level }}</span>
        <span v-if="hasAnchor" class="anchor-scale__tip" aria-hidden="true">
          {{ anchorContent(level.value) }}
        </span>
      </label>
    </div>
  </div>
</template>

<style scoped lang="scss">
.anchor-scale {
  &__track {
    position: relative;
    display: flex;
    justify-content: space-between;
    padding: 0 14px;
    margin-top: var(--spacing-2);

    /* 刻度基线 */
    &::before {
      position: absolute;
      top: 12px;
      left: 14px;
      right: 14px;
      height: 2px;
      content: '';
      background: linear-gradient(
        90deg,
        var(--color-score-1),
        var(--color-score-3),
        var(--color-score-5)
      );
      border-radius: 999px;
      opacity: 0.28;
    }
  }

  &__option {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 3px;
    align-items: center;
    width: 64px;
    cursor: pointer;

    &--readonly {
      cursor: default;
    }

    &:hover .anchor-scale__node {
      box-shadow: 0 0 0 4px color-mix(in srgb, var(--slot-tone) 24%, transparent);
    }
  }

  &__input {
    position: absolute;
    width: 1px;
    height: 1px;
    margin: -1px;
    clip: rect(0 0 0 0);
    overflow: hidden;
    pointer-events: auto;

    &:focus-visible + .anchor-scale__node {
      box-shadow: 0 0 0 4px color-mix(in srgb, var(--slot-tone) 40%, transparent);
    }
  }

  &__node {
    width: 14px;
    height: 14px;
    background: #ffffff;
    border: 2.5px solid var(--slot-tone);
    border-radius: 50%;
    transition: transform var(--duration-fast) var(--ease-out-soft);
  }

  &__option--active &__node {
    background: var(--slot-tone);
    transform: scale(1.28);
  }

  &__num {
    font-family: var(--font-score);
    font-size: var(--font-size-sm);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    color: var(--color-text-tertiary);
  }

  &__option--active &__num {
    color: var(--slot-tone);
  }

  &__level {
    font-size: 10px;
    color: var(--color-text-disabled);
  }

  /* 无 tooltip 组件依赖：hover 展示锚点卡（锚点是评分依据，必须始终可读） */
  &__tip {
    position: absolute;
    bottom: calc(100% + 8px);
    left: 50%;
    z-index: 20;
    width: 232px;
    padding: var(--spacing-2) var(--spacing-3);
    font-size: var(--font-size-xs);
    font-weight: 400;
    line-height: 1.6;
    color: var(--color-bg-card);
    text-align: left;
    pointer-events: none;
    background: var(--color-ink);
    border-radius: var(--radius-md);
    opacity: 0;
    transform: translateX(-50%) translateY(4px);
    transition: opacity var(--duration-fast) var(--ease-out-soft);
  }

  &__option:hover &__tip {
    opacity: 1;
    transform: translateX(-50%) translateY(0);
  }
}
</style>
