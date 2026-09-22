<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import FilterBar from '@/components/common/FilterBar.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import TeacherScoreTable from '@/components/teacher/TeacherScoreTable.vue'
import { fetchTeacherScoresApi } from '@/api/teacher'
import { useAuthStore } from '@/stores/auth'
import { useDictStore } from '@/stores/dict'
import { PAGE_SIZES } from '@/constants'
import type { TeacherScoreItem } from '@/types/teacher'

const router = useRouter()
const auth = useAuthStore()
const dict = useDictStore()

const rows = ref<TeacherScoreItem[]>([])
const loading = ref(true)
const error = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ semester: string; departmentId: number | '' }>({
  semester: '',
  departmentId: ''
})

/** 教研室筛选仅对督导开放：主任数据由后端裁剪为本室（§7.8）。 */
const canFilterDepartment = computed(() => auth.role === 'supervisor')

async function load(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    const result = await fetchTeacherScoresApi({
      semester: query.semester || undefined,
      departmentId: canFilterDepartment.value ? query.departmentId || undefined : undefined,
      page: page.value,
      pageSize: pageSize.value
    })
    rows.value = result.list
    total.value = result.total
  } catch {
    error.value = true
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function search(): void {
  page.value = 1
  void load()
}

function reset(): void {
  query.semester = ''
  query.departmentId = ''
  page.value = 1
  void load()
}

function handlePageChange(value: number): void {
  page.value = value
  void load()
}

function handlePageSizeChange(value: number): void {
  pageSize.value = value
  page.value = 1
  void load()
}

function handleView(item: TeacherScoreItem): void {
  void router.push({ name: 'teacher-detail', params: { id: item.teacherId } })
}

onMounted(() => {
  void dict.load()
  void load()
})
</script>

<template>
  <div class="teacher-list">
    <PageHeader title="教师管理" subtitle="查看本室教师的综合评分、各维度分与评价次数，以便统筹教学支持" />

    <el-alert
      class="teacher-list__ethics"
      type="info"
      :closable="false"
      show-icon
      title="仅用于教学支持，不作为考核依据"
      description="评分用于帮助教师改进教学，不用于绩效与人事评价。请勿外传或作他用。"
    />

    <section class="teacher-list__panel">
      <FilterBar @search="search" @reset="reset">
        <el-select v-model="query.semester" placeholder="学期" clearable class="teacher-list__field">
          <el-option
            v-for="semester in dict.semesters"
            :key="semester"
            :label="semester"
            :value="semester"
          />
        </el-select>
        <el-select
          v-if="canFilterDepartment"
          v-model="query.departmentId"
          placeholder="教研室"
          clearable
          class="teacher-list__field teacher-list__field--wide"
        >
          <el-option
            v-for="department in dict.departments"
            :key="department.id"
            :label="department.name"
            :value="department.id"
          />
        </el-select>
      </FilterBar>

      <div v-if="error" class="teacher-list__state">
        <EmptyState description="教师评分数据加载失败">
          <template #action>
            <el-button type="primary" @click="load">重新加载</el-button>
          </template>
        </EmptyState>
      </div>
      <template v-else>
        <TeacherScoreTable :rows="rows" :loading="loading" @view="handleView" />
        <EmptyState
          v-if="!loading && rows.length === 0"
          class="teacher-list__state"
          description="暂无教师评价数据"
        />
        <div class="teacher-list__pagination">
          <el-pagination
            :current-page="page"
            :page-size="pageSize"
            :page-sizes="PAGE_SIZES"
            :total="total"
            layout="total, sizes, prev, pager, next"
            background
            @current-change="handlePageChange"
            @size-change="handlePageSizeChange"
          />
        </div>
      </template>
    </section>
  </div>
</template>

<style scoped lang="scss">
.teacher-list {
  &__ethics {
    margin-bottom: var(--spacing-4);
  }

  &__panel {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__field {
    width: 150px;

    &--wide {
      width: 200px;
    }
  }

  &__state {
    padding: var(--spacing-6);
  }

  &__pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--spacing-4);
  }
}
</style>
