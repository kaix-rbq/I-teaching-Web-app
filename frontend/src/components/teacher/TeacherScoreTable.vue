<script setup lang="ts">
import { EVALUATION_DIMENSIONS } from '@/constants'
import type { DimensionKey } from '@/types/evaluation'
import type { TeacherScoreItem } from '@/types/teacher'

withDefaults(
  defineProps<{
    rows: TeacherScoreItem[]
    loading?: boolean
  }>(),
  {
    loading: false
  }
)

const emit = defineEmits<{ (e: 'view', item: TeacherScoreItem): void }>()

/** 计分维度（frontier 为观测项，单列处理） */
const scoredDimensions = EVALUATION_DIMENSIONS.filter((item) => !item.isObservation)

function toItem(row: unknown): TeacherScoreItem {
  return row as TeacherScoreItem
}

function dimensionScore(item: TeacherScoreItem, key: DimensionKey): number | null {
  const dimension = item.dimensions.find((entry) => entry.key === key)
  return dimension?.agentScore ?? dimension?.score ?? null
}

function teacherScore(item: TeacherScoreItem): number | null {
  return item.agentScore ?? item.compositeScore ?? null
}

/** 缺失分一律渲染 —，禁止显示 0（前端 AGENTS §9.6） */
function formatScore(score: number | null): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

/** 综合分为 null 时置底，且不按 0 参与排序（§7.8） */
function sortByComposite(a: TeacherScoreItem, b: TeacherScoreItem): number {
  const left = teacherScore(a)
  const right = teacherScore(b)
  if (left === null && right === null) return 0
  if (left === null) return 1
  if (right === null) return -1
  return left - right
}

function handleRowClick(row: TeacherScoreItem): void {
  emit('view', row)
}
</script>

<template>
  <el-skeleton v-if="loading" :rows="6" animated />
  <el-table
    v-else
    class="teacher-score-table"
    :data="rows"
    row-key="teacherId"
    stripe
    @row-click="handleRowClick"
  >
    <el-table-column prop="teacherName" label="教师" min-width="130" fixed />
    <el-table-column prop="jobNo" label="工号" width="120" />
    <el-table-column label="AI 分" width="160" sortable :sort-method="sortByComposite">
      <template #default="{ row }">
        <template v-if="teacherScore(toItem(row)) === null">
          <span class="teacher-score-table__muted">暂无 AI 评价</span>
        </template>
        <template v-else>
          <span class="teacher-score-table__composite tabular-nums">
            {{ formatScore(teacherScore(toItem(row))) }}
          </span>
          <el-tag
            v-if="!row.sample.sampleSufficient"
            class="teacher-score-table__sample"
            type="warning"
            size="small"
            effect="light"
            round
          >
            样本不足（n={{ row.sample.evaluatedCount }}）
          </el-tag>
        </template>
      </template>
    </el-table-column>

    <el-table-column
      v-for="dimension in scoredDimensions"
      :key="dimension.key"
      :label="dimension.shortName"
      width="100"
      align="right"
    >
      <template #default="{ row }">
        <span class="tabular-nums">{{ formatScore(dimensionScore(toItem(row), dimension.key)) }}</span>
      </template>
    </el-table-column>

    <el-table-column label="前沿交叉" width="110" align="center">
      <template #default="{ row }">
        <el-tag
          v-if="dimensionScore(toItem(row), 'frontier') !== null"
          type="warning"
          size="small"
          effect="light"
          round
        >
          亮点
        </el-tag>
        <span v-else class="teacher-score-table__muted">—</span>
      </template>
    </el-table-column>

    <el-table-column label="评价次数" width="100" align="center">
      <template #default="{ row }">
        <span class="tabular-nums">{{ row.sample.evaluatedCount }}</span>
      </template>
    </el-table-column>

    <el-table-column label="操作" width="90" align="center">
      <template #default="{ row }">
        <el-button link type="primary" @click.stop="emit('view', toItem(row))">查看</el-button>
      </template>
    </el-table-column>

    <template #empty>
      <span class="teacher-score-table__empty">暂无教师评价数据</span>
    </template>
  </el-table>
</template>

<style scoped lang="scss">
.teacher-score-table {
  width: 100%;

  &__composite {
    font-weight: 600;
    color: var(--color-primary);
  }

  &__sample {
    margin-left: var(--spacing-2);
  }

  &__muted {
    color: var(--color-text-tertiary);
  }

  &__empty {
    color: var(--color-text-tertiary);
  }

  :deep(.el-table__row) {
    cursor: pointer;
  }
}
</style>
