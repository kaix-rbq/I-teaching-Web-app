<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AiBadge from '@/components/common/AiBadge.vue'
import ResourceList from '@/components/course/ResourceList.vue'
import ResourceUploader from '@/components/course/ResourceUploader.vue'
import ScoreRadar from '@/components/evaluation/ScoreRadar.vue'
import EvaluationTimeline from '@/components/evaluation/EvaluationTimeline.vue'
import ScoreTrendChart from '@/components/evaluation/ScoreTrendChart.vue'
import AgentSuggestionList from '@/components/evaluation/AgentSuggestionList.vue'
import AgentChat from '@/components/evaluation/AgentChat.vue'
import { fetchCourseDetailApi, fetchCourseEvaluationSummaryApi } from '@/api/course'
import { fetchTeacherEvaluationsApi } from '@/api/teacher'
import { fetchAgentSuggestionsApi, fetchScoreTrendApi } from '@/api/agent'
import {
  deleteResourceApi,
  downloadResourceApi,
  fetchResourcesApi,
  uploadResourceApi
} from '@/api/resource'
import { useAgentChat } from '@/composables/useAgentChat'
import { useAuthStore } from '@/stores/auth'
import { useDictStore } from '@/stores/dict'
import {
  CURRENT_SEMESTER,
  EVALUATION_DIMENSIONS,
  getCourseStatusMeta,
  MAX_RESOURCE_SIZE,
  RESOURCE_ACCEPT
} from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { Course, CourseEvaluationSummary } from '@/types/course'
import type { Resource } from '@/types/resource'
import type { AgentSuggestion } from '@/types/agent'
import type { ScoreTrendSeries, TeacherTimelineItem } from '@/types/teacher'

/**
 * 教学提优页（教师）。
 * 排版原则（避免与「我的质量档案」重复）：
 *   1. 课程基本信息置顶；
 *   2. 「本课程综合评分」：五维双源横向柱条（增长动画）+ 综合分雷达；
 *   3. 督导评语流 / 智能体提优建议 / 分数趋势自上而下；
 *   4. 资源管理入口置于页面最下方。
 */
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const dict = useDictStore()

const courseId = computed(() => String(route.params.id ?? ''))

const course = ref<Course | null>(null)
const courseSummary = ref<CourseEvaluationSummary | null>(null)
const timeline = ref<TeacherTimelineItem[]>([])
const suggestions = ref<AgentSuggestion[]>([])
const trend = ref<ScoreTrendSeries[]>([])

const loading = ref(true)
const error = ref(false)
const commentLoading = ref(false)
const improveLoading = ref(false)
const semester = ref('')
const trendDimension = ref('composite')
const chatVisible = ref(false)
const barsReady = ref(false)

const resources = ref<Resource[]>([])
const resourceLoading = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)

const { messages, streaming, send, stop } = useAgentChat()

const canManageResources = computed(
  () => auth.role === 'teacher' && course.value?.teacherId === auth.user?.id
)

const statusMeta = computed(() =>
  course.value ? getCourseStatusMeta(course.value.status) : null
)

const summaryItems = computed(() => {
  if (!course.value) return []
  return [
    { label: '学分', value: course.value.credit, unit: '分' },
    { label: '学时', value: course.value.hours, unit: '学时' },
    {
      label: '班级',
      value: course.value.classes?.length ?? course.value.classCount,
      unit: '个'
    },
    { label: '学生人次', value: course.value.studentCount, unit: '人次' }
  ]
})

const compositeScore = computed(() => courseSummary.value?.compositeScore ?? null)
const compositeColor = computed(() => scoreToneColor(scoreTone(compositeScore.value)))

/** 五维双源柱条：每维督导 / AI 两条，宽度为 0-100 分。 */
const dimensionRows = computed(() =>
  EVALUATION_DIMENSIONS.map((meta) => {
    const dim = courseSummary.value?.dimensions.find((item) => item.key === meta.key)
    return {
      ...meta,
      supervisorScore: dim?.supervisorScore ?? null,
      agentScore: dim?.agentScore ?? null,
      score: dim?.score ?? null,
      supervisorRatio: barsReady.value ? Math.max(0, Math.min(100, dim?.supervisorScore ?? 0)) : 0,
      agentRatio: barsReady.value ? Math.max(0, Math.min(100, dim?.agentScore ?? 0)) : 0
    }
  })
)

