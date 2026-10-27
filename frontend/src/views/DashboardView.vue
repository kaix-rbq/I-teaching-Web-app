<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { Collection, Plus, UserFilled } from '@element-plus/icons-vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ScoreHeatmap from '@/components/evaluation/ScoreHeatmap.vue'
import EvaluationQueue from '@/components/supervision/EvaluationQueue.vue'
import ScheduleBoard from '@/components/schedule/ScheduleBoard.vue'
import ClassroomIssueBoard from '@/components/classroom/ClassroomIssueBoard.vue'
import { fetchDashboardApi } from '@/api/dashboard'
import { fetchTeacherScoresApi } from '@/api/teacher'
import { fetchSamplingTasksApi } from '@/api/supervision'
import { fetchSessionDetailApi } from '@/api/session'
import { fetchCourseDetailApi } from '@/api/course'
import { useAuthStore } from '@/stores/auth'
import { useSemester } from '@/composables/useSemester'
import { EVALUATION_DIMENSIONS } from '@/constants'
import type { DashboardData, SamplingTask, SupervisionPlan } from '@/types/supervision'
import type { Course } from '@/types/course'
import type { SessionDetail } from '@/types/evaluation'
import type { TeacherScoreItem } from '@/types/teacher'

/**
 * 角色工作台。
 *   主任：事务管理周视图 + 一屏决策带（先谈谁、薄弱在哪）+ 异常行热力；
 *   督导：今日待评横带 + 接下来 / 补录两组任务流 + 抽样复评侧栏；
 *   教师：不使用本页，登录后直达「教师工作台」。
 */
const router = useRouter()
const auth = useAuthStore()
const { semester } = useSemester()

const loading = ref(true)
const error = ref(false)
const data = ref<DashboardData | null>(null)

const WEEK_LABELS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

/* 主任：本室教师评分（质量热力数据源） */
const teacherScores = ref<TeacherScoreItem[]>([])
const teacherScoresLoading = ref(false)

const workbenchTitle = computed(() => (auth.role === 'supervisor' ? '督导工作台' : '主任驾驶舱'))
const todayLabel = computed(() => `${dayjs().format('M月D日')} ${WEEK_LABELS[dayjs().day()]}`)

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '凌晨好'
  if (hour < 12) return '早上好'
  if (hour < 14) return '中午好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

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

function teacherAiScore(teacher: TeacherScoreItem): number | null {
  return teacher.agentScore ?? teacher.compositeScore ?? null
}

const weakestDimension = computed(() => {
  const scored = teacherScores.value.filter((teacher) => teacherAiScore(teacher) !== null)
  if (!scored.length) return null
  const acc = EVALUATION_DIMENSIONS.filter((dim) => !dim.isObservation).map((dim) => {
    const values = scored
      .map((teacher) => teacher.dimensions.find((item) => item.key === dim.key))
      .map((row) => row?.agentScore ?? row?.score ?? null)
      .filter((value): value is number => value !== null && value !== undefined)
    const avg = values.length ? values.reduce((sum, value) => sum + value, 0) / values.length : null
    return { ...dim, avg }
  })
  const withScore = acc.filter((item) => item.avg !== null)
  if (!withScore.length) return null
  return withScore.reduce((min, item) => ((item.avg ?? 0) < (min.avg ?? 0) ? item : min))
})

/** 一句话行动建议：按最弱维度给出可执行动作，不空泛。 */
const WEAK_DIMENSION_ADVICE: Record<string, string> = {
  objective: '安排一次集体备课，先核对教学目标与内容准确性。',
  content: '围绕内容深度组织一次课程研讨，补充案例与前沿素材。',
  interaction: '下次听评课重点观察提问等待时间与学生参与度。',
  organization: '帮教师复盘课堂节奏，重排环节时间分配。',
  frontier: '鼓励在课程中引入学科前沿或交叉应用案例。'
}

