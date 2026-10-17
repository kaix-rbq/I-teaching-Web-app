<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AiBadge from '@/components/common/AiBadge.vue'
import QualityBadge from '@/components/common/QualityBadge.vue'
import QualityCompass from '@/components/evaluation/QualityCompass.vue'
import ScoreDimensionsCard from '@/components/evaluation/ScoreDimensionsCard.vue'
import { fetchTeacherEvaluationsApi, fetchTeacherSummaryApi } from '@/api/teacher'
import { fetchImprovementsApi } from '@/api/improvement'
import { createSessionApi } from '@/api/session'
import { fetchCourseDetailApi } from '@/api/course'
import { fetchMyTimetableApi } from '@/api/dict'
import type { TimetableItem } from '@/types/dict'
import { useAuthStore } from '@/stores/auth'
import { useSemester } from '@/composables/useSemester'
import { flagText } from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { TeacherSummary, TeacherTimelineItem } from '@/types/teacher'
import type { ImprovementItem } from '@/types/improvement'

/**
 * 我的质量档案（教师首页，替代原「质量驾驶舱」，避免信息重复）。
 * 数据全部来自教师查本人接口；只读。
 *   - 顶部三卡：综合分 / 五维评分 / 提优分析；
 *   - 我的课程评分明细；
 *   - 授课快照：逐次授课卡片化，按质量水平标注亮点或不足。
 * 「全部评价流」「AI 提优建议摘要」已移除：评价信息与具体授课记录绑定，避免信息混乱。
 */
const router = useRouter()
const auth = useAuthStore()
const { semester } = useSemester()

const summary = ref<TeacherSummary | null>(null)
const timeline = ref<TeacherTimelineItem[]>([])
const improvements = ref<ImprovementItem[]>([])
const loading = ref(true)
const error = ref(false)

const flags = computed(() => summary.value?.flags ?? [])
const openImprovements = computed(() => improvements.value.filter((it) => it.status !== 'closed'))
const recentNotes = computed(() =>
  timeline.value.filter((t) => t.supervisorEvaluations?.length || t.agentEvaluation).slice(0, 3)
)

function goImprovements(): void {
  void router.push({ name: 'improvements' })
}

/**
 * 提优分析（预留 AI 接口，当前为示例文本）。
 * 未来由 AI 综合历次授课记录与评价信息生成；接口就绪前展示示例内容。
 */
const improveAnalysis = {
  directions: [
    '学生互动维度相对偏弱：有效提问比例不足，提问后等待时间偏短，学生回应多为简单附和。',
    '内容深度可再加强：真实项目案例偏少，知识点与行业前沿的结合不够自然。'
  ],
  suggestions: [
    '提问后保持 3-5 秒沉默，把「自问自答」改为「点名学生 + 追问一层」，可在每节课挑 2-3 个关键节点刻意练习。',
    '每个核心知识点配 1 个近两年的行业案例，并让学生判断「如果是你会怎么做」，把内容深度与互动一起带起来。'
  ]
}

/** 数字一致性铁律：直接展示后端值，不做前端四舍五入 */
function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

async function load(): Promise<void> {
  const id = auth.user?.id
  if (!id) return

  loading.value = true
  error.value = false
  try {
    summary.value = await fetchTeacherSummaryApi(id, { semester: semester.value || undefined })
  } catch {
    error.value = true
    summary.value = null
    loading.value = false
    return
  }
  loading.value = false

  try {
    const page = await fetchTeacherEvaluationsApi(id, {
      semester: semester.value || undefined,
      pageSize: 50
    })
    timeline.value = page.list
  } catch {
    timeline.value = []
  }

  try {
    const items = await fetchImprovementsApi({ pageSize: 20 })
    improvements.value = items.list ?? []
  } catch {
    improvements.value = []
  }
}

function goImprove(courseId: number): void {
  void router.push({ name: 'course-improve', params: { id: courseId } })
}

