<script setup lang="ts">
import { getCourseStatusMeta } from '@/constants'
import type { Course } from '@/types/course'

withDefaults(
  defineProps<{
    rows: Course[]
    loading?: boolean
    showEdit?: boolean
  }>(),
  {
    loading: false,
    showEdit: false
  }
)

const emit = defineEmits<{
  (e: 'view', course: Course): void
  (e: 'edit', course: Course): void
}>()

function handleRowClick(row: Course): void {
  emit('view', row)
}

function asCourse(row: unknown): Course {
  return row as Course
}
</script>

<template>
  <el-skeleton v-if="loading" :rows="8" animated class="course-table__skeleton" />
  <el-table
    v-else
    class="course-table"
    :data="rows"
    row-key="id"
    stripe
    @row-click="handleRowClick"
  >
    <el-table-column prop="code" label="课程编码" width="120" />
    <el-table-column prop="name" label="课程名称" min-width="180">
      <template #default="{ row }">
        <span class="course-table__name">{{ row.name }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="teacherName" label="授课教师" width="110" />
    <el-table-column prop="department" label="教研室" min-width="180" />
    <el-table-column prop="semester" label="学期" width="120" />
    <el-table-column prop="classCount" label="班级数" width="90" align="center">
      <template #default="{ row }">{{ row.classCount }}</template>
    </el-table-column>
    <el-table-column prop="studentCount" label="学生人次" width="100" align="right">
      <template #default="{ row }">
        <span class="tabular-nums">{{ row.studentCount }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="resourceCount" label="资源数" width="90" align="right">
      <template #default="{ row }">
        <span class="tabular-nums">{{ row.resourceCount }}</span>
      </template>
    </el-table-column>
    <el-table-column label="状态" width="100" align="center">
      <template #default="{ row }">
        <el-tag :type="getCourseStatusMeta(row.status).tag" effect="light" round>
          {{ getCourseStatusMeta(row.status).label }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column label="操作" width="130" fixed="right">
      <template #default="{ row }">
        <el-button link type="primary" @click.stop="emit('view', asCourse(row))">查看</el-button>
        <el-button v-if="showEdit" link type="primary" @click.stop="emit('edit', asCourse(row))">
          编辑
        </el-button>
      </template>
    </el-table-column>
    <template #empty>
      <span class="course-table__empty">暂无课程数据</span>
    </template>
  </el-table>
</template>

<style scoped lang="scss">
.course-table {
  width: 100%;

  &__skeleton {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border-radius: var(--radius-lg);
  }

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
