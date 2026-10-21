<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
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
import { createSessionApi, fetchSessionDetailApi } from '@/api/session'
import { fetchCoursesApi, fetchCourseDetailApi } from '@/api/course'
import { fetchMyTimetableApi } from '@/api/dict'
import type { TimetableItem } from '@/types/dict'
import { useAuthStore } from '@/stores/auth'
import { useDictStore } from '@/stores/dict'
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
const dict = useDictStore()
const { semester } = useSemester()

const loading = ref(true)
const error = ref(false)
const data = ref<DashboardData | null>(null)
const creatingPlanId = ref<number | null>(null)

const WEEK_LABELS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

/* 督导：新增授课记录（替代教务系统录入环节） */
const createVisible = ref(false)
const createSubmitting = ref(false)
const courses = ref<Course[]>([])
const coursesLoading = ref(false)
/* 课表下拉：选定教师后按课表收敛课程/班级；课表为空时回退到课程库 */
const timetable = ref<TimetableItem[]>([])
const timetableLoading = ref(false)
const createForm = reactive<{
  teacherId: number | ''
  courseId: number | ''
  classId: number | ''
  sessionDate: string
  period: string
  topic: string
}>({
  teacherId: '',
  courseId: '',
  classId: '',
  sessionDate: '',
  period: '',
  topic: ''
})

/** 选定授课教师后，课程下拉只呈现该教师的课程（授课教师由课程决定，保证一致） */
const filteredCourses = computed(() => {
  if (createForm.teacherId === '') return courses.value
  if (timetable.value.length) {
    const seen = new Map<number, { id: number; name: string; code: string }>()
    for (const row of timetable.value) {
      if (!seen.has(row.courseId)) {
        seen.set(row.courseId, { id: row.courseId, name: row.courseName, code: row.courseCode })
      }
    }
    return [...seen.values()]
  }
  return courses.value.filter((course) => course.teacherId === createForm.teacherId)
})

/** 选定课程后，班级下拉只呈现该课在课表中的班级（后端建课要求 classId 必填） */
const courseClasses = ref<Array<{ id: number; className: string }>>([])
const courseClassesLoading = ref(false)
const classOptions = computed(() => {
  if (createForm.courseId === '') return []
  const fromTimetable = timetable.value
    .filter((row) => row.courseId === createForm.courseId)
    .map((row) => ({ id: row.classId, name: row.className }))
  /* 课表未维护该课时回退到课程开课班级，保证对话框不被卡死 */
  if (fromTimetable.length) return fromTimetable
  return courseClasses.value.map((c) => ({ id: c.id, name: c.className }))
})

async function loadCourseClasses(courseId: number | ''): Promise<void> {
  courseClasses.value = []
  if (courseId === '') return
  courseClassesLoading.value = true
  try {
    const detail = await fetchCourseDetailApi(courseId)
    courseClasses.value = (detail.classes ?? []).map((c) => ({ id: c.id, className: c.className }))
  } catch {
    courseClasses.value = []
  } finally {
    courseClassesLoading.value = false
  }
}

async function loadTimetable(teacherId: number | ''): Promise<void> {
  timetable.value = []
  if (teacherId === '') return
  timetableLoading.value = true
  try {
    timetable.value = await fetchMyTimetableApi({ teacherId }) ?? []
  } catch {
    timetable.value = []
  } finally {
    timetableLoading.value = false
  }
}

watch(
  () => createForm.teacherId,
  (teacherId) => {
    if (
      createForm.courseId !== '' &&
      !filteredCourses.value.some((course) => course.id === createForm.courseId)
    ) {
      createForm.courseId = ''
    }
    createForm.classId = ''
    void loadTimetable(teacherId)
  }
)

watch(
  () => createForm.courseId,
  (courseId) => {
    createForm.classId = ''
    /* 课表无该课班级时回退拉取课程开课班级 */
    if (!timetable.value.some((row) => row.courseId === courseId)) {
      void loadCourseClasses(courseId)
    } else {
      courseClasses.value = []
    }
  }
)

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
  if (focusPlan.value.sessionId === null) return '尚未建档，点击后创建授课记录并进入评估。'
  return '已有授课记录，可以直接进入评估。'
})

const focusActionLabel = computed(() => {
  if (!focusPlan.value) return '暂无任务'
  return focusIsFuture.value ? '未到授课日' : '去评估'
})

function handleFocus(): void {
  if (!focusPlan.value || focusIsFuture.value) return
  if (focusPlan.value.sessionId === null) {
    void handleQueueCreate(focusPlan.value)
    return
  }
  void router.push({ name: 'session-evaluation', params: { id: focusPlan.value.sessionId } })
}