const weakestAdvice = computed(
  () => WEAK_DIMENSION_ADVICE[weakestDimension.value?.key ?? ''] ?? '先积累几节课的 AI 评价样本，再决定跟进重点。'
)

function goCourseList(): void {
  void router.push({ name: 'course-list' })
}

function goNewCourse(): void {
  void router.push({ name: 'course-new' })
}

function goTeacherList(): void {
  void router.push({ name: 'teacher-list' })
}

function goTeacherQuality(teacherId: number): void {
  void router.push({ name: 'teacher-detail', params: { id: teacherId } })
}

/* ===== 督导任务流 ===== */
const todayPlans = computed(() =>
  (data.value?.plans ?? [])
    .filter((plan) => !plan.evaluated && dayjs(plan.plannedDate).isSame(dayjs(), 'day'))
    .sort((a, b) => dayjs(a.plannedDate).valueOf() - dayjs(b.plannedDate).valueOf())
)

const futurePlans = computed(() =>
  (data.value?.plans ?? [])
    .filter((plan) => !plan.evaluated && dayjs(plan.plannedDate).isAfter(dayjs(), 'day'))
    .sort((a, b) => dayjs(a.plannedDate).valueOf() - dayjs(b.plannedDate).valueOf())
)

const pastPlans = computed(() =>
  (data.value?.plans ?? [])
    .filter((plan) => !plan.evaluated && dayjs(plan.plannedDate).isBefore(dayjs(), 'day'))
    .sort((a, b) => dayjs(b.plannedDate).valueOf() - dayjs(a.plannedDate).valueOf())
)

const focusPlan = computed<SupervisionPlan | null>(() => todayPlans.value[0] ?? futurePlans.value[0] ?? null)
const focusIsToday = computed(() => Boolean(focusPlan.value && dayjs(focusPlan.value.plannedDate).isSame(dayjs(), 'day')))
const focusIsFuture = computed(() => Boolean(focusPlan.value && dayjs(focusPlan.value.plannedDate).isAfter(dayjs(), 'day')))

const focusDateLabel = computed(() => {
  if (!focusPlan.value) return todayLabel.value
  const date = dayjs(focusPlan.value.plannedDate)
  return `${date.format('M月D日')} ${WEEK_LABELS[date.day()]}`
})

const otherUpcoming = computed(() => {
  const focusId = focusPlan.value?.id
  return [...todayPlans.value, ...futurePlans.value].filter((plan) => plan.id !== focusId)
})

const focusCourse = ref<Course | null>(null)
const focusSession = ref<SessionDetail | null>(null)

async function loadFocusDetail(): Promise<void> {
  const plan = focusPlan.value
  focusCourse.value = null
  focusSession.value = null
  if (!plan) return
  const [courseResult, sessionResult] = await Promise.allSettled([
    fetchCourseDetailApi(plan.courseId),
    plan.sessionId === null ? Promise.resolve(null) : fetchSessionDetailApi(plan.sessionId)
  ])
  if (courseResult.status === 'fulfilled') focusCourse.value = courseResult.value
  if (sessionResult.status === 'fulfilled') focusSession.value = sessionResult.value
}

watch(() => focusPlan.value?.id, () => void loadFocusDetail())

const focusClass = computed(() => {
  if (focusSession.value?.className) return focusSession.value.className
  return focusCourse.value?.classes?.[0]?.className || '班级待补'
})

const focusLocation = computed(() => focusCourse.value?.classes?.[0]?.location || '地点待定')
const focusPeriod = computed(() => focusSession.value?.period || '节次待定')

const focusStatus = computed(() => {
  if (!focusPlan.value) return '今天没有待评安排。'
  if (focusIsFuture.value) return '授课日期未到，评估入口在当天开放。'
  if (focusPlan.value.sessionId === null) {
    return '教师尚未创建本场授课记录；教师下课后刷新，系统会自动关联并可直接评估。'
  }
  return '已关联教师授课记录，可以直接进入评估。'
})

