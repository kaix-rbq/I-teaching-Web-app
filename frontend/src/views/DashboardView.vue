<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import StatCard from '@/components/common/StatCard.vue'
import CourseCard from '@/components/course/CourseCard.vue'
import ScheduleTable from '@/components/supervision/ScheduleTable.vue'
import CoverageCard from '@/components/supervision/CoverageCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { fetchDashboardApi } from '@/api/dashboard'
import { useAuthStore } from '@/stores/auth'
import { getRoleMeta } from '@/constants'
import { formatDateTime } from '@/utils/format'
import type { Course } from '@/types/course'
import type { DashboardData, SupervisionPlan } from '@/types/supervision'

const router = useRouter()
const auth = useAuthStore()

const loading = ref(true)
const error = ref(false)
const data = ref<DashboardData | null>(null)

const roleLabel = computed(() => getRoleMeta(auth.user?.role)?.label ?? '')

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '凌晨好'
  if (hour < 12) return '早上好'
  if (hour < 14) return '中午好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

async function loadData(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    data.value = await fetchDashboardApi()
  } catch {
    error.value = true
    data.value = null
  } finally {
    loading.value = false
  }
}

function goCourseImprove(course: Course): void {
  void router.push({ name: 'course-improve', params: { id: course.id } })
}

function goCourseList(): void {
  void router.push({ name: 'course-list' })
}

function goNewCourse(): void {
  void router.push({ name: 'course-new' })
}

function goTeacherList(): void {
  void router.push({ name: 'teacher-list' })
}

function goSupervision(): void {
  void router.push({ name: 'supervision' })
}

function handlePlanView(plan: SupervisionPlan): void {
  void router.push({ name: 'course-detail', params: { id: plan.courseId } })
}

function goUploadResource(): void {
  const first = data.value?.courses?.[0]
  if (first) {
    void router.push({ name: 'course-detail', params: { id: first.id }, query: { tab: 'resource' } })
  } else {
    goCourseList()
  }
}

onMounted(() => {
  void loadData()
})
</script>