/**
 * 授课快照 → 当堂课（授课记录）评价页：查看该次课的督导与 AI 智能体评价，
 * 以及当堂课的转写文本入口，帮助教师回顾授课细节。
 */
function goSessionRecord(sessionId: number): void {
  void router.push({ name: 'session-evaluation', params: { id: sessionId } })
}

/* ===== 教师自建授课（B1）：本人课表下拉课程/班级 ===== */
const createVisible = ref(false)
const createSubmitting = ref(false)
const timetable = ref<TimetableItem[]>([])
const timetableLoading = ref(false)
const createForm = reactive<{
  courseId: number | ''
  classId: number | ''
  sessionDate: string
  period: string
  topic: string
}>({
  courseId: '',
  classId: '',
  sessionDate: '',
  period: '',
  topic: ''
})

const courseOptions = computed(() => {
  const seen = new Map<number, { id: number; name: string; code: string }>()
  for (const row of timetable.value) {
    if (!seen.has(row.courseId)) {
      seen.set(row.courseId, { id: row.courseId, name: row.courseName, code: row.courseCode })
    }
  }
  /* 课表未维护时回退到本人课程（班级再按课程详情回退），对话框不被卡死 */
  if (!seen.size && summary.value) {
    for (const c of summary.value.courses) {
      if (!seen.has(c.courseId)) {
        seen.set(c.courseId, { id: c.courseId, name: c.courseName, code: c.courseCode })
      }
    }
  }
  return [...seen.values()]
})

const fallbackClasses = ref<Array<{ id: number; className: string }>>([])

const classOptions = computed(() => {
  if (createForm.courseId === '') return []
  const fromTimetable = timetable.value
    .filter((row) => row.courseId === createForm.courseId)
    .map((row) => ({ id: row.classId, name: row.className }))
  if (fromTimetable.length) return fromTimetable
  return fallbackClasses.value.map((c) => ({ id: c.id, name: c.className }))
})

function disabledFuture(date: Date): boolean {
  return date.getTime() > Date.now()
}

async function openCreate(): Promise<void> {
  createForm.courseId = ''
  createForm.classId = ''
  createForm.sessionDate = ''
  createForm.period = ''
  createForm.topic = ''
  createVisible.value = true
  timetableLoading.value = true
  try {
    timetable.value = await fetchMyTimetableApi() ?? []
  } catch {
    timetable.value = []
  } finally {
    timetableLoading.value = false
  }
}

async function submitCreate(): Promise<void> {
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
    ElMessage.success('授课记录已创建，可补传录音走 AI 评析')
    createVisible.value = false
    await load()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    createSubmitting.value = false
  }
}

watch(
  () => createForm.courseId,
  (courseId) => {
    createForm.classId = ''
    /* 课表无该课班级时回退拉取课程开课班级 */
    if (courseId === '' || timetable.value.some((row) => row.courseId === courseId)) {
      fallbackClasses.value = []
      return
    }
    void (async () => {
      try {
        const detail = await fetchCourseDetailApi(courseId)
        fallbackClasses.value = (detail.classes ?? []).map((c) => ({ id: c.id, className: c.className }))
      } catch {
        fallbackClasses.value = []
      }
    })()
  }
)

/* ===== 授课快照：按质量水平标注亮点或主要不足 ===== */

function scoreColor(item: TeacherTimelineItem): string {
  return scoreToneColor(scoreTone(item.compositeScore))
}

function isHighScore(item: TeacherTimelineItem): boolean {
  const tone = scoreTone(item.compositeScore)
  return tone === 4 || tone === 5
}

function isLowScore(item: TeacherTimelineItem): boolean {
  const tone = scoreTone(item.compositeScore)
  return tone === 1 || tone === 2 || tone === 3
}

function snapshotHighlights(item: TeacherTimelineItem): string {
  const sup = item.supervisorEvaluations?.[0]
  return sup?.highlights || item.agentEvaluation?.highlights || ''
}