const focusActionLabel = computed(() => {
  if (!focusPlan.value) return '暂无任务'
  if (focusIsFuture.value) return '未到授课日'
  if (focusPlan.value.sessionId === null) return '等待教师建档'
  return '去评估'
})

const focusCanEvaluate = computed(
  () => Boolean(focusPlan.value && !focusIsFuture.value && focusPlan.value.sessionId !== null)
)

function handleFocus(): void {
  if (!focusPlan.value || focusIsFuture.value) return
  if (focusPlan.value.sessionId === null) {
    ElMessage.info('教师尚未创建本场授课记录，请等待教师下课后刷新')
    return
  }
  void router.push({ name: 'session-evaluation', params: { id: focusPlan.value.sessionId } })
}

/** 已评估记录的复盘入口：课程列表 → 课程综合评分 → 历史授课记录。 */
function goSupervisorCourses(): void {
  void router.push({ name: 'course-list' })
}

/** 授课记录已创建但未评价：直达评估页。 */
function handleQueueEvaluate(plan: SupervisionPlan): void {
  if (plan.sessionId === null) return
  void router.push({ name: 'session-evaluation', params: { id: plan.sessionId } })
}

/** 待 AI 评析授课记录：直达评估页。 */
function goPendingEvaluate(sessionId: number): void {
  void router.push({ name: 'session-evaluation', params: { id: sessionId } })
}

/* 督导：抽样任务队列（B2/T1，R1.2）——每满 5 条 AI 终评抽 1 条待人工复评 */
const sampling = ref<SamplingTask[]>([])
const samplingLoading = ref(false)

async function loadSampling(): Promise<void> {
  if (auth.role !== 'supervisor') return
  samplingLoading.value = true
  try {
    const page = await fetchSamplingTasksApi({ semester: semester.value, pending: true, pageSize: 8 })
    sampling.value = page.list ?? []
  } catch {
    sampling.value = []
  } finally {
    samplingLoading.value = false
  }
}

function goSamplingEvaluate(task: SamplingTask): void {
  void router.push({ name: 'session-evaluation', params: { id: task.sessionId } })
}

function reloadQuality(): void {
  void loadData()
  if (auth.hasRole('director')) void loadDirectorScores()
  if (auth.role === 'supervisor') void loadSampling()
}

watch(semester, reloadQuality)

onMounted(reloadQuality)
</script>

