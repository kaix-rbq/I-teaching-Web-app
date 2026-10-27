<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AiScoreHero from '@/components/evaluation/AiScoreHero.vue'
import ScoreDimensionsCard from '@/components/evaluation/ScoreDimensionsCard.vue'
import TeacherAiReportCard from '@/components/evaluation/TeacherAiReportCard.vue'
import { fetchTeacherAiReportApi } from '@/api/agent'
import { fetchTeacherEvaluationsApi, fetchTeacherSummaryApi } from '@/api/teacher'
import { fetchCourseSessionsApi } from '@/api/session'
import { useAuthStore } from '@/stores/auth'
import { useSemester } from '@/composables/useSemester'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { TeacherSummary, TeacherTimelineItem } from '@/types/teacher'
import type { AiReportEnvelope } from '@/types/agent'
import type { EvaluationDTO, SessionListItem } from '@/types/evaluation'

/**
 * 我的质量档案：只承载 AI 评价台账。
 * 今日教学与课堂问题已迁往「教师工作台」，本页不再重复日程信息。
 * 分数一律取 AI 评价（agentScore），缺失分渲染 —，不使用 0 冒充。
 */
const router = useRouter()
const auth = useAuthStore()
const { semester } = useSemester()

const WEEK_LABELS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

const summary = ref<TeacherSummary | null>(null)
const timeline = ref<TeacherTimelineItem[]>([])
const courseSessions = ref<Array<SessionListItem & { courseId: number; courseName: string; courseCode: string }>>([])
const loading = ref(true)
const error = ref(false)

const aiReport = ref<AiReportEnvelope | null>(null)
const aiReportLoading = ref(false)
const aiReportError = ref(false)
let aiReportTimer: ReturnType<typeof setTimeout> | null = null
let aiReportSeq = 0
let aiReportDisposed = false

const latestAiEvaluation = computed<EvaluationDTO | null>(
  () => timeline.value.find((item) => item.agentEvaluation)?.agentEvaluation ?? null
)

/**
 * 授课快照 = 已评价时间线 + 尚未评价的今日/近期授课记录。
 * 工作台「下课」新建的记录没有评价，也必须在这里出现（显示 — / 暂无 AI 评价）。
 */
const snapshotRows = computed<TeacherTimelineItem[]>(() => {
  const evaluated = timeline.value.slice()
  const seen = new Set(evaluated.map((item) => item.sessionId))
  for (const session of courseSessions.value) {
    if (seen.has(session.id)) continue
    seen.add(session.id)
    evaluated.push({
      sessionId: session.id,
      sessionDate: session.sessionDate,
      period: session.period,
      topic: session.topic,
      courseId: session.courseId,
      courseCode: session.courseCode,
      courseName: session.courseName,
      status: session.status,
      compositeScore: session.agentScore ?? null,
      supervisorScore: null,
      agentScore: session.agentScore ?? null,
      supervisorEvaluations: [],
      agentEvaluation: null
    })
  }
  return evaluated.sort((left, right) => {
    if (left.sessionDate !== right.sessionDate) return left.sessionDate < right.sessionDate ? 1 : -1
    return right.sessionId - left.sessionId
  })
})

function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

/** 课程综合分取 AI 分（AI 未评时回退后端综合分） */
function courseScore(course: TeacherSummary['courses'][number]): number | null {
  return course.agentScore ?? course.compositeScore ?? null
}

function sessionScore(item: TeacherTimelineItem): number | null {
  return item.agentScore ?? item.compositeScore ?? null
}

function scoreColor(item: TeacherTimelineItem): string {
  return scoreToneColor(scoreTone(sessionScore(item)))
}

function scoreState(score: number | null | undefined): string {
  const tone = scoreTone(score)
  if (tone === null) return '暂无 AI 评价'
  if (tone === 5) return '优秀'
  if (tone === 4) return '良好'
  return '待改进'
}

function snapshotNote(item: TeacherTimelineItem): string {
  const evaluation = item.agentEvaluation
  if (!evaluation) return ''
  const tone = scoreTone(sessionScore(item))
  if (tone === 4 || tone === 5) return evaluation.highlights || ''
  return evaluation.improvements || ''
}

function weekdayText(date: string): string {
  return WEEK_LABELS[dayjs(date).day()]
}

function formatDay(date: string): string {
  return dayjs(date).format('MM-DD')
}

function goImprove(courseId: number): void {
  void router.push({ name: 'course-improve', params: { id: courseId } })
}

function goSessionRecord(sessionId: number): void {
  void router.push({ name: 'session-evaluation', params: { id: sessionId } })
}

function clearAiReportTimer(): void {
  if (aiReportTimer) {
    clearTimeout(aiReportTimer)
    aiReportTimer = null
  }
}

/**
 * AI 综合总结独立加载：报告接口失败只影响该卡片，不阻断 summary/时间线主内容。
 * 生成中每 30 秒刷新一次，重复调用会先清理旧 timer，避免叠加。
 */