function snapshotImprovements(item: TeacherTimelineItem): string {
  const sup = item.supervisorEvaluations?.[0]
  return sup?.improvements || item.agentEvaluation?.improvements || ''
}

onMounted(() => {
  void load()
})

watch(semester, () => {
  void load()
})
</script>

<template>
  <div class="my-quality">
    <PageHeader title="我的质量档案">
      <template #subtitle>督导与 AI 双源融合的教学质量全景 · 当前学期口径（{{ semester }}）</template>
    </PageHeader>

    <el-alert
      class="my-quality__ethics"
      type="info"
      :closable="false"
      show-icon
      title="仅用于教学支持，不作为考核依据"
      description="评分用于帮助教师改进教学，不用于绩效与人事评价。请勿外传或作他用。"
    />

    <el-skeleton v-if="loading" :rows="10" animated />

    <EmptyState v-else-if="error || !summary" description="质量档案加载失败">
      <template #action>
        <el-button type="primary" @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <template v-else>
      <!-- 三卡横排：我的综合分 | 五维雷达 | 提优分析 -->
      <section class="my-quality__top">
        <div class="my-quality__card">
          <h2 class="my-quality__card-title">我的综合分</h2>
          <QualityCompass :summary="summary" size="lg" />
          <div v-if="flags.length" class="my-quality__flags">
            <el-alert
              v-for="flag in flags"
              :key="flag"
              :title="flagText(flag)"
              type="warning"
              :closable="false"
              show-icon
            />
          </div>
        </div>

        <div class="my-quality__card">
          <ScoreDimensionsCard :summary="summary" />
        </div>

        <div class="my-quality__card my-quality__card--ai">
          <h2 class="my-quality__card-title">
            提优分析
            <el-tag size="small" effect="plain" round>AI · 示例</el-tag>
          </h2>
          <div class="my-quality__analysis">
            <section class="my-quality__analysis-block">
              <h3 class="my-quality__analysis-title">可提优方向</h3>
              <ul class="my-quality__analysis-list">
                <li v-for="(text, index) in improveAnalysis.directions" :key="`dir-${index}`">
                  {{ text }}
                </li>
              </ul>
            </section>
            <section class="my-quality__analysis-block">
              <h3 class="my-quality__analysis-title">提优建议</h3>
              <ul class="my-quality__analysis-list">
                <li v-for="(text, index) in improveAnalysis.suggestions" :key="`sug-${index}`">
                  {{ text }}
                </li>
              </ul>
            </section>
          </div>
          <p class="my-quality__analysis-hint">
            <AiBadge text="AI 生成" /> 综合历次授课记录与评价信息生成，当前为示例内容，接口就绪后自动刷新。
          </p>
        </div>
      </section>

      <!-- 我的课程评分明细 -->
      <section class="my-quality__block">
        <h2 class="my-quality__title">我的课程评分明细</h2>
        <ul v-if="summary.courses.length" class="my-quality__courses">
          <li v-for="course in summary.courses" :key="course.courseId" class="my-quality__course">
            <div class="my-quality__course-info">
              <span class="my-quality__course-code">{{ course.courseCode }}</span>
              <span class="my-quality__course-name">{{ course.courseName }}</span>
            </div>
            <template v-if="course.compositeScore === null">
              <span class="my-quality__muted">暂无评价</span>
            </template>
            <template v-else>
              <span
                class="my-quality__course-score score-num"
                :style="{ color: scoreToneColor(scoreTone(course.compositeScore)) }"
              >
                {{ formatScore(course.compositeScore) }}
              </span>
              <QualityBadge :score="course.compositeScore" />
            </template>
            <span class="my-quality__course-sample tabular-nums">n={{ course.sample.evaluatedCount }}</span>
            <el-button
              class="my-quality__improve-btn"
              type="primary"
              plain
              size="small"
              round
              @click="goImprove(course.courseId)"
            >
              去提优 →
            </el-button>
          </li>
        </ul>
        <p v-else class="my-quality__muted">本学期暂无课程评分</p>
      </section>

      <!-- 授课快照：逐次授课卡片，颜色对应质量水平，标注亮点或不足 -->
      <section class="my-quality__block">
        <div class="my-quality__block-head">
          <h2 class="my-quality__title">授课快照</h2>
          <el-button type="primary" plain size="small" round @click="openCreate">
            + 新增授课记录
          </el-button>
        </div>
        <div v-if="timeline.length" class="my-quality__snapshots">
          <article
            v-for="item in timeline"
            :key="item.sessionId"
            class="my-quality__snapshot"
            :style="{ '--snapshot-color': scoreColor(item) }"
          >
            <header class="my-quality__snapshot-head">
              <div class="my-quality__snapshot-title">
                <span class="my-quality__snapshot-course">{{ item.courseName }}</span>
                <span class="my-quality__snapshot-meta">
                  {{ item.sessionDate }} · {{ item.period || '—' }}
                </span>
              </div>
              <QualityBadge :score="item.compositeScore" />
            </header>
            <p class="my-quality__snapshot-topic">{{ item.topic || '未填写主题' }}</p>

            <div class="my-quality__snapshot-score score-num" :style="{ color: scoreColor(item) }">
              {{ formatScore(item.compositeScore) }}
            </div>

            <p
              v-if="isHighScore(item) && snapshotHighlights(item)"
              class="my-quality__snapshot-note my-quality__snapshot-note--good"
            >
              <span class="my-quality__snapshot-tag">亮点</span>
              {{ snapshotHighlights(item) }}
            </p>
            <p
              v-else-if="isLowScore(item) && snapshotImprovements(item)"
              class="my-quality__snapshot-note my-quality__snapshot-note--warn"
            >
              <span class="my-quality__snapshot-tag">主要不足</span>
              {{ snapshotImprovements(item) }}
            </p>
            <p v-else class="my-quality__snapshot-note my-quality__snapshot-note--muted">
              {{ item.compositeScore === null ? '该次课暂无评价' : '常规授课，暂无特别标注' }}
            </p>

            <el-button
              class="my-quality__improve-btn"
              type="primary"
              plain
              size="small"
              round
              @click="goSessionRecord(item.sessionId)"
            >
              查看详细记录 →
            </el-button>
          </article>
        </div>
        <p v-else class="my-quality__muted">本学期暂无授课记录</p>
      </section>

      <!-- 待改进清单（N4）：教师自评 → 督导确认 -->
      <section class="my-quality__block">
        <div class="my-quality__block-head">
          <h2 class="my-quality__title">待改进清单</h2>
          <el-button link type="primary" @click="goImprovements">全部清单 →</el-button>
        </div>
        <ul v-if="openImprovements.length" class="my-quality__improve-list">
          <li
            v-for="(it, index) in openImprovements.slice(0, 5)"
            :key="it.id"
            class="my-quality__improve-row anim-rise"
            :style="{ '--stagger': `${index * 45}ms` }"
          >
            <span class="my-quality__improve-status" :data-status="it.status">
              {{ it.status === 'done_self' ? '待督导确认' : '进行中' }}
            </span>
            <span class="my-quality__improve-title">{{ it.title }}</span>
          </li>
        </ul>
        <p v-else class="my-quality__muted">
          暂无待改进项，去
          <el-button link type="primary" @click="goImprovements">改进清单</el-button>
          新建一条。
        </p>
      </section>

      <!-- 督导意见与 AI 观察：双源并置（督导实线 / AI 虚线） -->
      <section class="my-quality__block">
        <h2 class="my-quality__title">督导意见与 AI 观察</h2>
        <div v-if="recentNotes.length" class="my-quality__notes">
          <article
            v-for="(item, index) in recentNotes"
            :key="item.sessionId"
            class="my-quality__note anim-rise"
            :style="{ '--stagger': `${index * 60}ms` }"
          >
            <header class="my-quality__note-head">
              <span class="my-quality__note-course">{{ item.courseName }}</span>
              <span class="my-quality__note-date tabular-nums">{{ item.sessionDate }}</span>
            </header>
            <div v-if="item.supervisorEvaluations?.length" class="my-quality__note-block">
              <span class="my-quality__note-tag">督导观察</span>
              <p class="my-quality__note-text">
                {{ item.supervisorEvaluations[0].comment || item.supervisorEvaluations[0].highlights || '—' }}
              </p>
            </div>
            <div v-if="item.agentEvaluation" class="my-quality__note-block ai-panel my-quality__note-block--ai">
              <span class="my-quality__note-tag my-quality__note-tag--ai">AI 观察</span>
              <p class="my-quality__note-text">
                {{ item.agentEvaluation.highlights || item.agentEvaluation.comment || '—' }}
              </p>
            </div>
          </article>
        </div>
        <p v-else class="my-quality__muted">本学期暂无评价意见</p>
      </section>
    </template>
  </div>

  <!-- 教师自建授课（B1）：课表下拉课程/班级，建课后可补传录音走 AI 评析 -->
  <el-dialog v-model="createVisible" title="新增授课记录" width="480px">
    <el-form label-width="72px">
      <el-form-item label="课程" required>
        <el-select
          v-model="createForm.courseId"
          placeholder="请选择课程"
          filterable
          :loading="timetableLoading"
          class="my-quality__dialog-field"
        >
          <el-option
            v-for="course in courseOptions"
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
          :disabled="createForm.courseId === ''"
          class="my-quality__dialog-field"
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
          class="my-quality__dialog-field"
        />
      </el-form-item>
      <el-form-item label="节次" required>
        <el-input
          v-model="createForm.period"
          placeholder="如 3-4 节"
          maxlength="32"
          class="my-quality__dialog-field"
        />
      </el-form-item>
      <el-form-item label="主题">
        <el-input
          v-model="createForm.topic"
          placeholder="本次课主题（选填）"
          maxlength="128"
          class="my-quality__dialog-field"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="createVisible = false">取消</el-button>
      <el-button type="primary" :loading="createSubmitting" @click="submitCreate">
        创建
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.my-quality {
  &__ethics {
    margin-bottom: var(--spacing-4);
  }

  &__top {
    display: grid;
    grid-template-columns: minmax(260px, 1fr) minmax(320px, 1.5fr) minmax(280px, 1.2fr);
    gap: var(--spacing-4);
    margin-bottom: var(--spacing-4);

    @media (max-width: 1280px) {
      grid-template-columns: 1fr 1fr;
    }

    @media (max-width: 960px) {
      grid-template-columns: 1fr;
    }
  }

  &__card {
    display: flex;
    flex-direction: column;
    padding: var(--spacing-5);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);

    &--ai {
      background: linear-gradient(180deg, var(--color-ai-bg, #f5f3ff) 0%, var(--color-bg-card) 62%);
    }
  }

  &__card-title {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    margin-bottom: var(--spacing-3);
    padding-bottom: var(--spacing-2);
    font-size: var(--font-size-base);
    color: var(--color-text-primary);
    border-bottom: 1px solid var(--color-divider);
  }

  &__flags {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    margin-top: var(--spacing-4);
  }

  &__analysis {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
    flex: 1;
  }

  &__analysis-block {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
  }

  &__analysis-title {
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-ai, #6d28d9);
  }

  &__analysis-list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-1);
    padding-left: 1.1em;
    margin: 0;
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-secondary);
  }

  &__analysis-hint {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-1);
    align-items: center;
    margin-top: var(--spacing-4);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__block {
    padding: var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__title {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__courses {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    padding: 0;
    margin: 0;
    list-style: none;
  }

  &__course {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-3);
    align-items: center;
    padding: var(--spacing-3) var(--spacing-4);
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-md);
  }

  &__course-info {
    display: flex;
    flex: 1;
    min-width: 220px;
    gap: var(--spacing-2);
    align-items: baseline;
  }

  &__course-code {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__course-name {
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__course-score {
    font-size: var(--font-size-score-md);
    font-weight: 600;
  }

  &__course-sample {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__improve-btn {
    margin-left: auto;
  }

  &__snapshots {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--spacing-4);

    @media (max-width: 1024px) {
      grid-template-columns: 1fr;
    }
  }

  &__snapshot {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    padding: var(--spacing-4) var(--spacing-4) var(--spacing-4) var(--spacing-5);
    overflow: hidden;
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);

    &::before {
      position: absolute;
      top: 0;
      bottom: 0;
      left: 0;
      width: 4px;
      content: '';
      background-color: var(--snapshot-color);
    }
  }

  &__snapshot-head {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    justify-content: space-between;
  }

  &__snapshot-title {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  &__snapshot-course {
    overflow: hidden;
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__snapshot-meta {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__snapshot-topic {
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__snapshot-score {
    font-size: var(--font-size-score-lg);
    font-weight: 600;
    line-height: 1.1;
  }

  &__snapshot-note {
    display: flex;
    gap: var(--spacing-2);
    align-items: flex-start;
    font-size: var(--font-size-sm);
    line-height: 1.6;
    color: var(--color-text-secondary);

    &--good {
      color: var(--color-text-secondary);
    }

    &--warn {
      color: var(--color-text-secondary);
    }

    &--muted {
      color: var(--color-text-tertiary);
    }
  }

  &__snapshot-tag {
    flex-shrink: 0;
    padding: 1px 8px;
    font-size: var(--font-size-xs);
    color: #ffffff;
    background-color: var(--snapshot-color);
    border-radius: 999px;
  }

  &__muted {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__block-head {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__improve-list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
  }

  &__improve-row {
    display: flex;
    gap: var(--spacing-3);
    align-items: center;
    padding: var(--spacing-3) var(--spacing-4);
    background-color: var(--color-bg-page);
    border-radius: var(--radius-md);
  }

  &__improve-status {
    flex-shrink: 0;
    padding: 2px 10px;
    font-size: var(--font-size-xs);
    color: var(--color-info);
    background-color: rgba(19, 194, 194, 0.12);
    border-radius: 999px;

    &[data-status='done_self'] {
      color: var(--color-warning);
      background-color: rgba(250, 173, 20, 0.14);
    }
  }

  &__improve-title {
    font-size: var(--font-size-sm);
    color: var(--color-text-primary);
  }

  &__notes {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--spacing-4);

    @media (max-width: 1200px) {
      grid-template-columns: 1fr;
    }
  }

  &__note {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    padding: var(--spacing-4);
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
  }

  &__note-head {
    display: flex;
    gap: var(--spacing-2);
    align-items: baseline;
    justify-content: space-between;
    padding-bottom: var(--spacing-2);
    border-bottom: 1px dashed var(--color-divider);
  }

  &__note-course {
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__note-date {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__note-block {
    display: flex;
    gap: var(--spacing-2);
    align-items: flex-start;

    &--ai {
      padding: var(--spacing-2) var(--spacing-3);
    }
  }

  &__note-tag {
    flex-shrink: 0;
    padding: 1px 8px;
    font-size: var(--font-size-xs);
    color: var(--color-supervisor);
    background-color: var(--color-supervisor-bg);
    border-radius: 999px;

    &--ai {
      color: var(--color-ai-deep);
      background-color: transparent;
    }
  }

  &__note-text {
    font-size: var(--font-size-sm);
    line-height: 1.6;
    color: var(--color-text-secondary);
  }
}
</style>