const courseRadarItems = computed(() =>
  (courseSummary.value?.dimensions ?? []).map((dim) => ({
    key: dim.key,
    name: EVALUATION_DIMENSIONS.find((item) => item.key === dim.key)?.shortName ?? dim.name,
    score: dim.score,
    supervisorScore: dim.supervisorScore,
    agentScore: dim.agentScore,
    isObservation: dim.isObservation
  }))
)

/** 本课程的历次评价（按课次倒序），用于督导评语区块 */
const courseTimeline = computed(() =>
  timeline.value.filter((item) => item.courseId === Number(courseId.value))
)

const trendOptions = computed(() =>
  trend.value.map((series) => ({
    key: series.key,
    name: series.key === 'composite' ? '综合分' : series.name,
    isObservation: series.isObservation
  }))
)

const activeTrend = computed(
  () => trend.value.find((item) => item.key === trendDimension.value) ?? trend.value[0] ?? null
)

const activeTrendPoints = computed(() => activeTrend.value?.points ?? [])

function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

async function loadPage(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    course.value = await fetchCourseDetailApi(courseId.value)
    semester.value = course.value.semester || CURRENT_SEMESTER
  } catch {
    error.value = true
    course.value = null
    loading.value = false
    return
  }
  loading.value = false
  await Promise.all([loadResources(), loadScores(), loadImprove()])
  await nextTick()
  barsReady.value = true
}

async function loadScores(): Promise<void> {
  const params = semester.value ? { semester: semester.value } : {}
  courseSummary.value = await fetchCourseEvaluationSummaryApi(courseId.value, params).catch(() => null)

  commentLoading.value = true
  try {
    const result = await fetchTeacherEvaluationsApi(auth.user?.id ?? 0, {
      semester: semester.value || undefined,
      pageSize: 50
    })
    timeline.value = result.list
  } catch {
    timeline.value = []
  } finally {
    commentLoading.value = false
  }
}

async function loadImprove(): Promise<void> {
  improveLoading.value = true
  try {
    const [suggestionResult, trendResult] = await Promise.all([
      fetchAgentSuggestionsApi(courseId.value).catch(() => [] as AgentSuggestion[]),
      fetchScoreTrendApi({
        teacherId: auth.user?.id,
        courseId: courseId.value,
        semester: semester.value || undefined
      }).catch(() => [] as ScoreTrendSeries[])
    ])
    suggestions.value = suggestionResult
    trend.value = trendResult
    if (!trendResult.some((series) => series.key === trendDimension.value)) {
      trendDimension.value = trendResult[0]?.key ?? 'composite'
    }
  } finally {
    improveLoading.value = false
  }
}

function handleSemesterChange(): void {
  void loadScores()
  void loadImprove()
}

async function loadResources(): Promise<void> {
  resourceLoading.value = true
  try {
    resources.value = await fetchResourcesApi(courseId.value)
  } catch {
    resources.value = []
  } finally {
    resourceLoading.value = false
  }
}

async function handleUpload(file: File): Promise<void> {
  uploading.value = true
  uploadProgress.value = 0
  try {
    await uploadResourceApi(courseId.value, file, (percent) => {
      uploadProgress.value = percent
    })
    ElMessage.success('资源上传成功')
    await loadResources()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    uploading.value = false
    uploadProgress.value = 0
  }
}

