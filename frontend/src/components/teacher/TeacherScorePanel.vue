<script setup lang="ts">
import { computed } from 'vue'
import ScoreRadar from '@/components/evaluation/ScoreRadar.vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import { formatDate } from '@/utils/format'
import type { DimensionKey } from '@/types/evaluation'
import type { TeacherSummary, TeacherTimelineItem } from '@/types/teacher'

const props = withDefaults(
  defineProps<{
    summary: TeacherSummary | null
    timeline: TeacherTimelineItem[]
    loading?: boolean
    timelineLoading?: boolean
  }>(),
  {
    loading: false,
    timelineLoading: false
  }
)

const emit = defineEmits<{ (e: 'select-course', courseId: number): void }>()

const FLAG_TEXT: Record<string, string> = {
  no_data: '暂无评价',
  sample_insufficient: '样本不足，当前分数代表性有限',
  disjoint: '督导与智能体评价尚未对齐，当前分数代表性有限',
  formula_mixed: '口径版本混杂，请谨慎解读'
}

const flags = computed(() => props.summary?.flags ?? [])

const radarItems = computed(() =>
  (props.summary?.dimensions ?? []).map((dimension) => ({
    key: dimension.key,
    name: dimension.name,
    score: dimension.score,
    isObservation: dimension.isObservation
  }))
)

const dimensionRows = computed(() =>
  EVALUATION_DIMENSIONS.map((meta) => {
    const dimension = props.summary?.dimensions.find((item) => item.key === meta.key)
    const score = dimension?.score ?? null
    return {
      ...meta,
      score,
      ratio: score === null ? 0 : Math.max(0, Math.min(100, score))
    }
  })
)

function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

function toCourse(row: unknown): TeacherSummary['courses'][number] {
  return row as TeacherSummary['courses'][number]
}

function courseDimensionScore(course: unknown, key: DimensionKey): number | null {
  return toCourse(course).dimensions.find((dimension) => dimension.key === key)?.score ?? null
}

function handleCourseClick(row: unknown): void {
  emit('select-course', toCourse(row).courseId)
}

function flagText(flag: string): string {
  return FLAG_TEXT[flag] ?? flag
}
</script>

