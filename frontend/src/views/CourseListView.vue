<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import FilterBar from '@/components/common/FilterBar.vue'
import CourseTable from '@/components/course/CourseTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { useCourseList } from '@/composables/useCourseList'
import { useAuthStore } from '@/stores/auth'
import { useDictStore } from '@/stores/dict'
import { COURSE_STATUS, PAGE_SIZES } from '@/constants'
import type { Course } from '@/types/course'

const router = useRouter()
const auth = useAuthStore()
const dict = useDictStore()

const {
  query,
  rows,
  loading,
  error,
  page,
  pageSize,
  total,
  fetchList,
  search,
  reset,
  handlePageChange,
  handlePageSizeChange
} = useCourseList()

const pageTitle = computed(() => {
  if (auth.role === 'director') return '课程管理'
  if (auth.role === 'supervisor') return '全校课程'
  return '我的课程'
})

const canFilterTeacher = computed(() => auth.role === 'director' || auth.role === 'supervisor')

function goDetail(course: Course): void {
  if (auth.role === 'teacher') {
    void router.push({ name: 'course-improve', params: { id: course.id } })
    return
  }
  void router.push({ name: 'course-detail', params: { id: course.id } })
}

function goEdit(course: Course): void {
  void router.push({ name: 'course-edit', params: { id: course.id } })
}

function goNew(): void {
  void router.push({ name: 'course-new' })
}

onMounted(() => {
  void dict.load()
})
</script>

<template>
  <div class="course-list">
    <PageHeader :title="pageTitle" subtitle="查看课程简介与开课情况">
      <template #actions>
        <el-button v-if="auth.role === 'director'" type="primary" @click="goNew">
          新增课程
        </el-button>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <el-select v-model="query.semester" placeholder="学期" clearable class="course-list__field">
        <el-option
          v-for="semester in dict.semesters"
          :key="semester"
          :label="semester"
          :value="semester"
        />
      </el-select>

      <el-select
        v-if="canFilterTeacher"
        v-model="query.departmentId"
        placeholder="教研室"
        clearable
        class="course-list__field course-list__field--wide"
      >
        <el-option
          v-for="department in dict.departments"
          :key="department.id"
          :label="department.name"
          :value="department.id"
        />
      </el-select>

      <el-select
        v-if="canFilterTeacher"
        v-model="query.teacherId"
        placeholder="授课教师"
        clearable
        class="course-list__field"
      >
        <el-option
          v-for="teacher in dict.teachers"
          :key="teacher.id"
          :label="teacher.name"
          :value="teacher.id"
        />
      </el-select>

      <el-select v-model="query.status" placeholder="状态" clearable class="course-list__field">
        <el-option
          v-for="item in COURSE_STATUS"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </el-select>

      <el-input
        v-model="query.keyword"
        placeholder="搜索课程名称 / 编码"
        clearable
        class="course-list__search"
        @keyup.enter="search"
      />
    </FilterBar>

    <div class="course-list__body">
      <EmptyState v-if="error" description="课程数据加载失败，请重试">
        <template #action>
          <el-button type="primary" @click="fetchList">重新加载</el-button>
        </template>
      </EmptyState>

      <template v-else>
        <CourseTable
          :rows="rows"
          :loading="loading"
          :show-edit="auth.role === 'director'"
          @view="goDetail"
          @edit="goEdit"
        />
        <EmptyState
          v-if="!loading && rows.length === 0"
          class="course-list__empty"
          description="没有符合条件的课程"
        >
          <template #action>
            <el-button @click="reset">重置筛选条件</el-button>
          </template>
        </EmptyState>
      </template>
    </div>

    <div v-if="!error" class="course-list__pagination">
      <el-pagination
        :current-page="page"
        :page-size="pageSize"
        :page-sizes="PAGE_SIZES"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        background
        @current-change="handlePageChange"
        @size-change="handlePageSizeChange"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.course-list {
  &__field {
    width: 150px;

    &--wide {
      width: 200px;
    }
  }

  &__search {
    width: 240px;
  }

  &__body {
    overflow: hidden;
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__empty {
    padding: var(--spacing-8);
  }

  &__pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--spacing-4);
  }
}
</style>