<template>
  <div class="dashboard">
    <div class="dashboard__greeting">
      <h1 class="dashboard__hello">{{ greeting }}，{{ auth.user?.name }}</h1>
      <p class="dashboard__role">{{ roleLabel }} · 欢迎回到「爱教学」工作台</p>
    </div>

    <div v-if="error" class="dashboard__error">
      <EmptyState description="工作台数据加载失败">
        <template #action>
          <el-button type="primary" @click="loadData">重新加载</el-button>
        </template>
      </EmptyState>
    </div>

    <template v-else>
      <div class="dashboard__stats">
        <template v-if="loading">
          <el-skeleton v-for="n in 4" :key="n" animated class="dashboard__stat-skeleton">
            <template #template>
              <el-skeleton-item variant="rect" style="height: 96px; border-radius: 12px" />
            </template>
          </el-skeleton>
        </template>
        <template v-else>
          <StatCard
            v-for="stat in data?.stats ?? []"
            :key="stat.label"
            :label="stat.label"
            :value="stat.value"
            :unit="stat.unit"
            :icon="stat.icon"
            :tone="stat.tone"
          />
        </template>
      </div>

      <!-- 主任视图 -->
      <div v-if="auth.role === 'director'" class="dashboard__grid">
        <section class="dashboard__panel">
          <div class="dashboard__panel-head">
            <h2 class="dashboard__panel-title">管理入口</h2>
          </div>
          <div class="dashboard__entries">
            <button class="dashboard__entry" type="button" @click="goCourseList">
              <span class="dashboard__entry-title">课程管理</span>
              <span class="dashboard__entry-desc">
                查看本室课程简介与开课情况，新增或维护课程信息
              </span>
            </button>
            <button class="dashboard__entry" type="button" @click="goTeacherList">
              <span class="dashboard__entry-title">教师管理</span>
              <span class="dashboard__entry-desc">
                查看本室教师综合评分与各维度分，统筹教学支持
              </span>
            </button>
          </div>
        </section>

        <aside class="dashboard__side">
          <section class="dashboard__panel">
            <h2 class="dashboard__panel-title">快捷操作</h2>
            <el-button type="primary" class="dashboard__quick" @click="goNewCourse">
              新增课程
            </el-button>
            <el-button class="dashboard__quick" @click="goCourseList">进入课程管理</el-button>
            <el-button class="dashboard__quick" @click="goTeacherList">进入教师管理</el-button>
          </section>
        </aside>
      </div>

      <!-- 教师视图 -->
      <div v-else-if="auth.role === 'teacher'" class="dashboard__grid">
        <section class="dashboard__panel">
          <div class="dashboard__panel-head">
            <h2 class="dashboard__panel-title">我的课程</h2>
            <el-button link type="primary" @click="goCourseList">查看全部</el-button>
          </div>
          <div v-if="loading" class="dashboard__cards">
            <el-skeleton v-for="n in 4" :key="n" animated :rows="4" />
          </div>
          <div v-else-if="(data?.courses ?? []).length" class="dashboard__cards">
            <CourseCard
              v-for="course in data?.courses ?? []"
              :key="course.id"
              :course="course"
              @click="goCourseImprove"
            />
          </div>
          <EmptyState v-else description="本学期暂无授课课程" />
        </section>

        <aside class="dashboard__side">
          <section class="dashboard__panel">
            <h2 class="dashboard__panel-title">资源快捷入口</h2>
            <el-button type="primary" class="dashboard__quick" @click="goUploadResource">
              上传课程资源
            </el-button>
          </section>

          <section class="dashboard__panel">
            <h2 class="dashboard__panel-title">近期上传资源</h2>
            <ul v-if="(data?.recentResources ?? []).length" class="dashboard__recent">
              <li
                v-for="resource in data?.recentResources ?? []"
                :key="resource.id"
                class="dashboard__recent-item"
              >
                <span class="dashboard__recent-name">{{ resource.name }}</span>
                <span class="dashboard__recent-meta">
                  {{ formatDateTime(resource.uploadedAt) }}
                </span>
              </li>
            </ul>
            <EmptyState v-else description="暂无上传记录" />
          </section>
        </aside>
      </div>

      <!-- 督导视图 -->
      <div v-else-if="auth.role === 'supervisor'" class="dashboard__grid">
        <section class="dashboard__panel">
          <div class="dashboard__panel-head">
            <h2 class="dashboard__panel-title">听评课安排</h2>
            <el-button link type="primary" @click="goSupervision">查看全部</el-button>
          </div>
          <ScheduleTable :plans="data?.plans ?? []" :loading="loading" @view="handlePlanView" />
        </section>

        <aside class="dashboard__side">
          <CoverageCard :overall="data?.coverage ?? null" :loading="loading" />
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.dashboard {
  &__greeting {
    margin-bottom: var(--spacing-6);
  }

  &__hello {
    font-size: var(--font-size-3xl);
    color: var(--color-text-primary);
  }

  &__role {
    margin-top: var(--spacing-1);
    color: var(--color-text-tertiary);
  }

  &__error {
    padding: var(--spacing-8);
    background-color: var(--color-bg-card);
    border-radius: var(--radius-lg);
  }

  &__stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--spacing-4);
    margin-bottom: var(--spacing-6);

    @media (max-width: 1280px) {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  &__stat-skeleton {
    width: 100%;
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

    & + & {
      margin-top: var(--spacing-4);
    }
  }

  &__panel-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__panel-title {
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__side {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
  }

  &__quick {
    width: 100%;
    margin-left: 0;

    & + & {
      margin-top: var(--spacing-2);
    }
  }

  &__entries {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--spacing-4);

    @media (max-width: 1024px) {
      grid-template-columns: 1fr;
    }
  }

  &__entry {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    padding: var(--spacing-6);
    text-align: left;
    cursor: pointer;
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    transition: border-color 0.2s ease, box-shadow 0.2s ease;

    &:hover {
      border-color: var(--color-primary);
      box-shadow: var(--shadow-card-hover);
    }
  }

  &__entry-title {
    font-size: var(--font-size-xl);
    font-weight: 600;
    color: var(--color-primary);
  }

  &__entry-desc {
    font-size: var(--font-size-sm);
    line-height: 1.6;
    color: var(--color-text-tertiary);
  }

  &__cards {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--spacing-4);

    @media (max-width: 1024px) {
      grid-template-columns: 1fr;
    }
  }

  &__recent {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
  }

  &__recent-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-3);
    padding: var(--spacing-2) var(--spacing-3);
    cursor: pointer;
    border-radius: var(--radius-md);
    transition: background-color 0.2s ease;

    &:hover {
      background-color: var(--color-primary-bg);
    }
  }

  &__recent-name {
    overflow: hidden;
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__recent-meta {
    flex-shrink: 0;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }
}
</style>
