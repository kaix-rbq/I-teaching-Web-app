<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import FilterBar from '@/components/common/FilterBar.vue'
import StatCard from '@/components/common/StatCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import CoverageCard from '@/components/supervision/CoverageCard.vue'
import ScheduleTable from '@/components/supervision/ScheduleTable.vue'
import { fetchCoverageApi, fetchPlansApi } from '@/api/supervision'
import { PAGE_SIZES, SUPERVISION_STATUS } from '@/constants'
import type {
  CoverageStat,
  SupervisionPlan,
  SupervisionStatus
} from '@/types/supervision'

const router = useRouter()

const coverage = ref<CoverageStat | null>(null)
const coverageLoading = ref(true)

const plans = ref<SupervisionPlan[]>([])
const plansLoading = ref(true)
const plansError = ref(false)
const planTotal = ref(0)
const scheduledTotal = ref(0)

const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ status: SupervisionStatus | ''; date: string }>({
  status: '',
  date: ''
})

async function loadCoverage(): Promise<void> {
  coverageLoading.value = true
  try {
    coverage.value = await fetchCoverageApi()
  } catch {
    coverage.value = null
  } finally {
    coverageLoading.value = false
  }
}

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

async function loadScheduledTotal(): Promise<void> {
  try {
    const result = await fetchPlansApi({ page: 1, pageSize: 1 })
    scheduledTotal.value = result.total
  } catch {
    scheduledTotal.value = 0
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
  void loadCoverage()
  void loadScheduledTotal()
  void loadPlans()
})
</script>

<template>
  <div class="supervision">
    <PageHeader title="督导总览" subtitle="查看督导覆盖率与听评课安排" />

    <div class="supervision__stats">
      <StatCard
        label="全校课程数"
        :value="coverage?.totalCourses ?? 0"
        unit="门"
        icon="course"
        tone="primary"
      />
      <StatCard
        label="本学期听评课计划"
        :value="scheduledTotal"
        unit="项"
        icon="plan"
        tone="info"
      />
      <StatCard
        label="已完成听评课"
        :value="coverage?.supervisedCourses ?? 0"
        unit="项"
        icon="done"
        tone="success"
      />
      <StatCard
        label="督导覆盖率"
        :value="Math.round((coverage?.rate ?? 0) * 100)"
        unit="%"
        icon="coverage"
        tone="supervisor"
      />
    </div>

    <div class="supervision__grid">
      <section class="supervision__panel">
        <h2 class="supervision__panel-title">听评课安排</h2>

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

      <aside class="supervision__side">
        <CoverageCard :overall="coverage" :loading="coverageLoading" />
      </aside>
    </div>
  </div>
</template>

<style scoped lang="scss">
.supervision {
  &__stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--spacing-4);
    margin-bottom: var(--spacing-6);

    @media (max-width: 1280px) {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  &__grid {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: var(--spacing-6);
    align-items: start;

    @media (max-width: 1280px) {
      grid-template-columns: 1fr;
    }
  }

  &__panel {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__panel-title {
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
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
