<script setup lang="ts">
import AiScoreHero from '@/components/evaluation/AiScoreHero.vue'
import ScoreDimensionsCard from '@/components/evaluation/ScoreDimensionsCard.vue'
import EvaluationTimeline from '@/components/evaluation/EvaluationTimeline.vue'
import QualityBadge from '@/components/common/QualityBadge.vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { DimensionKey } from '@/types/evaluation'
import type { TeacherSummary, TeacherTimelineItem } from '@/types/teacher'

/**
 * 教师画像详情主体（docs/frontend_AGENTS.md §7.9）。
 * 纵向四段叙事：AI 评价总览 → 五维评分 → 按课程明细 → 历次评价流。
 * 分数一律取 AI 评价（agentScore）；AI 未评时回退后端综合分，缺失分渲染 — 不补 0。
 * 样本与 flags 在总览横带内露出，不再单列卡片。
 */
withDefaults(
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

/** 数字一致性铁律：直接展示后端值，不做前端四舍五入 */
function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

function toCourse(row: unknown): TeacherSummary['courses'][number] {
  return row as TeacherSummary['courses'][number]
}

/** 课程综合分取 AI 分（AI 未评时回退后端综合分，保持与总览一致的口径） */
function courseComposite(course: unknown): number | null {
  const row = toCourse(course)
  return row.agentScore ?? row.compositeScore ?? null
}

function courseDimensionScore(course: unknown, key: DimensionKey): number | null {
  const dimension = toCourse(course).dimensions.find((item) => item.key === key)
  return dimension?.agentScore ?? dimension?.score ?? null
}

function handleCourseClick(row: unknown): void {
  emit('select-course', toCourse(row).courseId)
}
</script>

<template>
  <el-skeleton v-if="loading" :rows="10" animated />

  <div v-else-if="summary" class="teacher-panel">
    <!-- ① 总览：AI 评价综合分 + 样本口径（含 flags 警示） -->
    <AiScoreHero :summary="summary" :semester="summary.semester" />

    <!-- ② 五维评分：AI 单源雷达 + 维度读数 -->
    <section class="teacher-panel__section">
      <ScoreDimensionsCard :summary="summary" />
    </section>

    <!-- ③ 按课程明细 -->
    <section class="teacher-panel__section">
      <h2 class="teacher-panel__section-title">按课程明细</h2>
      <el-table :data="summary.courses" row-key="courseId" stripe @row-click="handleCourseClick">
        <el-table-column prop="courseCode" label="课程编码" width="130" />
        <el-table-column prop="courseName" label="课程名称" min-width="180" />
        <el-table-column label="综合分" width="180" align="right">
          <template #default="{ row }">
            <template v-if="courseComposite(row) === null">
              <span class="teacher-panel__muted">暂无评价</span>
            </template>
            <template v-else>
              <span
                class="teacher-panel__composite score-num"
                :style="{ color: scoreToneColor(scoreTone(courseComposite(row))) }"
              >
                {{ formatScore(courseComposite(row)) }}
              </span>
              <QualityBadge class="teacher-panel__badge" :score="courseComposite(row)" />
            </template>
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

    <!-- ④ 历次评价：竖线评价流 -->
    <section class="teacher-panel__section">
      <h2 class="teacher-panel__section-title">历次评价</h2>
      <EvaluationTimeline :items="timeline" :loading="timelineLoading" />
    </section>
  </div>
</template>

<style scoped lang="scss">
.teacher-panel {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-4);

  &__section {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__section-title {
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__composite {
    margin-right: var(--spacing-2);
    font-weight: 600;
  }

  &__badge {
    margin-left: var(--spacing-2);
  }

  &__muted {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
