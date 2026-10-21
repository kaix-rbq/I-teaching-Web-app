<script setup lang="ts">
import { computed } from 'vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { TeacherScoreItem } from '@/types/teacher'

/**
 * 教师名录行（画像列表默认视图）。
 * 只保留决策所需的一行：姓名、部门、AI 分、样本、薄弱维度。
 * 罗盘与五维详图留在详情页，列表不重复呈现。
 */
const props = withDefaults(
  defineProps<{
    item: TeacherScoreItem
    department?: string
  }>(),
  { department: '' }
)

const emit = defineEmits<{ (e: 'view', item: TeacherScoreItem): void }>()

const aiScore = computed(() => props.item.agentScore ?? props.item.compositeScore ?? null)
const tone = computed(() => scoreTone(aiScore.value))
const scoreColor = computed(() => scoreToneColor(tone.value))

function formatScore(score: number | null): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

/** 薄弱维度取可计分维度中的最低分；无数据时留空，不虚构 */
const weakest = computed(() => {
  const rows = props.item.dimensions
    .filter((entry) => !entry.isObservation)
    .map((entry) => ({
      name: entry.name,
      shortName: EVALUATION_DIMENSIONS.find((meta) => meta.key === entry.key)?.shortName ?? entry.name,
      score: entry.agentScore ?? entry.score ?? null
    }))
    .filter((entry): entry is { name: string; shortName: string; score: number } => entry.score !== null)
  if (!rows.length) return null
  return rows.reduce((min, entry) => (entry.score < min.score ? entry : min))
})

function open(): void {
  emit('view', props.item)
}
</script>

<template>
  <article
    class="teacher-row"
    role="link"
    tabindex="0"
    :aria-label="`查看 ${item.teacherName} 的教师画像`"
    @click="open"
    @keydown.enter.prevent="open"
    @keydown.space.prevent="open"
  >
    <div class="teacher-row__identity">
      <span class="teacher-row__name">{{ item.teacherName }}</span>
      <span class="teacher-row__jobno tabular-nums">{{ item.jobNo }}</span>
    </div>
    <span class="teacher-row__department">{{ department || '—' }}</span>
    <span class="teacher-row__score score-num" :style="{ color: scoreColor }">
      {{ formatScore(aiScore) }}
    </span>
    <span class="teacher-row__sample tabular-nums">
      n={{ item.sample.evaluatedCount }}
      <em v-if="!item.sample.sampleSufficient">样本不足</em>
    </span>
    <span class="teacher-row__weak">
      {{ weakest ? `薄弱 ${weakest.shortName}` : '暂无可计分维度' }}
    </span>
  </article>
</template>

<style scoped lang="scss">
.teacher-row {
  display: grid;
  grid-template-columns: minmax(140px, 1.1fr) minmax(120px, 1fr) 72px 132px minmax(120px, 1.1fr);
  gap: var(--spacing-4);
  align-items: center;
  padding: var(--spacing-4) var(--spacing-2);
  cursor: pointer;
  border-top: 1px solid var(--color-divider);
  transition: background-color var(--duration-fast) var(--ease-out-soft);

  &:hover {
    background-color: var(--color-bg-page);
  }

  &:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: -2px;
    border-radius: var(--radius-sm);
  }

  &__identity {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: baseline;
  }

  &__name {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__jobno {
    font-family: var(--font-score);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__department {
    overflow: hidden;
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__score {
    font-size: var(--font-size-score-md);
    text-align: right;
  }

  &__sample {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);

    em {
      margin-left: var(--spacing-1);
      font-style: normal;
      color: var(--color-warning);
    }
  }

  &__weak {
    overflow: hidden;
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }
}

@media (max-width: 1080px) {
  .teacher-row {
    grid-template-columns: minmax(120px, 1fr) 64px 116px minmax(100px, 1fr);

    &__department {
      display: none;
    }
  }
}

@media (max-width: 840px) {
  .teacher-row {
    grid-template-columns: minmax(110px, 1fr) 56px 96px;

    &__weak {
      display: none;
    }
  }
}
</style>