<template>
  <el-skeleton v-if="loading" :rows="10" animated />

  <template v-else-if="summary">
    <!-- 评分总览 -->
    <section class="teacher-panel__block">
      <div class="teacher-panel__overview">
        <div class="teacher-panel__composite">
          <span class="teacher-panel__composite-label">综合分</span>
          <strong class="teacher-panel__composite-value tabular-nums">
            {{ formatScore(summary.compositeScore) }}
          </strong>
          <el-tag
            v-if="!summary.sample.sampleSufficient"
            type="warning"
            effect="light"
            round
          >
            样本不足（n={{ summary.sample.evaluatedCount }}）
          </el-tag>
        </div>

        <dl class="teacher-panel__sides">
          <div>
            <dt>督导分</dt>
            <dd class="tabular-nums">{{ formatScore(summary.supervisorScore) }}</dd>
          </div>
          <div>
            <dt>智能体分</dt>
            <dd class="tabular-nums">{{ formatScore(summary.agentScore) }}</dd>
          </div>
          <div>
            <dt>融合权重</dt>
            <dd>
              督导 {{ Math.round(summary.weights.supervisor * 100) }}% / 智能体
              {{ Math.round(summary.weights.agent * 100) }}%
            </dd>
          </div>
          <div>
            <dt>样本场次</dt>
            <dd class="tabular-nums">
              已评价 {{ summary.sample.evaluatedCount }} / 共 {{ summary.sample.sessionCount }}
            </dd>
          </div>
        </dl>
      </div>

      <div v-if="flags.length" class="teacher-panel__flags">
        <el-alert
          v-for="flag in flags"
          :key="flag"
          :title="flagText(flag)"
          type="warning"
          :closable="false"
          show-icon
        />
      </div>
    </section>

    <!-- 五维评分 -->
    <section class="teacher-panel__block">
      <h2 class="teacher-panel__title">五维评分</h2>
      <div class="teacher-panel__dimensions">
        <div class="teacher-panel__radar">
          <ScoreRadar :items="radarItems" :max="100" />
        </div>
        <div class="teacher-panel__bars">
          <div v-for="row in dimensionRows" :key="row.key" class="teacher-panel__bar">
            <div class="teacher-panel__bar-head">
              <span class="teacher-panel__bar-name">
                {{ row.name }}
                <el-tag v-if="row.isObservation" type="warning" size="small" effect="light" round>
                  观测项
                </el-tag>
                <el-tag v-else type="info" size="small" effect="plain" round>
                  {{ Math.round(row.weight * 100) }}%
                </el-tag>
              </span>
              <span class="teacher-panel__bar-value tabular-nums">
                {{ formatScore(row.score) }}
              </span>
            </div>
            <div class="teacher-panel__bar-track">
              <div
                class="teacher-panel__bar-fill"
                :class="{ 'teacher-panel__bar-fill--observation': row.isObservation }"
                :style="{ width: `${row.ratio}%` }"
              />
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- 按课程明细 -->
    <section class="teacher-panel__block">
      <h2 class="teacher-panel__title">按课程明细</h2>
      <el-table :data="summary.courses" row-key="courseId" stripe @row-click="handleCourseClick">
        <el-table-column prop="courseCode" label="课程编码" width="130" />
        <el-table-column prop="courseName" label="课程名称" min-width="180" />
        <el-table-column label="综合分" width="110" align="right">
          <template #default="{ row }">
            <span class="tabular-nums">{{ formatScore(row.compositeScore) }}</span>
          </template>
        </el-table-column>
        <el-table-column
          v-for="dimension in EVALUATION_DIMENSIONS.filter((item) => !item.isObservation)"
          :key="dimension.key"
          :label="dimension.shortName"
          width="100"
          align="right"
        >
          <template #default="{ row }">
            <span class="tabular-nums">{{ formatScore(courseDimensionScore(row, dimension.key)) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="评价次数" width="100" align="center">
          <template #default="{ row }">
            <span class="tabular-nums">{{ row.sample.evaluatedCount }}</span>
          </template>
        </el-table-column>
        <template #empty>
          <span class="teacher-panel__muted">本学期暂无课程评分</span>
        </template>
      </el-table>
    </section>

    <!-- 历次评价时间线 -->
    <section class="teacher-panel__block">
      <h2 class="teacher-panel__title">历次评价</h2>
      <el-skeleton v-if="timelineLoading" :rows="4" animated />
      <template v-else>
        <article
          v-for="item in timeline"
          :key="item.sessionId"
          class="teacher-panel__timeline-item"
        >
          <header class="teacher-panel__timeline-head">
            <div class="teacher-panel__timeline-when">
              <span class="teacher-panel__timeline-date">{{ formatDate(item.sessionDate) }}</span>
              <span class="teacher-panel__timeline-meta">
                {{ item.courseName }} · {{ item.period || '—' }}
                <template v-if="item.topic"> · {{ item.topic }}</template>
              </span>
            </div>
            <span class="teacher-panel__timeline-score tabular-nums">
              {{ formatScore(item.compositeScore) }}
            </span>
          </header>

          <div
            v-for="evaluation in item.supervisorEvaluations"
            :key="`${evaluation.evaluatorType}-${evaluation.evaluatorId}`"
            class="teacher-panel__comment"
          >
            <div class="teacher-panel__comment-who">
              {{ evaluation.evaluatorName || '督导' }}
              <span class="teacher-panel__comment-time">{{ formatDate(evaluation.updatedAt) }}</span>
            </div>
            <dl v-if="evaluation.highlights" class="teacher-panel__comment-row">
              <dt>亮点</dt>
              <dd>{{ evaluation.highlights }}</dd>
            </dl>
            <dl v-if="evaluation.improvements" class="teacher-panel__comment-row">
              <dt>待改进</dt>
              <dd>{{ evaluation.improvements }}</dd>
            </dl>
            <dl v-if="evaluation.suggestions" class="teacher-panel__comment-row">
              <dt>建议</dt>
              <dd>{{ evaluation.suggestions }}</dd>
            </dl>
            <dl v-if="evaluation.comment" class="teacher-panel__comment-row">
              <dt>总体</dt>
              <dd>{{ evaluation.comment }}</dd>
            </dl>
          </div>

          <p
            v-if="item.supervisorEvaluations.length === 0 && !item.agentEvaluation"
            class="teacher-panel__muted"
          >
            本次课暂无评价内容
          </p>
        </article>

        <p v-if="timeline.length === 0" class="teacher-panel__muted">本学期暂无评价记录</p>
      </template>
    </section>
  </template>
</template>

<style scoped lang="scss">
.teacher-panel {
  &__block {
    padding: var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__title {
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__overview {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-6);
  }

  &__composite {
    display: flex;
    align-items: center;
    gap: var(--spacing-3);
  }

  &__composite-label {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__composite-value {
    font-size: var(--font-size-stat);
    font-weight: 600;
    color: var(--color-primary);
  }

  &__sides {
    display: grid;
    grid-template-columns: repeat(2, minmax(180px, 1fr));
    gap: var(--spacing-3) var(--spacing-6);

    dt {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    dd {
      font-size: var(--font-size-base);
      font-weight: 500;
      color: var(--color-text-primary);
    }
  }

  &__flags {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    margin-top: var(--spacing-4);
  }

  &__dimensions {
    display: grid;
    grid-template-columns: 300px 1fr;
    gap: var(--spacing-6);
    align-items: center;

    @media (max-width: 1280px) {
      grid-template-columns: 1fr;
    }
  }

  &__bar {
    margin-bottom: var(--spacing-3);

    &:last-child {
      margin-bottom: 0;
    }
  }

  &__bar-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-1);
  }

  &__bar-name {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__bar-value {
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__bar-track {
    height: 8px;
    overflow: hidden;
    background-color: var(--color-divider);
    border-radius: var(--radius-sm);
  }

  &__bar-fill {
    height: 100%;
    background-color: var(--color-primary);
    border-radius: var(--radius-sm);
    transition: width 0.3s ease;

    &--observation {
      background-color: var(--color-warning);
    }
  }

  &__timeline-item {
    padding: var(--spacing-4);
    margin-bottom: var(--spacing-3);
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);

    &:last-of-type {
      margin-bottom: 0;
    }
  }

  &__timeline-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-4);
    margin-bottom: var(--spacing-2);
  }

  &__timeline-when {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  &__timeline-date {
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__timeline-meta {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__timeline-score {
    font-size: var(--font-size-xl);
    font-weight: 600;
    color: var(--color-primary);
  }

  &__comment {
    padding-top: var(--spacing-2);
    margin-top: var(--spacing-2);
    border-top: 1px dashed var(--color-divider);
  }

  &__comment-who {
    margin-bottom: var(--spacing-1);
    font-size: var(--font-size-sm);
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  &__comment-time {
    margin-left: var(--spacing-2);
    font-size: var(--font-size-xs);
    font-weight: 400;
    color: var(--color-text-tertiary);
  }

  &__comment-row {
    display: flex;
    gap: var(--spacing-2);
    margin-top: var(--spacing-1);
    font-size: var(--font-size-sm);

    dt {
      flex-shrink: 0;
      width: 52px;
      color: var(--color-text-tertiary);
    }

    dd {
      flex: 1;
      line-height: 1.6;
      color: var(--color-text-secondary);
    }
  }

  &__muted {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