async function loadAiReport(): Promise<void> {
  const id = auth.user?.id
  if (!id || aiReportDisposed) return
  const seq = ++aiReportSeq
  clearAiReportTimer()
  aiReportLoading.value = true
  aiReportError.value = false
  try {
    const data = await fetchTeacherAiReportApi(id, {
      semester: semester.value || undefined
    })
    if (seq !== aiReportSeq || aiReportDisposed) return
    aiReport.value = data
    if (data.status === 'pending' || data.status === 'running') {
      aiReportTimer = setTimeout(() => {
        void loadAiReport()
      }, 30_000)
    }
  } catch {
    if (seq !== aiReportSeq || aiReportDisposed) return
    aiReportError.value = true
    aiReport.value = null
  } finally {
    if (seq === aiReportSeq) aiReportLoading.value = false
  }
}

async function load(): Promise<void> {
  const id = auth.user?.id
  if (!id) return
  void loadAiReport()
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
    const result = await fetchTeacherEvaluationsApi(id, {
      semester: semester.value || undefined,
      pageSize: 50
    })
    timeline.value = result.list ?? []
  } catch {
    timeline.value = []
  }

  const courses = summary.value?.courses ?? []
  const results = await Promise.allSettled(
    courses.map((course) =>
      fetchCourseSessionsApi(course.courseId, {
        semester: semester.value || undefined,
        pageSize: 50
      })
    )
  )
  const sessions: Array<SessionListItem & { courseId: number; courseName: string; courseCode: string }> = []
  results.forEach((result, index) => {
    if (result.status !== 'fulfilled') return
    const course = courses[index]
    for (const item of result.value.list ?? []) {
      sessions.push({
        ...item,
        courseId: course.courseId,
        courseName: course.courseName,
        courseCode: course.courseCode
      })
    }
  })
  courseSessions.value = sessions
}

onMounted(() => {
  void load()
})

watch(semester, () => {
  void load()
})

onUnmounted(() => {
  aiReportDisposed = true
  aiReportSeq += 1
  clearAiReportTimer()
})
</script>

<template>
  <div class="my-quality">
    <PageHeader title="我的质量档案" subtitle="AI 评价台账，仅用于教学改进，不作为考核依据" />

    <p class="my-quality__ethics">
      评分仅用于教学支持与改进，不作为考核依据。请勿外传或作他用。
    </p>

    <el-skeleton v-if="loading" :rows="10" animated />

    <EmptyState v-else-if="error || !summary" description="质量档案加载失败">
      <template #action>
        <el-button @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <div v-else class="my-quality__body">
      <AiScoreHero
        :summary="summary"
        :semester="semester"
        :recent-evaluation="latestAiEvaluation"
      />

      <section class="ledger-section" aria-label="AI 综合总结">
        <TeacherAiReportCard
          :envelope="aiReport"
          :loading="aiReportLoading"
          :error="aiReportError"
          @reload="loadAiReport"
        />
      </section>

      <section class="ledger-section" aria-label="五维评分">
        <ScoreDimensionsCard :summary="summary" />
      </section>

      <section class="ledger-section" aria-label="我的课程评分">
        <header class="ledger-head">
          <h2 class="ledger-title">我的课程评分</h2>
          <span class="ledger-sub">按课程查看本学期 AI 评价</span>
        </header>
        <ul v-if="summary.courses.length" class="courses">
          <li v-for="course in summary.courses" :key="course.courseId" class="courses__row">
            <span class="courses__code">{{ course.courseCode }}</span>
            <span class="courses__name">{{ course.courseName }}</span>
            <span class="courses__sample tabular-nums">n={{ course.sample.evaluatedCount }}</span>
            <span class="courses__score score-num" :style="{ color: scoreToneColor(scoreTone(courseScore(course))) }">
              {{ formatScore(courseScore(course)) }}
            </span>
            <span class="courses__state">
              {{ courseScore(course) === null ? '暂无 AI 评价' : scoreState(courseScore(course)) }}
            </span>
            <button type="button" class="text-link" @click="goImprove(course.courseId)">去提优</button>
          </li>
        </ul>
        <p v-else class="ledger-empty">本学期暂无课程评分</p>
      </section>

      <section class="ledger-section" aria-label="授课快照">
        <header class="ledger-head">
          <h2 class="ledger-title">授课快照</h2>
          <span class="ledger-sub">按课次倒序，点击查看 AI 评价与督导评语</span>
        </header>
        <ol v-if="snapshotRows.length" class="snapshots">
          <li
            v-for="item in snapshotRows"
            :key="item.sessionId"
            class="snapshot"
            role="link"
            tabindex="0"
            :aria-label="`查看 ${item.courseName} ${item.sessionDate} 的授课记录`"
            @click="goSessionRecord(item.sessionId)"
            @keydown.enter.prevent="goSessionRecord(item.sessionId)"
            @keydown.space.prevent="goSessionRecord(item.sessionId)"
          >
            <div class="snapshot__rail">
              <span class="snapshot__date tabular-nums">{{ formatDay(item.sessionDate) }}</span>
              <span class="snapshot__weekday">{{ weekdayText(item.sessionDate) }}</span>
              <span class="snapshot__period">{{ item.period || '节次待定' }}</span>
              <i class="snapshot__node" :style="{ backgroundColor: scoreColor(item) }" aria-hidden="true" />
            </div>
            <div class="snapshot__body">
              <p class="snapshot__course">{{ item.courseName }}</p>
              <p class="snapshot__topic">{{ item.topic || '未填写主题' }}</p>
              <p v-if="snapshotNote(item)" class="snapshot__note">{{ snapshotNote(item) }}</p>
            </div>
            <div class="snapshot__score-wrap">
              <span class="snapshot__score score-num" :style="{ color: scoreColor(item) }">
                {{ formatScore(sessionScore(item)) }}
              </span>
              <span class="snapshot__state">{{ scoreState(sessionScore(item)) }}</span>
            </div>
            <span class="snapshot__link">查看记录</span>
          </li>
        </ol>
        <p v-else class="ledger-empty">本学期暂无授课记录</p>
      </section>
    </div>
  </div>
