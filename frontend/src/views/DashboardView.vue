<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import StatCard from '@/components/common/StatCard.vue'
import CourseCard from '@/components/course/CourseCard.vue'
import CoverageCard from '@/components/supervision/CoverageCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import QualityCompass from '@/components/evaluation/QualityCompass.vue'
import ScoreHeatmap from '@/components/evaluation/ScoreHeatmap.vue'
import ScoreRadar, { type RadarItem } from '@/components/evaluation/ScoreRadar.vue'
import EvaluationQueue from '@/components/supervision/EvaluationQueue.vue'
import { fetchDashboardApi } from '@/api/dashboard'
import { fetchTeacherScoresApi, fetchTeacherSummaryApi, fetchTeacherEvaluationsApi } from '@/api/teacher'
import { useAuthStore } from '@/stores/auth'
import { useSemester } from '@/composables/useSemester'
import { EVALUATION_DIMENSIONS, getRoleMeta } from '@/constants'
import { formatDate, formatDateTime } from '@/utils/format'
import type { Course } from '@/types/course'
import type { DashboardData, SupervisionPlan } from '@/types/supervision'
import type { TeacherScoreItem, TeacherSummary, TeacherTimelineItem } from '@/types/teacher'

const router = useRouter()
const auth = useAuthStore()
const { semester } = useSemester()

const loading = ref(true)
const error = ref(false)
const data = ref<DashboardData | null>(null)

/* 主任：本室教师评分（质量热力数据源） */
const teacherScores = ref<TeacherScoreItem[]>([])
const teacherScoresLoading = ref(false)

/* 教师：本人质量档案摘要（罗盘 + 雷达）与最近评价流 */
const mySummary = ref<TeacherSummary | null>(null)
const mySummaryLoading = ref(false)
const recentEvaluations = ref<TeacherTimelineItem[]>([])

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

async function loadDirectorScores(): Promise<void> {
  teacherScoresLoading.value = true
  try {
    const page = await fetchTeacherScoresApi({ semester: semester.value, page: 1, pageSize: 100 })
    teacherScores.value = page.list ?? []
  } catch {
    teacherScores.value = []
  } finally {
    teacherScoresLoading.value = false
  }
}

async function loadTeacherQuality(): Promise<void> {
  const id = auth.user?.id
  if (!id) return
  mySummaryLoading.value = true
  try {
    mySummary.value = await fetchTeacherSummaryApi(id, { semester: semester.value })
  } catch {
    mySummary.value = null
  } finally {
    mySummaryLoading.value = false
  }
  try {
    const page = await fetchTeacherEvaluationsApi(id, { semester: semester.value, page: 1, pageSize: 3 })
    recentEvaluations.value = page.list ?? []
  } catch {
    recentEvaluations.value = []
  }
}

/* 主任侧「重点关注」：全部由 teacher-scores 单次响应派生，零额外请求 */
const weakestDimension = computed(() => {
  const scored = teacherScores.value.filter((teacher) => teacher.compositeScore !== null)
  if (!scored.length) return null
  const acc = EVALUATION_DIMENSIONS.filter((dim) => !dim.isObservation).map((dim) => {
    const values = scored
      .map((teacher) => teacher.dimensions.find((item) => item.key === dim.key)?.score)
      .filter((value): value is number => value !== null && value !== undefined)
    const avg = values.length ? values.reduce((sum, value) => sum + value, 0) / values.length : null
    return { ...dim, avg }
  })
  const withScore = acc.filter((item) => item.avg !== null)
  if (!withScore.length) return null
  return withScore.reduce((min, item) => ((item.avg ?? 0) < (min.avg ?? 0) ? item : min))
})

const insufficientTeachers = computed(() =>
  teacherScores.value.filter((teacher) => teacher.compositeScore === null || !teacher.sample.sampleSufficient)
)

const radarItems = computed<RadarItem[]>(() =>
  (mySummary.value?.dimensions ?? []).map((dim) => ({
    key: dim.key,
    name: EVALUATION_DIMENSIONS.find((item) => item.key === dim.key)?.shortName ?? dim.name,
    score: dim.score,
    isObservation: dim.isObservation
  }))
)