<template>
  <div class="dashboard">
    <header class="dashboard__greeting">
      <div>
        <h1 class="dashboard__hello">{{ greeting }}，{{ auth.user?.name }}</h1>
        <p class="dashboard__role">{{ workbenchTitle }}，{{ semester }}</p>
      </div>
      <p class="dashboard__ethics">评分仅用于教学支持与改进，不作为考核依据</p>
    </header>

    <EmptyState v-if="error" description="工作台数据加载失败">
      <template #action>
        <el-button type="primary" @click="reloadQuality">重新加载</el-button>
      </template>
    </EmptyState>

    <template v-else>
      <!-- 主任：事务管理周视图 -->
      <template v-if="auth.role === 'director'">
        <ScheduleBoard class="dashboard__schedule" />

        <div class="director-grid">
          <section class="board">
            <header class="board__head">
              <h2 class="board__title">本室质量热力</h2>
              <span class="board__sub">只标出需要关注的教师，点击一行下钻画像</span>
            </header>
            <ScoreHeatmap
              :teachers="teacherScores"
              :loading="teacherScoresLoading"
              highlight-abnormal
              @select="goTeacherQuality"
            />
          </section>

          <aside class="director-side">
            <section class="board">
              <h2 class="board__title">快捷入口</h2>
              <div class="quick-entries">
                <button type="button" class="quick-entry" @click="goNewCourse">
                  <span class="quick-entry__icon"><el-icon><Plus /></el-icon></span>
                  <span class="quick-entry__label">新增课程</span>
                </button>
                <button type="button" class="quick-entry" @click="goCourseList">
                  <span class="quick-entry__icon"><el-icon><Collection /></el-icon></span>
                  <span class="quick-entry__label">进入课程库</span>
                </button>
                <button type="button" class="quick-entry" @click="goTeacherList">
                  <span class="quick-entry__icon"><el-icon><UserFilled /></el-icon></span>
                  <span class="quick-entry__label">进入教师画像</span>
                </button>
              </div>

              <div v-if="weakestDimension" class="weakness-card">
                <span class="weakness-card__stripe" aria-hidden="true" />
                <div class="weakness-card__body">
                  <span class="weakness-card__label">维度均分最低</span>
                  <p class="weakness-card__name">
                    {{ weakestDimension.name }}
                    <strong class="score-num">{{ formatScore(weakestDimension.avg) }}</strong>
                  </p>
                  <p class="weakness-card__advice">{{ weakestAdvice }}</p>
                </div>
              </div>
            </section>
          </aside>
        </div>

        <ClassroomIssueBoard class="dashboard__issues" />
      </template>

      <!-- 督导：今日待评横带 + 两组任务流 -->
      <template v-else-if="auth.role === 'supervisor'">
        <section class="focus-band anim-fade-slide">
          <div class="focus-band__main">
            <p class="focus-band__date">{{ focusIsToday ? '今天' : '下一节' }} {{ focusDateLabel }}</p>
            <h2 class="focus-band__title">
              {{ focusPlan?.courseName || '今天没有待评安排' }}
            </h2>
            <p v-if="focusPlan" class="focus-band__meta">
              <span>{{ focusPlan.teacherName }}</span>
              <span>{{ focusClass }}</span>
              <span>{{ focusPeriod }}</span>
              <span>{{ focusLocation }}</span>
            </p>
            <p class="focus-band__status">{{ focusStatus }}</p>
          </div>

          <div class="focus-band__action">
            <el-tooltip
              :disabled="!focusIsFuture"
              content="授课日期未到，评估入口在当天开放"
              placement="top"
            >
              <span>
                <el-button
                  type="primary"
                  size="large"
                  :disabled="!focusCanEvaluate"
                  @click="handleFocus"
                >
                  {{ focusActionLabel }}
                </el-button>
              </span>
            </el-tooltip>
            <span v-if="focusIsToday && todayPlans.length > 1" class="focus-band__count">
              今天还有 {{ todayPlans.length - 1 }} 节待评
            </span>
          </div>
        </section>

        <div class="supervisor-grid">
          <section class="board">
            <header class="board__head">
              <h2 class="board__title">接下来</h2>
              <span class="board__sub">按授课日期顺序</span>
            </header>
            <EvaluationQueue
              :plans="otherUpcoming"
              :loading="loading"
              empty-text="接下来暂无待评安排"
              @evaluate="handleQueueEvaluate"
            />

            <div class="makeup">
              <header class="board__head">
                <h2 class="board__title">补录</h2>
                <span class="board__sub">已过授课日期且尚未评价</span>
              </header>
              <EvaluationQueue
                :plans="pastPlans"
                :loading="loading"
                empty-text="没有需要补录的课次"
                @evaluate="handleQueueEvaluate"
              />
            </div>
          </section>

          <aside class="supervisor-side">
            <section class="board">
              <header class="board__head">
                <h2 class="board__title">抽样复评</h2>
                <span class="board__sub">每 5 条 AI 终评抽 1 条</span>
              </header>
              <el-skeleton v-if="samplingLoading" :rows="3" animated />
              <ul v-else-if="sampling.length" class="sampling">
                <li v-for="task in sampling" :key="task.sessionId" class="sampling__row">
                  <button type="button" class="sampling__link" @click="goSamplingEvaluate(task)">
                    <span class="sampling__course">{{ task.courseName }}</span>
                    <span class="sampling__meta">
                      {{ task.teacherName }}
                      {{ task.reason === 'risk' ? '高风险' : 'AI 最低分' }}
                    </span>
                  </button>
                </li>
              </ul>
              <p v-else class="board__empty">暂无待复评样本</p>

              <div class="quick-links quick-links--tight">
                <button type="button" class="text-action" @click="goSupervisorCourses">课程列表</button>
              </div>
            </section>
          </aside>
        </div>

        <section class="board board--pending">
          <header class="board__head">
            <h2 class="board__title">待 AI 评析</h2>
            <span class="board__sub">手动新增或 AI 尚未评析的授课记录</span>
          </header>

          <ol v-if="(data?.pendingSessions ?? []).length" class="session-track">
            <li
              v-for="row in data?.pendingSessions ?? []"
              :key="row.sessionId"
              class="session-track__row"
            >
              <div class="session-track__rail">
                <span class="session-track__date tabular-nums">{{ row.sessionDate }}</span>
                <span class="session-track__period">{{ row.period || '节次待定' }}</span>
              </div>
              <div class="session-track__body">
                <p class="session-track__course">{{ row.courseName }}</p>
                <p class="session-track__topic">{{ row.topic || '未填写主题' }}</p>
              </div>
              <span class="session-track__teacher">{{ row.teacherName }}</span>
              <span class="session-track__status">待 AI 评析</span>
              <button
                type="button"
                class="text-action text-action--next"
                @click="goPendingEvaluate(row.sessionId)"
              >
                去评估
              </button>
            </li>
          </ol>
          <p v-else class="board__empty">暂无待 AI 评析的授课记录</p>
        </section>

        <ClassroomIssueBoard class="dashboard__issues" />
      </template>
    </template>
  </div>
