<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import FilterBar from '@/components/common/FilterBar.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ScheduleTable from '@/components/supervision/ScheduleTable.vue'
import { fetchPlansApi } from '@/api/supervision'
import { PAGE_SIZES, SUPERVISION_STATUS } from '@/constants'
import type { SupervisionPlan, SupervisionStatus } from '@/types/supervision'

const router = useRouter()

const plans = ref<SupervisionPlan[]>([])
const plansLoading = ref(true)
const plansError = ref(false)
const planTotal = ref(0)

const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ status: SupervisionStatus | ''; date: string }>({
  status: '',
  date: ''
})

async function loadPlans(): Promise<void> {
  plansLoading.value = true
  plansError.value = false
  try {
    const result = await fetchPlansApi({
      status: query.status || undefined,
      date: query.date || undefined,
      page: page.value,
      pageSize: pageSize.value
    })
    plans.value = result.list
    planTotal.value = result.total
  } catch {
    plansError.value = true
    plans.value = []
    planTotal.value = 0
  } finally {
    plansLoading.value = false
  }
}

function search(): void {
  page.value = 1
  void loadPlans()
}

function reset(): void {
  query.status = ''
  query.date = ''
  page.value = 1
  void loadPlans()
}

function handlePageChange(value: number): void {
  page.value = value
  void loadPlans()
}

function handlePageSizeChange(value: number): void {
  pageSize.value = value
  page.value = 1
  void loadPlans()
}

function handlePlanView(plan: SupervisionPlan): void {
  void router.push({ name: 'course-detail', params: { id: plan.courseId } })
}

onMounted(() => {
  void loadPlans()
})
</script>

<template>
  <div class="supervision">
    <PageHeader
      title="听评课管理"
      subtitle="完整听评课安排，可按状态与计划日期筛选；覆盖率与统计卡见工作台"
    />

    <section class="supervision__panel">
      <FilterBar @search="search" @reset="reset">
        <el-select v-model="query.status" placeholder="状态" clearable class="supervision__field">
          <el-option
            v-for="item in SUPERVISION_STATUS"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
        <el-date-picker
          v-model="query.date"
          type="date"
          placeholder="计划日期（起）"
          value-format="YYYY-MM-DD"
          class="supervision__field"
        />
      </FilterBar>

      <div v-if="plansError" class="supervision__state">
        <EmptyState description="听评课安排加载失败">
          <template #action>
            <el-button type="primary" @click="loadPlans">重新加载</el-button>
          </template>
        </EmptyState>
      </div>
      <template v-else>
        <ScheduleTable :plans="plans" :loading="plansLoading" @view="handlePlanView" />
        <EmptyState
          v-if="!plansLoading && plans.length === 0"
          class="supervision__state"
          description="暂无符合条件的听评课安排"
        />
        <div class="supervision__pagination">
          <el-pagination
            :current-page="page"
            :page-size="pageSize"
            :page-sizes="PAGE_SIZES"
            :total="planTotal"
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
.supervision {
  &__panel {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__field {
    width: 180px;
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
