<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import StatCard from '@/components/common/StatCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import QualityBadge from '@/components/common/QualityBadge.vue'
import ScoreHeatmap from '@/components/evaluation/ScoreHeatmap.vue'
import EvaluationQueue from '@/components/supervision/EvaluationQueue.vue'
import { fetchDashboardApi } from '@/api/dashboard'
import { fetchTeacherScoresApi } from '@/api/teacher'
import { fetchSamplingTasksApi } from '@/api/supervision'
import { createSessionApi } from '@/api/session'
import { fetchCoursesApi, fetchCourseDetailApi } from '@/api/course'
import { fetchMyTimetableApi } from '@/api/dict'
import type { TimetableItem } from '@/types/dict'
import { useAuthStore } from '@/stores/auth'
import { useDictStore } from '@/stores/dict'
import { useSemester } from '@/composables/useSemester'
import { EVALUATION_DIMENSIONS, getRoleMeta } from '@/constants'
import type { DashboardData, SamplingTask, SupervisionPlan } from '@/types/supervision'
import type { Course } from '@/types/course'
import type { TeacherScoreItem } from '@/types/teacher'

/**
 * 工作台 / 质量驾驶舱。
 *   - 主任：本室质量热力 + 重点关注（质量驾驶舱）；
 *   - 督导：聚焦「记录课程并评估」的工作台——待评课队列 + 草稿箱。
 * 教师端不设质量驾驶舱（信息与「我的质量档案」重复），登录后直达质量档案。
 */
const router = useRouter()
const auth = useAuthStore()
const dict = useDictStore()
const { semester } = useSemester()

const loading = ref(true)
const error = ref(false)
const data = ref<DashboardData | null>(null)
const creatingPlanId = ref<number | null>(null)

/* 督导：新增授课记录（替代教务系统录入环节） */
const createVisible = ref(false)
const createSubmitting = ref(false)
const courses = ref<Course[]>([])
const coursesLoading = ref(false)
/* B11 课表下拉：选定教师后按课表收敛课程/班级；课表为空时回退到课程库 */
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