</template>

<style scoped lang="scss">
.dashboard {
  &__greeting {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-4);
    align-items: flex-end;
    justify-content: space-between;
    margin-bottom: var(--spacing-6);
  }

  &__hello {
    font-size: var(--font-size-3xl);
    color: var(--color-text-primary);
  }

  &__role {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__ethics {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

}

/* ===== 主任：决策带（与教师首页同一块「黑板」） ===== */
.decision {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(320px, 0.85fr) auto;
  gap: var(--spacing-8);
  align-items: end;
  padding: var(--spacing-8);
  color: var(--color-ink-text);
  background-color: var(--color-ink);
  border-radius: var(--radius-lg);
}

.decision__lead {
  min-width: 0;
}

.decision__date {
  font-size: var(--font-size-sm);
  color: var(--color-ink-text-dim);
}

.decision__title {
  margin-top: var(--spacing-3);
  font-size: 34px;
  font-weight: 600;
  line-height: 1.2;
  color: #ffffff;
}

.decision__sentence {
  margin-top: var(--spacing-2);
  font-size: var(--font-size-sm);
  line-height: 1.7;
  color: var(--color-ink-text);
}

.decision__people {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.decision__person-link {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 64px 96px;
  gap: var(--spacing-3);
  align-items: baseline;
  width: 100%;
  padding: var(--spacing-3) 0;
  font: inherit;
  text-align: left;
  cursor: pointer;
  background: none;
  border: none;
  border-bottom: 1px solid var(--color-ink-2);

  &:hover .decision__name {
    text-decoration: underline;
    text-underline-offset: 4px;
  }

  &:focus-visible {
    outline: 2px solid var(--color-primary-hover);
    outline-offset: 3px;
    border-radius: 2px;
  }
}

.decision__name {
  overflow: hidden;
  font-size: var(--font-size-xl);
  font-weight: 600;
  color: #ffffff;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.decision__score {
  font-size: var(--font-size-score-md);
  color: var(--color-ink-text);
  text-align: right;
}

.decision__weak {
  font-size: var(--font-size-xs);
  color: var(--color-ink-text-dim);
  text-align: right;
}

.decision__empty {
  font-size: var(--font-size-sm);
  color: var(--color-ink-text-dim);
}

.decision__link {
  align-self: end;
  padding: 0;
  font: inherit;
  font-size: var(--font-size-sm);
  color: var(--color-ink-text);
  cursor: pointer;
  background: none;
  border: none;
  border-bottom: 1px solid var(--color-ink-3);

  &:hover {
    color: #ffffff;
  }

  &:focus-visible {
    outline: 2px solid var(--color-primary-hover);
    outline-offset: 3px;
    border-radius: 2px;
  }
}

.director-grid {
  display: grid;
  grid-template-columns: minmax(0, 2.2fr) minmax(280px, 0.8fr);
  gap: var(--spacing-4);
  align-items: start;
  margin-top: var(--spacing-4);
}

.director-side {
  min-width: 0;
}

/* ===== 通用纸面区块 ===== */
.board {
  padding: var(--spacing-6);
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);

  & + & {
    margin-top: var(--spacing-4);
  }

  &__head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2) var(--spacing-3);
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__title {
    font-size: var(--font-size-xl);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__sub {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__empty {
    padding: var(--spacing-4) 0;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__note {
    padding-top: var(--spacing-4);
    margin-top: var(--spacing-4);
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-secondary);
    border-top: 1px solid var(--color-divider);
  }

  &--pending {
    margin-top: var(--spacing-4);
  }
}

.quick-links {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--spacing-2);

  &--tight {
    padding-top: var(--spacing-4);
    margin-top: var(--spacing-4);
    border-top: 1px solid var(--color-divider);
  }
}

/* ===== 主任快捷入口：图标 + 文字按钮卡 ===== */
.quick-entries {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.quick-entry {
  display: flex;
  gap: var(--spacing-3);
  align-items: center;
  width: 100%;
  padding: var(--spacing-3);
  font: inherit;
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  text-align: left;
  cursor: pointer;
  background-color: var(--color-bg-page);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-md);
  transition: border-color var(--duration-fast) var(--ease-out-soft), background-color var(--duration-fast) var(--ease-out-soft);

  &:hover,
  &:focus-visible {
    background-color: var(--color-primary-bg);
    border-color: color-mix(in srgb, var(--color-primary) 40%, #ffffff);
    outline: none;
  }

  &:focus-visible {
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-primary) 35%, transparent);
  }

  &__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    font-size: 16px;
    color: var(--color-primary);
    background-color: var(--color-bg-card);
    border-radius: var(--radius-sm);
  }

  &__label {
    font-weight: 500;
  }
}