</template>

<style scoped lang="scss">
.my-quality {
  &__ethics {
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__body {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
  }
}

.ledger-section {
  padding: var(--spacing-5) var(--spacing-6);
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
}

.ledger-head {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-2) var(--spacing-3);
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: var(--spacing-4);
}

.ledger-title {
  font-size: var(--font-size-lg);
}

.ledger-sub {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.ledger-empty {
  padding: var(--spacing-4) 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-tertiary);
}

.courses {
  display: flex;
  flex-direction: column;

  &__row {
    display: grid;
    grid-template-columns: 82px minmax(0, 1fr) 72px 72px 88px auto;
    gap: var(--spacing-3);
    align-items: center;
    padding: var(--spacing-3) 0;
    border-bottom: 1px dashed var(--color-divider);

    &:last-child {
      border-bottom: none;
    }
  }

  &__code {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__name {
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__sample,
  &__state {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__score {
    font-size: var(--font-size-lg);
    text-align: right;
  }
}

.text-link {
  padding: 0;
  font: inherit;
  font-size: var(--font-size-sm);
  color: var(--color-primary);
  cursor: pointer;
  background: none;
  border: none;

  &:hover {
    text-decoration: underline;
    text-underline-offset: 3px;
  }

  &:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 3px;
    border-radius: 2px;
  }
}

.snapshots {
  display: flex;
  flex-direction: column;
}

.snapshot {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr) 96px 72px;
  gap: var(--spacing-4);
  align-items: center;
  padding: var(--spacing-3) 0;
  cursor: pointer;
  border-bottom: 1px dashed var(--color-divider);
  transition: background-color var(--duration-fast) var(--ease-out-soft);

  &:last-child {
    border-bottom: none;
  }

  &:hover,
  &:focus-visible {
    background-color: var(--color-bg-page);
    outline: none;
  }

  &:focus-visible {
    box-shadow: inset 0 0 0 2px color-mix(in srgb, var(--color-primary) 35%, transparent);
    border-radius: var(--radius-sm);
  }

  &__rail {
    display: grid;
    grid-template-columns: auto auto;
    gap: 2px var(--spacing-2);
    align-items: center;
    padding-right: var(--spacing-3);
    border-right: 1px solid var(--color-divider);
  }

  &__date {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__weekday,
  &__period {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__period {
    grid-column: 1 / -1;
  }

  &__node {
    grid-column: 2;
    grid-row: 1 / 3;
    justify-self: end;
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }

  &__course {
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__topic {
    margin-top: 2px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__note {
    margin-top: var(--spacing-2);
    font-size: var(--font-size-xs);
    line-height: 1.6;
    color: var(--color-text-secondary);
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  &__score-wrap {
    display: flex;
    flex-direction: column;
    gap: 2px;
    align-items: flex-end;
  }

  &__score {
    font-size: var(--font-size-xl);
  }

  &__state {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__link {
    font-size: var(--font-size-xs);
    color: var(--color-primary);
    text-align: right;
  }
}

@media (max-width: 1024px) {
  .courses__row {
    grid-template-columns: 72px minmax(0, 1fr) 64px 64px auto;

    .courses__state {
      display: none;
    }
  }

  .snapshot {
    grid-template-columns: 88px minmax(0, 1fr) 80px;

    &__link {
      display: none;
    }
  }
}

@media (max-width: 720px) {
  .courses__row {
    grid-template-columns: 64px minmax(0, 1fr) 60px auto;

    .courses__sample {
      display: none;
    }
  }

  .snapshot {
    grid-template-columns: 76px minmax(0, 1fr);

    &__score-wrap {
      grid-column: 2;
      flex-direction: row;
      gap: var(--spacing-2);
      align-items: baseline;
    }
  }
}
</style>
