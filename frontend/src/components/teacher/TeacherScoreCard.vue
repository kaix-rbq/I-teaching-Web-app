<script setup lang="ts">
import { computed } from 'vue'
import QualityCompass from '@/components/evaluation/QualityCompass.vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { TeacherScoreItem } from '@/types/teacher'

/**
 * 教师质量卡片（《前端设计-new》§5.6 列表页默认视图）。
 * 三件套：罗盘缩略 + 五维迷你条 + 样本徽章；null → 灰环「暂无评价」，禁止以 0 充当。
 */
const props = withDefaults(
  defineProps<{
    item: TeacherScoreItem
  }>(),
  {}
)

const emit = defineEmits<{ (e: 'view', item: TeacherScoreItem): void }>()

const dimBars = computed(() =>
  EVALUATION_DIMENSIONS.map((meta) => {
    const dimension = props.item.dimensions.find((entry) => entry.key === meta.key)
    const score = dimension?.score ?? null
    return {
      key: meta.key,
      name: meta.shortName,
      isObservation: meta.isObservation,
      score,
      ratio: score === null ? 0 : Math.max(0, Math.min(100, score)),
      color: scoreToneColor(scoreTone(score))
    }
  })
)

function formatScore(score: number | null): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}
</script>

<template>
  <article class="teacher-card" @click="emit('view', item)">
    <header class="teacher-card__head">
      <span class="teacher-card__name">{{ item.teacherName }}</span>
      <span class="teacher-card__jobno">{{ item.jobNo }}</span>
      <el-tag
        v-if="item.compositeScore !== null && !item.sample.sampleSufficient"
        class="teacher-card__warn"
        type="warning"
        size="small"
        effect="plain"
        round
      >
        样本不足
      </el-tag>
    </header>

    <QualityCompass :summary="item" size="sm" />

    <ul class="teacher-card__dims">
      <li v-for="bar in dimBars" :key="bar.key" class="teacher-card__dim">
        <span class="teacher-card__dim-name" :title="bar.isObservation ? '观测项，不计入加权' : ''">
          {{ bar.name }}
        </span>
        <span class="teacher-card__dim-track">
          <i
            class="teacher-card__dim-fill"
            :class="{ 'teacher-card__dim-fill--void': bar.score === null }"
            :style="{ width: `${bar.ratio}%`, backgroundColor: bar.color }"
          />
        </span>
        <span class="teacher-card__dim-value tabular-nums">{{ formatScore(bar.score) }}</span>
      </li>
    </ul>

    <footer class="teacher-card__foot">
      <span class="teacher-card__sample tabular-nums">n={{ item.sample.evaluatedCount }}</span>
      <el-tag
        v-if="item.compositeScore !== null && item.sample.sampleSufficient"
        type="success"
        size="small"
        effect="plain"
        round
      >
        样本充足
      </el-tag>
      <span class="teacher-card__cta">查看画像 →</span>
    </footer>
  </article>
</template>

<style scoped lang="scss">
.teacher-card {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
  padding: var(--spacing-4);
  cursor: pointer;
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  transition: border-color var(--duration-fast) var(--ease-out-soft),
    box-shadow var(--duration-fast) var(--ease-out-soft),
    transform var(--duration-fast) var(--ease-out-soft);

  &:hover,
  &:focus-visible {
    border-color: var(--color-primary-light);
    box-shadow: var(--shadow-card-hover, var(--shadow-card));
    transform: translateY(-2px);
  }

  &__head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
  }

  &__name {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__jobno {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__warn {
    margin-left: auto;
  }

  &__dims {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-1);
    width: 100%;
    padding: 0;
    margin: 0;
    list-style: none;
  }

  &__dim {
    display: grid;
    grid-template-columns: 56px 1fr 40px;
    gap: var(--spacing-2);
    align-items: center;
    font-size: var(--font-size-xs);
    color: var(--color-text-secondary);
  }

  &__dim-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__dim-track {
    position: relative;
    height: 6px;
    overflow: hidden;
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-sm);
  }

  &__dim-fill {
    position: absolute;
    inset: 0 auto 0 0;
    border-radius: var(--radius-sm);

    &--void {
      background-color: var(--color-score-void);
    }
  }

  &__dim-value {
    text-align: right;
    color: var(--color-text-secondary);
  }

  &__foot {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    padding-top: var(--spacing-2);
    margin-top: var(--spacing-1);
    border-top: 1px dashed var(--color-divider);
  }

  &__sample {
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--color-text-secondary);
  }

  &__cta {
    margin-left: auto;
    font-size: var(--font-size-xs);
    color: var(--color-primary);
    opacity: 0;
    transition: opacity var(--duration-fast) var(--ease-out-soft);
  }

  &:hover &__cta,
  &:focus-visible &__cta {
    opacity: 1;
  }
}
</style>