/* ===== 维度短板：色块卡 + 分值 + 行动建议 ===== */
.weakness-card {
  display: flex;
  gap: var(--spacing-3);
  padding: var(--spacing-3);
  margin-top: var(--spacing-4);
  background-color: color-mix(in srgb, var(--color-warning) 9%, #ffffff);
  border: 1px solid color-mix(in srgb, var(--color-warning) 28%, #ffffff);
  border-radius: var(--radius-md);

  &__stripe {
    flex-shrink: 0;
    width: 4px;
    background-color: var(--color-warning);
    border-radius: 2px;
  }

  &__body {
    min-width: 0;
  }

  &__label {
    font-size: var(--font-size-xs);
    color: #8a5a00;
  }

  &__name {
    margin-top: 2px;
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-primary);

    strong {
      margin-left: var(--spacing-2);
      font-size: var(--font-size-lg);
      color: #8a5a00;
    }
  }

  &__advice {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-xs);
    line-height: 1.6;
    color: var(--color-text-secondary);
  }
}

.dashboard__schedule {
  margin-bottom: var(--spacing-4);
}

.dashboard__issues {
  margin-top: var(--spacing-4);
}

.text-action {
  padding: 0;
  font: inherit;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  cursor: pointer;
  background: none;
  border: none;

  &:hover {
    color: var(--color-primary);
    text-decoration: underline;
    text-underline-offset: 3px;
  }

  &:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 3px;
    border-radius: 2px;
  }

  &--next {
    color: var(--color-primary);
  }
}