/** 今日待评课：一键创建授课记录并直达评估页（避免多级点击）。 */
async function handleQueueCreate(plan: SupervisionPlan): Promise<void> {
  creatingPlanId.value = plan.id
  try {
    const session = await createSessionApi({
      courseId: plan.courseId,
      sessionDate: plan.plannedDate,
      period: '待定',
      planId: plan.id
    })
    ElMessage.success('授课记录已创建，开始评估')
    void router.push({ name: 'session-evaluation', params: { id: session.id } })
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    creatingPlanId.value = null
  }
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

/** 待评估授课记录：直达评估页。 */
function goPendingEvaluate(sessionId: number): void {
  void router.push({ name: 'session-evaluation', params: { id: sessionId } })
}

async function ensureCourses(): Promise<void> {
  if (courses.value.length) return
  coursesLoading.value = true
  try {
    const page = await fetchCoursesApi({ page: 1, pageSize: 100 })
    courses.value = page.list ?? []
  } catch {
    courses.value = []
  } finally {
    coursesLoading.value = false
  }
}

async function openCreateSession(): Promise<void> {
  createForm.teacherId = ''
  createForm.courseId = ''
  createForm.classId = ''
  createForm.sessionDate = ''
  createForm.period = ''
  createForm.topic = ''
  timetable.value = []
  createVisible.value = true
  await Promise.all([dict.load(), ensureCourses()])
}

function disabledFuture(date: Date): boolean {
  return date.getTime() > Date.now()
}

async function submitCreateSession(): Promise<void> {
  if (createForm.courseId === '') {
    ElMessage.error('请选择课程')
    return
  }
  if (createForm.classId === '') {
    ElMessage.error('请选择班级')
    return
  }
  if (!createForm.sessionDate) {
    ElMessage.error('请选择授课日期')
    return
  }
  if (!createForm.period.trim()) {
    ElMessage.error('请填写节次')
    return
  }
  createSubmitting.value = true
  try {
    await createSessionApi({
      courseId: createForm.courseId,
      classId: createForm.classId,
      sessionDate: createForm.sessionDate,
      period: createForm.period.trim(),
      topic: createForm.topic.trim()
    })
    ElMessage.success('授课记录已创建，可在「待评估授课记录」中评估')
    createVisible.value = false
    await loadData()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    createSubmitting.value = false
  }
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
                  :disabled="!focusPlan || focusIsFuture"
                  :loading="creatingPlanId !== null"
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
              @create="handleQueueCreate"
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
                @create="handleQueueCreate"
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
                <button type="button" class="text-action" @click="openCreateSession">新增授课记录</button>
              </div>
            </section>
          </aside>
        </div>

        <section class="board board--pending">
          <header class="board__head">
            <h2 class="board__title">待评估授课记录</h2>
            <span class="board__sub">手动新增或尚未评价的授课记录</span>
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
              <span class="session-track__status">未评估</span>
              <button
                type="button"
                class="text-action text-action--next"
                @click="goPendingEvaluate(row.sessionId)"
              >
                去评估
              </button>
            </li>
          </ol>
          <p v-else class="board__empty">暂无待评估授课记录</p>
        </section>

        <ClassroomIssueBoard class="dashboard__issues" />
      </template>
    </template>

    <!-- 新增授课记录（督导，替代教务系统录入环节） -->
    <el-dialog v-model="createVisible" title="新增授课记录" width="520px">
      <el-form label-width="88px">
        <el-form-item label="授课教师" required>
          <el-select
            v-model="createForm.teacherId"
            placeholder="请选择授课教师"
            filterable
            class="dashboard__dialog-field"
          >
            <el-option
              v-for="teacher in dict.teachers"
              :key="teacher.id"
              :label="teacher.name"
              :value="teacher.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="课程" required>
          <el-select
            v-model="createForm.courseId"
            placeholder="请选择课程"
            filterable
            :loading="coursesLoading"
            class="dashboard__dialog-field"
          >
            <el-option
              v-for="course in filteredCourses"
              :key="course.id"
              :label="`${course.name}（${course.code}）`"
              :value="course.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="班级" required>
          <el-select
            v-model="createForm.classId"
            placeholder="请选择班级"
            filterable
            :loading="timetableLoading || courseClassesLoading"
            :disabled="createForm.courseId === ''"
            class="dashboard__dialog-field"
          >
            <el-option
              v-for="cls in classOptions"
              :key="cls.id"
              :label="cls.name"
              :value="cls.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="授课日期" required>
          <el-date-picker
            v-model="createForm.sessionDate"
            type="date"
            placeholder="不得晚于今天"
            value-format="YYYY-MM-DD"
            :disabled-date="disabledFuture"
            class="dashboard__dialog-field"
          />
        </el-form-item>
        <el-form-item label="节次" required>
          <el-input
            v-model="createForm.period"
            placeholder="如 3-4 节"
            maxlength="32"
            class="dashboard__dialog-field"
          />
        </el-form-item>
        <el-form-item label="主题">
          <el-input
            v-model="createForm.topic"
            placeholder="本次课主题（选填）"
            maxlength="128"
            class="dashboard__dialog-field"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createSubmitting" @click="submitCreateSession">
          创建
        </el-button>
      </template>
    </el-dialog>
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

  &__dialog-field {
    width: 100%;
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

/* ===== 待评估授课记录：时间轨行 ===== */
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