async function handleDelete(resource: Resource): Promise<void> {
  try {
    await ElMessageBox.confirm(`确定删除资源「${resource.name}」吗？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  await deleteResourceApi(resource.id)
  ElMessage.success('资源已删除')
  await loadResources()
}

async function handleDownload(resource: Resource): Promise<void> {
  try {
    const blob = await downloadResourceApi(resource.id)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = resource.name
    link.click()
    URL.revokeObjectURL(url)
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  }
}

function handleAsk(question: string): void {
  send({ courseId: Number(courseId.value), question }, question)
}

function goBack(): void {
  router.back()
}

onMounted(async () => {
  await dict.load()
  await loadPage()
})
</script>

<template>
  <div class="course-improve">
    <PageHeader>
      <template #title>
        <div class="course-improve__title">
          <el-button link @click="goBack">返回</el-button>
          <span class="course-improve__name">{{ course?.name || '教学提优' }}</span>
          <el-tag v-if="course" effect="plain">{{ course.code }}</el-tag>
          <el-tag v-if="statusMeta" :type="statusMeta.tag" effect="light" round>
            {{ statusMeta.label }}
          </el-tag>
        </div>
      </template>
      <template #subtitle>本课程提优 · 督导评分与 AI 建议双源驱动</template>
      <template #actions>
        <el-select
          v-model="semester"
          placeholder="学期"
          clearable
          class="course-improve__semester"
          @change="handleSemesterChange"
        >
          <el-option v-for="item in dict.semesters" :key="item" :label="item" :value="item" />
        </el-select>
      </template>
    </PageHeader>

    <el-alert
      class="course-improve__ethics"
      type="info"
      :closable="false"
      show-icon
      title="仅用于教学支持，不作为考核依据"
      description="本页评分与建议用于帮助您改进教学，不用于绩效与人事评价。请勿外传或作他用。"
    />

    <el-skeleton v-if="loading" :rows="10" animated />

    <EmptyState v-else-if="error || !course" description="课程信息加载失败">
      <template #action>
        <el-button type="primary" @click="loadPage">重新加载</el-button>
      </template>
    </EmptyState>

    <template v-else>
      <!-- 1. 课程基本信息置顶 -->
      <section class="course-improve__info">
        <div class="course-improve__info-main">
          <div class="course-improve__info-head">
            <h2 class="course-improve__info-name">{{ course.name }}</h2>
            <el-tag effect="plain">{{ course.code }}</el-tag>
          </div>
          <p class="course-improve__info-desc">{{ course.description || '暂无课程简介' }}</p>
          <p class="course-improve__info-meta">
            <span>所属教研室：{{ course.department }}</span>
            <span>授课教师：{{ course.teacherName }}</span>
            <span>学期：{{ course.semester }}</span>
          </p>
        </div>
        <div class="course-improve__info-stats">
          <div v-for="item in summaryItems" :key="item.label" class="course-improve__info-stat">
            <span class="course-improve__info-value tabular-nums">
              {{ item.value }}<em v-if="item.unit">{{ item.unit }}</em>
            </span>
            <span class="course-improve__info-label">{{ item.label }}</span>
          </div>
        </div>
      </section>

      <!-- 2. 本课程综合评分：五维双源柱条（增长动画） + 综合分雷达 -->
      <section class="course-improve__score">
        <div class="course-improve__score-main">
          <div class="course-improve__score-head">
            <h2 class="course-improve__section-title">您在当前课程获得的综合评分</h2>
            <div class="course-improve__legend">
              <span class="course-improve__legend-item">
                <i class="course-improve__swatch course-improve__swatch--sup" />督导评价
              </span>
              <span class="course-improve__legend-item">
                <i class="course-improve__swatch course-improve__swatch--ai" />AI 智能体评价
              </span>
            </div>
          </div>

          <ul v-if="dimensionRows.length" class="course-improve__dims">
            <li v-for="row in dimensionRows" :key="row.key" class="course-improve__dim">
              <div class="course-improve__dim-head">
                <span class="course-improve__dim-name">
                  {{ row.name }}
                  <el-tag v-if="row.isObservation" type="warning" size="small" effect="light" round>
                    观测项
                  </el-tag>
                  <el-tag v-else type="info" size="small" effect="plain" round>
                    {{ Math.round(row.weight * 100) }}%
                  </el-tag>
                </span>
                <span class="course-improve__dim-score tabular-nums">
                  {{ formatScore(row.score) }}
                </span>
              </div>

              <div class="course-improve__bar-row">
                <span class="course-improve__bar-label">督导</span>
                <div class="course-improve__bar-track">
                  <i
                    class="course-improve__bar-fill course-improve__bar-fill--sup"
                    :style="{ width: `${row.supervisorRatio}%` }"
                  />
                </div>
                <span class="course-improve__bar-value tabular-nums">
                  {{ formatScore(row.supervisorScore) }}
                </span>
              </div>

              <div class="course-improve__bar-row">
                <span class="course-improve__bar-label">AI</span>
                <div class="course-improve__bar-track">
                  <i
                    class="course-improve__bar-fill course-improve__bar-fill--ai"
                    :style="{ width: `${row.agentRatio}%` }"
                  />
                </div>
                <span class="course-improve__bar-value tabular-nums">
                  {{ formatScore(row.agentScore) }}
                </span>
              </div>
            </li>
          </ul>
          <EmptyState v-else description="本课程暂无评价" />
        </div>

        <aside class="course-improve__score-side">
          <div class="course-improve__composite">
            <span class="course-improve__composite-label">综合分</span>
            <span class="course-improve__composite-value score-num" :style="{ color: compositeColor }">
              {{ formatScore(compositeScore) }}
            </span>
            <span class="course-improve__composite-sample">
              已评 {{ courseSummary?.sample.evaluatedCount ?? 0 }} / {{ courseSummary?.sample.sessionCount ?? 0 }} 场
            </span>
          </div>
          <ScoreRadar :items="courseRadarItems" :max="100" />
        </aside>
      </section>

      <!-- 3. 督导评语流 -->
      <section class="course-improve__block">
        <h2 class="course-improve__section-title">督导评语流</h2>
        <EvaluationTimeline
          :items="courseTimeline"
          :loading="commentLoading"
          empty-text="本课程暂无督导评语"
        />
      </section>

      <!-- 4. 智能体提优建议 -->
      <section class="course-improve__block">
        <div class="course-improve__block-head">
          <h2 class="course-improve__section-title course-improve__section-title--inline">
            <AiBadge text="AI 建议" />
            智能体提优建议
          </h2>
          <el-button type="primary" plain round size="small" @click="chatVisible = true">
            与 AI 助手对话 →
          </el-button>
        </div>
        <AgentSuggestionList :items="suggestions" :loading="improveLoading" />
      </section>

      <!-- 5. 分数趋势 -->
      <section class="course-improve__block">
        <div class="course-improve__block-head">
          <h2 class="course-improve__section-title course-improve__section-title--inline">
            <AiBadge text="AI 趋势" />
            分数趋势
          </h2>
          <el-radio-group v-model="trendDimension" size="small">
            <el-radio-button v-for="option in trendOptions" :key="option.key" :value="option.key">
              {{ option.name }}
            </el-radio-button>
          </el-radio-group>
        </div>
        <ScoreTrendChart
          :points="activeTrendPoints"
          :dimension-name="activeTrend?.name"
          :loading="improveLoading"
        />
      </section>

      <!-- 6. 资源管理入口置底 -->
      <section class="course-improve__block">
        <h2 class="course-improve__section-title">课程资源</h2>
        <ResourceUploader
          v-if="canManageResources"
          :accept="RESOURCE_ACCEPT"
          :max-size="MAX_RESOURCE_SIZE"
          :uploading="uploading"
          :progress="uploadProgress"
          @file="handleUpload"
        />
        <div class="course-improve__resource-list">
          <ResourceList
            :resources="resources"
            :can-manage="canManageResources"
            :loading="resourceLoading"
            @download="handleDownload"
            @delete="handleDelete"
          />
        </div>
      </section>
    </template>

    <!-- 右下角常驻提优助手 -->
    <button
      v-if="course"
      type="button"
      class="course-improve__fab"
      aria-label="打开提优助手对话"
      @click="chatVisible = true"
    >
      <svg class="course-improve__fab-star" viewBox="0 0 24 24" aria-hidden="true">
        <path
          d="M12 2l2.4 7.6L22 12l-7.6 2.4L12 22l-2.4-7.6L2 12l7.6-2.4L12 2z"
          fill="currentColor"
        />
      </svg>
      <span>提优助手</span>
    </button>

    <el-drawer v-model="chatVisible" title="课程提优助手" direction="rtl" size="420px" append-to-body>
      <template #header>
        <div class="course-improve__drawer-head">
          <AiBadge text="AI 助手" />
          <span class="course-improve__drawer-title">课程提优助手</span>
        </div>
      </template>
      <AgentChat :messages="messages" :streaming="streaming" @send="handleAsk" @stop="stop" />
    </el-drawer>
  </div>
</template>

<style scoped lang="scss">
.course-improve {
  &__title {
    display: flex;
    align-items: center;
    gap: var(--spacing-3);
  }

  &__name {
    font-size: var(--font-size-3xl);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__semester {
    width: 180px;
  }

  &__ethics {
    margin-bottom: var(--spacing-4);
  }

  /* 课程基本信息置顶 */
  &__info {
    display: grid;
    grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr);
    gap: var(--spacing-6);
    padding: var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);

    @media (max-width: 1024px) {
      grid-template-columns: 1fr;
    }
  }

  &__info-head {
    display: flex;
    gap: var(--spacing-3);
    align-items: center;
  }

  &__info-name {
    font-size: var(--font-size-2xl);
    color: var(--color-text-primary);
  }

  &__info-desc {
    margin-top: var(--spacing-2);
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-secondary);
  }

  &__info-meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-5);
    margin-top: var(--spacing-3);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__info-stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--spacing-3);
    align-content: center;
    padding-left: var(--spacing-6);
    border-left: 1px solid var(--color-divider);

    @media (max-width: 1024px) {
      padding-left: 0;
      border-left: none;
    }
  }

  &__info-stat {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-1);
    text-align: center;
  }

  &__info-value {
    font-size: var(--font-size-xl);
    font-weight: 600;
    color: var(--color-text-primary);

    em {
      margin-left: 2px;
      font-size: var(--font-size-sm);
      font-style: normal;
      font-weight: 400;
      color: var(--color-text-tertiary);
    }
  }

  &__info-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  /* 综合评分区 */
  &__score {
    display: grid;
    grid-template-columns: minmax(0, 1.8fr) minmax(240px, 1fr);
    gap: var(--spacing-4);
    margin-bottom: var(--spacing-4);

    @media (max-width: 1024px) {
      grid-template-columns: 1fr;
    }
  }

  &__score-main,
  &__score-side {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__score-head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-3);
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-5);
  }

  &__section-title {
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);

    &--inline {
      display: flex;
      gap: var(--spacing-2);
      align-items: center;
    }
  }

  &__legend {
    display: flex;
    gap: var(--spacing-4);
    font-size: var(--font-size-xs);
    color: var(--color-text-secondary);
  }

  &__legend-item {
    display: inline-flex;
    gap: 6px;
    align-items: center;
  }

  &__swatch {
    display: inline-block;
    width: 18px;
    height: 8px;
    border-radius: 999px;

    &--sup {
      background-color: var(--color-primary);
    }

    &--ai {
      background-color: var(--color-ai-bright);
    }
  }

  &__dims {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
    padding: 0;
    margin: 0;
    list-style: none;
  }

  &__dim {
    padding-bottom: var(--spacing-3);
    border-bottom: 1px dashed var(--color-divider);

    &:last-child {
      padding-bottom: 0;
      border-bottom: none;
    }
  }

  &__dim-head {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-2);
  }

  &__dim-name {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__dim-score {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__bar-row {
    display: grid;
    grid-template-columns: 40px 1fr 48px;
    gap: var(--spacing-2);
    align-items: center;
    margin-bottom: var(--spacing-1);
  }

  &__bar-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__bar-track {
    height: 10px;
    overflow: hidden;
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: 999px;
  }

  &__bar-fill {
    display: block;
    height: 100%;
    border-radius: 999px;
    transition: width 0.9s var(--ease-out-soft, ease);

    &--sup {
      background-color: var(--color-primary);
    }

    &--ai {
      background-color: var(--color-ai-bright);
    }
  }

  &__bar-value {
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--color-text-secondary);
    text-align: right;
  }

  &__score-side {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }

  &__composite {
    display: flex;
    flex-direction: column;
    align-items: center;
    margin-bottom: var(--spacing-2);
  }

  &__composite-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__composite-value {
    font-size: var(--font-size-score-xl);
    font-weight: 600;
    line-height: 1.1;
  }

  &__composite-sample {
    margin-top: 2px;
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

  &__block-head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-3);
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__resource-list {
    margin-top: var(--spacing-4);
  }

  &__fab {
    position: fixed;
    right: 24px;
    bottom: 24px;
    z-index: 1900;
    display: inline-flex;
    gap: 8px;
    align-items: center;
    padding: 12px 20px;
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: #fff;
    cursor: pointer;
    background: var(--gradient-ai);
    border: none;
    border-radius: 999px;
    box-shadow: 0 6px 20px rgba(109, 40, 217, 0.35);
    transition: transform var(--duration-fast) var(--ease-out-soft),
      box-shadow var(--duration-fast) var(--ease-out-soft);

    &:hover,
    &:focus-visible {
      transform: translateY(-2px);
      box-shadow: 0 10px 26px rgba(109, 40, 217, 0.45);
    }
  }

  &__fab-star {
    width: 16px;
    height: 16px;
  }

  &__drawer-head {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
  }

  &__drawer-title {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }
}
</style>