function firstComment(item: TeacherTimelineItem): string {
  const supervisor = item.supervisorEvaluations?.[0]
  return (supervisor && supervisor.comment) || item.agentEvaluation?.comment || ''
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

function goTeacherQuality(teacherId: number): void {
  void router.push({ name: 'teacher-detail', params: { id: teacherId } })
}

function handleQueueCreate(plan: SupervisionPlan): void {
  void router.push({
    name: 'supervision',
    query: { plan: String(plan.id), course: String(plan.courseId) }
  })
}

function handleQueueViewCourse(plan: SupervisionPlan): void {
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

function reloadQuality(): void {
  void loadData()
  if (auth.hasRole('director')) void loadDirectorScores()
  if (auth.hasRole('teacher')) void loadTeacherQuality()
}

watch(semester, reloadQuality)

onMounted(reloadQuality)
</script>

<template>
  <div class="dashboard">
    <div class="dashboard__greeting">
      <div>
        <h1 class="dashboard__hello">{{ greeting }}，{{ auth.user?.name }}</h1>
        <p class="dashboard__role">{{ roleLabel }} · 欢迎回到质量驾驶舱（{{ semester }}）</p>
      </div>
      <p class="dashboard__ethics">评分仅用于教学支持与改进，不作为考核依据</p>
    </div>

    <div v-if="error" class="dashboard__error">
      <EmptyState description="驾驶舱数据加载失败">
        <template #action>
          <el-button type="primary" @click="reloadQuality">重新加载</el-button>
        </template>
      </EmptyState>
    </div>

    <template v-else>
      <div class="dashboard__stats">
        <template v-if="loading">
          <el-skeleton v-for="n in 4" :key="n" animated class="dashboard__stat-skeleton">
            <template #template>
              <el-skeleton-item variant="rect" style="height: 76px; border-radius: 12px" />
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

      <!-- 主任视角：本室质量热力 + 重点关注 -->
      <div v-if="auth.role === 'director'" class="dashboard__grid">
        <section class="dashboard__panel">
          <div class="dashboard__panel-head">
            <h2 class="dashboard__panel-title">本室质量热力</h2>
            <span class="dashboard__panel-sub">教师 × 五维 · 点行下钻教师画像</span>
          </div>
          <ScoreHeatmap
            :teachers="teacherScores"
            :loading="teacherScoresLoading"
            @select="goTeacherQuality"
          />
        </section>

        <aside class="dashboard__side">
          <section class="dashboard__panel">
            <h2 class="dashboard__panel-title">重点关注</h2>
            <template v-if="!teacherScoresLoading && teacherScores.length">
              <p v-if="weakestDimension" class="dashboard__focus-item">
                <span class="dashboard__focus-label">维度均分最低</span>
                <span class="dashboard__focus-value">
                  {{ weakestDimension.shortName }}
                  <QualityBadge :score="weakestDimension.avg ?? null" />
                </span>
              </p>
              <p class="dashboard__focus-item">
                <span class="dashboard__focus-label">需关注样本</span>
                <span class="dashboard__focus-value">
                  {{ insufficientTeachers.length }} 位教师暂无评价或样本不足
                </span>
              </p>
              <el-button
                v-if="insufficientTeachers.length"
                text
                type="primary"
                class="dashboard__focus-link"
                @click="goTeacherList"
              >
                前往教师画像逐一查看 →
              </el-button>
            </template>
            <EmptyState v-else description="暂无教师评分数据" />
          </section>

          <section class="dashboard__panel">
            <h2 class="dashboard__panel-title">快捷操作</h2>
            <el-button type="primary" class="dashboard__quick" @click="goNewCourse">
              新增课程
            </el-button>
            <el-button class="dashboard__quick" @click="goCourseList">进入课程库</el-button>
            <el-button class="dashboard__quick" @click="goTeacherList">进入教师画像</el-button>
          </section>
        </aside>
      </div>

      <!-- 教师视角：我的质量总览 + 最近评价流 + 我的课程 -->
      <div v-else-if="auth.role === 'teacher'" class="dashboard__grid">
        <section class="dashboard__panel">
          <div class="dashboard__panel-head">
            <h2 class="dashboard__panel-title">我的教学质量</h2>
            <router-link class="dashboard__panel-sub-link" :to="{ name: 'course-list' }">
              课程视角明细见「我的课程」
            </router-link>
          </div>
          <div class="dashboard__quality">
            <QualityCompass
              class="dashboard__compass"
              :summary="mySummary"
              :loading="mySummaryLoading"
              size="lg"
              title="本学期综合分"
            />
            <div class="dashboard__radar">
              <ScoreRadar :items="radarItems" :max="100" :loading="mySummaryLoading" />
            </div>
          </div>
        </section>

        <aside class="dashboard__side">
          <section class="dashboard__panel">
            <div class="dashboard__panel-head">
              <h2 class="dashboard__panel-title">最近评价</h2>
              <router-link class="dashboard__panel-sub-link" :to="{ name: 'profile-quality' }">
                进入我的质量档案 →
              </router-link>
            </div>
            <ul v-if="recentEvaluations.length" class="dashboard__feed">
              <li
                v-for="item in recentEvaluations"
                :key="`${item.sessionId}-${item.courseId}`"
                class="dashboard__feed-item"
              >
                <div class="dashboard__feed-head">
                  <span class="dashboard__feed-course">{{ item.courseName }}</span>
                  <QualityBadge :score="item.compositeScore" />
                </div>
                <p class="dashboard__feed-topic">
                  {{ formatDate(item.sessionDate) }} · {{ item.topic || '—' }}
                </p>
                <p v-if="firstComment(item)" class="dashboard__feed-comment">
                  {{ firstComment(item) }}
                </p>
              </li>
            </ul>
            <EmptyState v-else description="本学期暂无督导评价，请留意课堂安排" />
          </section>

          <section class="dashboard__panel">
            <h2 class="dashboard__panel-title">资源快捷入口</h2>
            <el-button type="primary" class="dashboard__quick" @click="goUploadResource">
              上传课程资源
            </el-button>
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
          </section>
        </aside>
      </div>

      <!-- 督导视角：待评课队列第一行 -->
      <div v-else-if="auth.role === 'supervisor'" class="dashboard__grid dashboard__grid--queue">
        <section class="dashboard__panel">
          <div class="dashboard__panel-head">
            <h2 class="dashboard__panel-title">待评课队列</h2>
            <el-button link type="primary" @click="goSupervision">课堂评估页</el-button>
          </div>
          <EvaluationQueue
            :plans="data?.plans ?? []"
            :loading="loading"
            @create="handleQueueCreate"
            @view-course="handleQueueViewCourse"
          />
        </section>

        <aside class="dashboard__side">
          <CoverageCard :overall="data?.coverage ?? null" :loading="loading" />
        </aside>
      </div>

      <!-- 教师：我的课程卡带 -->
      <div v-if="auth.role === 'teacher'" class="dashboard__panel dashboard__panel--courses">
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
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.dashboard {
  &__greeting {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: var(--spacing-4);
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

  &__ethics {
    flex-shrink: 0;
    padding: 2px 10px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    background-color: var(--color-bg-page);
    border-radius: 999px;
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

    &--queue {
      grid-template-columns: 1.6fr 1fr;
    }

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

    &--courses {
      margin-top: var(--spacing-6);
    }
  }

  &__panel-head {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__panel-title {
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__panel-sub {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__panel-sub-link {
    font-size: var(--font-size-xs);
    color: var(--color-primary);
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

  &__focus {
    &-item {
      display: flex;
      flex-direction: column;
      gap: var(--spacing-1);
      padding: var(--spacing-3) 0;

      & + & {
        border-top: 1px dashed var(--color-divider);
      }
    }

    &-label {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    &-value {
      display: flex;
      gap: var(--spacing-2);
      align-items: center;
      font-size: var(--font-size-base);
      font-weight: 600;
      color: var(--color-text-primary);
    }

    &-link {
      margin-top: var(--spacing-2);
    }
  }

  &__quality {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: var(--spacing-8);
    align-items: center;

    @media (max-width: 1024px) {
      grid-template-columns: 1fr;
      justify-items: center;
    }
  }

  &__radar {
    min-width: 0;
  }

  &__feed {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);

    &-item {
      padding: var(--spacing-2) var(--spacing-3);
      background-color: var(--color-bg-page);
      border-radius: var(--radius-md);
    }

    &-head {
      display: flex;
      gap: var(--spacing-2);
      align-items: center;
      justify-content: space-between;
    }

    &-course {
      overflow: hidden;
      font-size: var(--font-size-sm);
      font-weight: 600;
      color: var(--color-text-primary);
      white-space: nowrap;
      text-overflow: ellipsis;
    }

    &-topic {
      margin-top: 2px;
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    &-comment {
      margin-top: var(--spacing-1);
      display: -webkit-box;
      overflow: hidden;
      font-size: var(--font-size-sm);
      line-height: 1.6;
      color: var(--color-text-secondary);
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 2;
    }
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
    margin-top: var(--spacing-3);
  }

  &__recent-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-3);
    padding: var(--spacing-2) var(--spacing-3);
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