const roleLabel = computed(() => getRoleMeta(auth.user?.role)?.label ?? '')
const workbenchTitle = computed(() => (auth.role === 'supervisor' ? '工作台' : '质量驾驶舱'))

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
    <div class="dashboard__greeting">
      <div>
        <h1 class="dashboard__hello">{{ greeting }}，{{ auth.user?.name }}</h1>
        <p class="dashboard__role">{{ roleLabel }} · {{ workbenchTitle }}（{{ semester }}）</p>
      </div>
      <p class="dashboard__ethics">评分仅用于教学支持与改进，不作为考核依据</p>
    </div>

    <div v-if="error" class="dashboard__error">
      <EmptyState description="工作台数据加载失败">
        <template #action>
          <el-button type="primary" @click="reloadQuality">重新加载</el-button>
        </template>
      </EmptyState>
    </div>

    <template v-else>
      <!-- 主任视角：统计卡 + 本室质量热力 + 重点关注 -->
      <template v-if="auth.role === 'director'">
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

        <div class="dashboard__grid">
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
      </template>

      <!-- 督导视角：待评课队列 + 草稿箱 + 待评估授课记录 -->
      <template v-else-if="auth.role === 'supervisor'">
        <div class="dashboard__grid dashboard__grid--workbench">
          <section class="dashboard__panel">
            <div class="dashboard__panel-head">
              <h2 class="dashboard__panel-title">待评课队列</h2>
              <div class="dashboard__panel-actions">
                <span class="dashboard__panel-sub">
                  今日 / 本周 / 本月 / 未评 · 已评估记录请到「课程列表」查看
                </span>
                <el-button size="small" @click="goSupervisorCourses">课程列表</el-button>
                <el-button type="primary" size="small" @click="openCreateSession">
                  新增授课记录
                </el-button>
              </div>
            </div>
            <EvaluationQueue
              :plans="data?.plans ?? []"
              :loading="loading"
              @create="handleQueueCreate"
              @evaluate="handleQueueEvaluate"
            />
          </section>

          <aside class="dashboard__side">
            <section class="dashboard__panel">
              <div class="dashboard__panel-head">
                <h2 class="dashboard__panel-title">抽样复评队列</h2>
                <span class="dashboard__panel-sub">每 5 条 AI 终评抽 1 条</span>
              </div>
              <el-skeleton v-if="samplingLoading" :rows="3" animated />
              <ul v-else-if="sampling.length" class="dashboard__sampling">
                <li
                  v-for="(task, index) in sampling"
                  :key="task.sessionId"
                  class="dashboard__sample anim-rise"
                  :style="{ '--stagger': `${Math.min(index, 6) * 50}ms` }"
                  @click="goSamplingEvaluate(task)"
                >
                  <div class="dashboard__sample-head">
                    <span class="dashboard__sample-course">{{ task.courseName }}</span>
                    <el-tag
                      size="small"
                      :type="task.reason === 'risk' ? 'danger' : 'info'"
                      effect="light"
                      round
                    >
                      {{ task.reason === 'risk' ? '高风险' : 'AI 最低分' }}
                    </el-tag>
                  </div>
                  <p class="dashboard__sample-meta">
                    {{ task.teacherName }} · {{ task.sessionDate }} · 第 {{ task.batch }} 批
                  </p>
                  <p v-if="task.riskTags.length" class="dashboard__sample-risks">
                    {{ task.riskTags.join('、') }}
                  </p>
                </li>
              </ul>
              <EmptyState v-else description="暂无待复评样本" />
            </section>
          </aside>
        </div>

        <!-- 待评估授课记录：手动新增或尚未评价的授课记录，直接去评估 -->
        <section class="dashboard__panel dashboard__panel--pending">
          <div class="dashboard__panel-head">
            <h2 class="dashboard__panel-title">待评估授课记录</h2>
            <span class="dashboard__panel-sub">手动新增或尚未评价的授课记录 · 点击「去评估」进入</span>
          </div>
          <el-table
            v-if="(data?.pendingSessions ?? []).length"
            :data="data?.pendingSessions ?? []"
            row-key="sessionId"
            stripe
          >
            <el-table-column label="授课日期" min-width="120">
              <template #default="{ row }">
                <span class="tabular-nums">{{ row.sessionDate }}</span>
              </template>
            </el-table-column>
            <el-table-column label="节次" min-width="100">
              <template #default="{ row }">{{ row.period || '—' }}</template>
            </el-table-column>
            <el-table-column label="课程" min-width="200">
              <template #default="{ row }">
                <span class="dashboard__pending-course">{{ row.courseName }}</span>
                <el-tag v-if="row.courseCode" size="small" effect="plain">
                  {{ row.courseCode }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="teacherName" label="授课教师" min-width="120" />
            <el-table-column label="主题" min-width="160">
              <template #default="{ row }">{{ row.topic || '—' }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default>
                <el-tag size="small" type="warning" effect="plain">未评估</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="110" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="goPendingEvaluate(row.sessionId)">
                  去评估
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <EmptyState v-else description="暂无待评估授课记录" />
        </section>
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

    &--workbench {
      grid-template-columns: 1.7fr 1fr;
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

  &__panel-actions {
    display: flex;
    gap: var(--spacing-3);
    align-items: center;
  }

  &__panel--pending {
    margin-top: var(--spacing-4);
  }

  &__pending-course {
    margin-right: var(--spacing-2);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__dialog-field {
    width: 100%;
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

  &__sampling {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    padding: 0;
    margin: 0;
    list-style: none;
  }

  &__sample {
    padding: var(--spacing-3);
    cursor: pointer;
    background-color: var(--color-bg-page);
    border: 1px solid transparent;
    border-radius: var(--radius-md);
    transition: box-shadow var(--duration-fast) var(--ease-out-soft),
      border-color var(--duration-fast) var(--ease-out-soft),
      transform var(--duration-fast) var(--ease-out-soft);

    &:hover {
      border-color: var(--color-primary-hover);
      box-shadow: var(--shadow-card-hover);
      transform: translateY(-1px);
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

    &-meta {
      margin-top: 2px;
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    &-risks {
      margin-top: var(--spacing-1);
      font-size: var(--font-size-xs);
      color: var(--color-error);
    }
  }
}
</style>
