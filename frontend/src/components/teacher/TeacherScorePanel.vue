<script setup lang="ts">
import { computed } from 'vue'
import QualityCompass from '@/components/evaluation/QualityCompass.vue'
import ScoreDimensionsCard from '@/components/evaluation/ScoreDimensionsCard.vue'
import EvaluationTimeline from '@/components/evaluation/EvaluationTimeline.vue'
import QualityBadge from '@/components/common/QualityBadge.vue'
import { EVALUATION_DIMENSIONS } from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { DimensionKey } from '@/types/evaluation'
import type { TeacherSummary, TeacherTimelineItem } from '@/types/teacher'

/**
 * 教师画像详情主体（《前端设计-new》§5.6）。
 * 三卡横排（罗盘 | 双源雷达 | 样本口径）→ 按课程明细 → 评价流时间线。
 * 铁律：缺失分渲染 — 不补 0；样本与 flags 必露；AI 评价以 AI 语法显形。
 */
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

/** 数字一致性铁律：直接展示后端值，不做前端四舍五入 */
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
    <!-- 三卡横排：双源罗盘 | 五维雷达（双源叠加）| 样本与口径 -->
    <section class="teacher-panel__top">
      <div class="teacher-panel__card">
        <h2 class="teacher-panel__card-title">综合分 · 双源罗盘</h2>
        <QualityCompass :summary="summary" size="lg" />
      </div>

      <div class="teacher-panel__card">
        <ScoreDimensionsCard :summary="summary" />
      </div>

      <div class="teacher-panel__card">
        <h2 class="teacher-panel__card-title">样本与口径</h2>
        <dl class="teacher-panel__facts">
          <div>
            <dt>已评场次</dt>
            <dd class="tabular-nums">
              {{ summary.sample.evaluatedCount }} / {{ summary.sample.sessionCount }}
            </dd>
          </div>
          <div>
            <dt>评价来源</dt>
            <dd class="tabular-nums">督导 {{ summary.sample.supervisorCount }} · AI {{ summary.sample.agentCount }}</dd>
          </div>
          <div>
            <dt>双源对齐</dt>
            <dd class="tabular-nums">{{ summary.sample.alignedCount }} 场</dd>
          </div>
          <div>
            <dt>融合权重</dt>
            <dd>
              督导 {{ Math.round(summary.weights.supervisor * 100) }}% / 智能体
              {{ Math.round(summary.weights.agent * 100) }}%
            </dd>
          </div>
          <div>
            <dt>口径版本</dt>
            <dd>{{ summary.formulaVersion || '—' }}</dd>
          </div>
        </dl>
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
        <p v-else class="teacher-panel__caliber-ok">样本与口径无异常提示</p>
      </div>
    </section>

    <!-- 按课程明细 -->
    <section class="teacher-panel__block">
      <h2 class="teacher-panel__title">按课程明细</h2>
      <el-table :data="summary.courses" row-key="courseId" stripe @row-click="handleCourseClick">
        <el-table-column prop="courseCode" label="课程编码" width="130" />
        <el-table-column prop="courseName" label="课程名称" min-width="180" />
        <el-table-column label="综合分" width="180" align="right">
          <template #default="{ row }">
            <template v-if="row.compositeScore === null">
              <span class="teacher-panel__muted">暂无评价</span>
            </template>
            <template v-else>
              <span
                class="teacher-panel__composite score-num"
                :style="{ color: scoreToneColor(scoreTone(row.compositeScore)) }"
              >
                {{ formatScore(row.compositeScore) }}
              </span>
              <QualityBadge class="teacher-panel__badge" :score="row.compositeScore" />
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

    <!-- 历次评价：竖线评价流 -->
    <section class="teacher-panel__block">
      <h2 class="teacher-panel__title">历次评价</h2>
      <EvaluationTimeline :items="timeline" :loading="timelineLoading" />
    </section>
  </template>
</template>

<style scoped lang="scss">
.teacher-panel {
  &__top {
    display: grid;
    grid-template-columns: minmax(260px, 1fr) minmax(320px, 1.5fr) minmax(260px, 1fr);
    gap: var(--spacing-4);
    margin-bottom: var(--spacing-4);

    @media (max-width: 1280px) {
      grid-template-columns: 1fr 1fr;
    }

    @media (max-width: 960px) {
      grid-template-columns: 1fr;
    }
  }

  &__card {
    display: flex;
    flex-direction: column;
    padding: var(--spacing-5);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__card-title {
    margin-bottom: var(--spacing-3);
    padding-bottom: var(--spacing-2);
    font-size: var(--font-size-base);
    color: var(--color-text-primary);
    border-bottom: 1px solid var(--color-divider);
  }

  &__facts {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);

    dt {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    dd {
      font-size: var(--font-size-sm);
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

  &__caliber-ok {
    margin-top: var(--spacing-4);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

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
