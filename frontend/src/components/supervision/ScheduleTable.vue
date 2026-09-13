<script setup lang="ts">
import { getSupervisionStatusMeta } from '@/constants'
import { formatDate } from '@/utils/format'
import type { SupervisionPlan } from '@/types/supervision'

withDefaults(
  defineProps<{
    plans: SupervisionPlan[]
    loading?: boolean
  }>(),
  {
    loading: false
  }
)

const emit = defineEmits<{ (e: 'view', plan: SupervisionPlan): void }>()

function handleRowClick(row: SupervisionPlan): void {
  emit('view', row)
}
</script>

<template>
  <el-skeleton v-if="loading" :rows="6" animated />
  <el-table
    v-else
    class="schedule-table"
    :data="plans"
    row-key="id"
    stripe
    @row-click="handleRowClick"
  >
    <el-table-column prop="courseName" label="课程名称" min-width="180">
      <template #default="{ row }">
        <span class="schedule-table__name">{{ row.courseName }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="teacherName" label="授课教师" width="110" />
    <el-table-column prop="supervisorName" label="督导人" width="110" />
    <el-table-column label="计划日期" width="140">
      <template #default="{ row }">{{ formatDate(row.plannedDate) }}</template>
    </el-table-column>
    <el-table-column label="状态" width="110" align="center">
      <template #default="{ row }">
        <el-tag :type="getSupervisionStatusMeta(row.status).tag" effect="light" round>
          {{ getSupervisionStatusMeta(row.status).label }}
        </el-tag>
      </template>
    </el-table-column>
    <template #empty>
      <span class="schedule-table__empty">暂无听评课安排</span>
    </template>
  </el-table>
</template>

<style scoped lang="scss">
.schedule-table {
  width: 100%;

  &__name {
    font-weight: 500;
    color: var(--color-primary);
  }

  &__empty {
    color: var(--color-text-tertiary);
  }

  :deep(.el-table__row) {
    cursor: pointer;
  }
}
</style>