/* ===== 督导：今日待评横带 ===== */
.focus-band {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--spacing-8);
  align-items: end;
  padding: var(--spacing-8);
  color: var(--color-ink-text);
  background-color: var(--color-ink);
  border-radius: var(--radius-lg);

  &__date {
    font-size: var(--font-size-sm);
    color: var(--color-ink-text-dim);
  }

  &__title {
    margin-top: var(--spacing-3);
    font-size: 34px;
    font-weight: 600;
    line-height: 1.2;
    color: #ffffff;
  }

  &__meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2) var(--spacing-4);
    margin-top: var(--spacing-3);
    font-size: var(--font-size-base);
    color: var(--color-ink-text);
  }

  &__status {
    margin-top: var(--spacing-2);
    font-size: var(--font-size-sm);
    color: var(--color-ink-text-dim);
  }

  &__action {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    align-items: flex-end;
  }

  &__count {
    font-size: var(--font-size-xs);
    color: var(--color-ink-text-dim);
  }
}

.supervisor-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.9fr) minmax(280px, 0.8fr);
  gap: var(--spacing-4);
  align-items: start;
  margin-top: var(--spacing-4);
}

.supervisor-side {
  min-width: 0;
}

.makeup {
  padding-top: var(--spacing-6);
  margin-top: var(--spacing-2);
  border-top: 1px solid var(--color-divider);
}

/* ===== 抽样复评：侧栏小字行 ===== */
.sampling {
  display: flex;
  flex-direction: column;

  &__row {
    border-top: 1px solid var(--color-divider);

    &:first-child {
      border-top: none;
    }
  }

  &__link {
    display: flex;
    flex-direction: column;
    gap: 2px;
    width: 100%;
    padding: var(--spacing-2) 0;
    font: inherit;
    text-align: left;
    cursor: pointer;
    background: none;
    border: none;

    &:hover .sampling__course {
      color: var(--color-primary);
    }

    &:focus-visible {
      outline: 2px solid var(--color-primary);
      outline-offset: 3px;
      border-radius: 2px;
    }
  }

  &__course {
    font-size: var(--font-size-sm);
    font-weight: 500;
    color: var(--color-text-primary);
  }

  &__meta {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }
}

/* ===== 待 AI 评析：时间轨行 ===== */
.session-track {
  display: flex;
  flex-direction: column;

  &__row {
    display: grid;
    grid-template-columns: 108px minmax(0, 1fr) 96px 64px 56px;
    gap: var(--spacing-4);
    align-items: center;
    padding: var(--spacing-3) 0;
    border-top: 1px solid var(--color-divider);

    &:first-child {
      border-top: none;
    }
  }

  &__rail {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  &__date {
    font-family: var(--font-score);
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__period {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__body {
    min-width: 0;
  }

  &__course {
    overflow: hidden;
    font-size: var(--font-size-base);
    font-weight: 500;
    color: var(--color-text-primary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__topic {
    overflow: hidden;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__teacher {
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__status {
    font-size: var(--font-size-xs);
    color: var(--color-warning);
  }
}

@media (max-width: 1080px) {
  .decision {
    grid-template-columns: 1fr;
    align-items: start;
  }

  .decision__title {
    font-size: 28px;
  }

  .decision__link {
    justify-self: start;
  }

  .director-grid,
  .supervisor-grid {
    grid-template-columns: 1fr;
  }

  .focus-band {
    grid-template-columns: 1fr;
  }

  .focus-band__title {
    font-size: 28px;
  }

  .focus-band__action {
    align-items: flex-start;
  }

  .session-track__row {
    grid-template-columns: 96px minmax(0, 1fr) 88px 56px;

    & .session-track__status {
      display: none;
    }
  }
}

@media (max-width: 840px) {
  .board,
  .decision,
  .focus-band {
    padding: var(--spacing-5, 20px);
  }

  .session-track__row {
    grid-template-columns: 88px minmax(0, 1fr) 56px;

    & .session-track__teacher {
      display: none;
    }
  }
}

@media (prefers-reduced-motion: reduce) {
  .decision,
  .focus-band {
    animation: none;
  }
}
</style>
