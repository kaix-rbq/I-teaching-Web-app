<script setup lang="ts">
import { computed } from 'vue'
import { scoreLevelLabel, scoreTone, scoreToneColor } from '@/utils/format'

/**
 * 质量等级徽章：分数旁的必挂件（色彩即等级）。
 * 规范见 docs/前端设计-new.md §4.4 / §8：
 * - null → 灰色「暂无评价」，禁止以 0 充当；
 * - 色阶只用于分数及其派生物，禁止作装饰。
 */
const props = withDefaults(
  defineProps<{
    /** 连续分（0-100）；null/undefined 显示「暂无评价」 */
    score?: number | null
    /** 等级文案覆盖（默认按色阶档位推导） */
    level?: string
    /** 是否展示分数数值 */
    showScore?: boolean
  }>(),
  { score: null, level: undefined, showScore: false }
)

const tone = computed(() => scoreTone(props.score))
const label = computed(() => props.level ?? scoreLevelLabel(tone.value))
const toneColor = computed(() => scoreToneColor(tone.value))
const scoreText = computed(() =>
  props.score === null || props.score === undefined ? null : Number(props.score).toFixed(1)
)
</script>

<template>
  <span
    class="quality-badge"
    :class="{ 'quality-badge--void': tone === null }"
    :style="{ '--badge-tone': toneColor }"
  >
    <span v-if="showScore && scoreText" class="quality-badge__score score-num">{{
      scoreText
    }}</span>
    <span class="quality-badge__level">{{ label }}</span>
  </span>
</template>

<style scoped lang="scss">
.quality-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 10px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--badge-tone) 14%, #ffffff);
  color: var(--badge-tone);
  font-size: var(--font-size-xs);
  font-weight: 600;
  line-height: 1.4;
  white-space: nowrap;
}

.quality-badge--void {
  background: color-mix(in srgb, var(--badge-tone) 18%, #ffffff);
}

.quality-badge__score {
  font-size: var(--font-size-sm);
  line-height: 1;
}
</style>
