<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ResourceList from '@/components/course/ResourceList.vue'
import ResourceUploader from '@/components/course/ResourceUploader.vue'
import ScoreSummaryPanel from '@/components/evaluation/ScoreSummaryPanel.vue'
import ScoreTrendChart from '@/components/evaluation/ScoreTrendChart.vue'
import AgentSuggestionList from '@/components/evaluation/AgentSuggestionList.vue'
import AgentChat from '@/components/evaluation/AgentChat.vue'
import { fetchCourseDetailApi, fetchCourseEvaluationSummaryApi } from '@/api/course'
import { fetchTeacherEvaluationsApi, fetchTeacherSummaryApi } from '@/api/teacher'
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
  getCourseStatusMeta,
  MAX_RESOURCE_SIZE,
  RESOURCE_ACCEPT
} from '@/constants'
import { formatDate } from '@/utils/format'
import type { Course, CourseEvaluationSummary } from '@/types/course'
import type { Resource } from '@/types/resource'
import type { AgentSuggestion } from '@/types/agent'
import type { ScoreTrendSeries, TeacherSummary, TeacherTimelineItem } from '@/types/teacher'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const dict = useDictStore()

const courseId = computed(() => String(route.params.id ?? ''))

const course = ref<Course | null>(null)
const courseSummary = ref<CourseEvaluationSummary | null>(null)
const teacherSummary = ref<TeacherSummary | null>(null)
const timeline = ref<TeacherTimelineItem[]>([])
const suggestions = ref<AgentSuggestion[]>([])
const trend = ref<ScoreTrendSeries[]>([])

const loading = ref(true)
const error = ref(false)
const scoreLoading = ref(false)
const commentLoading = ref(false)
const improveLoading = ref(false)
const semester = ref('')
const trendDimension = ref('composite')

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
    { label: '学期', value: course.value.semester, unit: '' },
    {
      label: '班级',
      value: course.value.classes?.length ?? course.value.classCount,
      unit: '个'
    },
    { label: '学生人次', value: course.value.studentCount, unit: '人次' }
  ]
})

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

async function loadCourse(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    course.value = await fetchCourseDetailApi(courseId.value)
    semester.value = course.value.semester || CURRENT_SEMESTER
  } catch {
    error.value = true
    course.value = null
    loading.value = false
  }
}

async function loadScores(): Promise<void> {
  const params = semester.value ? { semester: semester.value } : {}
  scoreLoading.value = true
  try {
    courseSummary.value = await fetchCourseEvaluationSummaryApi(
      courseId.value,
      params
    ).catch(() => null)
    teacherSummary.value = auth.user?.id
      ? await fetchTeacherSummaryApi(auth.user.id, params).catch(() => null)
      : null
  } finally {
    scoreLoading.value = false
  }

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

async function loadPage(): Promise<void> {
  loading.value = true
  await loadCourse()
  if (error.value) return
  loading.value = false
  await Promise.all([loadResources(), loadScores(), loadImprove()])
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
      <template #subtitle>基于督导评分与课堂记录的课程改进视图</template>
      <template #actions>
        <el-select
          v-model="semester"
          placeholder="学期"
          clearable
          class="course-improve__semester"
          @change="handleSemesterChange"
        >
          <el-option
            v-for="item in dict.semesters"
            :key="item"
            :label="item"
            :value="item"
          />
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
      <!-- 区块一：课程基本信息 -->
      <section class="course-improve__block">
        <h2 class="course-improve__block-title">课程基本信息</h2>
        <div class="course-improve__summary">
          <div v-for="item in summaryItems" :key="item.label" class="course-improve__summary-item">
            <span class="course-improve__summary-value tabular-nums">
              {{ item.value }}<em v-if="item.unit">{{ item.unit }}</em>
            </span>
            <span class="course-improve__summary-label">{{ item.label }}</span>
          </div>
        </div>
        <p class="course-improve__info">
          <span>所属教研室：{{ course.department }}</span>
          <span>授课教师：{{ course.teacherName }}</span>
          <span>学期：{{ course.semester }}</span>
        </p>
      </section>

      <!-- 区块二：课程资源 -->
      <section class="course-improve__block">
        <h2 class="course-improve__block-title">课程资源</h2>
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

      <!-- 区块三：评分区 -->
      <section class="course-improve__block">
        <ScoreSummaryPanel
          title="本课程评分"
          :summary="courseSummary"
          :loading="scoreLoading"
          empty-text="本课程暂无评价"
        />
        <el-divider />
        <ScoreSummaryPanel
          title="我的教学表现"
          :summary="teacherSummary"
          :loading="scoreLoading"
          empty-text="暂无我的评价"
          show-compare-hint
        />
      </section>

      <!-- 区块四：督导评语 -->
      <section class="course-improve__block">
        <h2 class="course-improve__block-title">督导评语</h2>
        <el-skeleton v-if="commentLoading" :rows="4" animated />
        <template v-else-if="courseTimeline.length">
          <article
            v-for="item in courseTimeline"
            :key="item.sessionId"
            class="course-improve__comment-item"
          >
            <header class="course-improve__comment-head">
              <div class="course-improve__comment-when">
                <span class="course-improve__comment-date">{{ formatDate(item.sessionDate) }}</span>
                <span class="course-improve__comment-meta">
                  {{ item.period || '—' }}
                  <template v-if="item.topic"> · {{ item.topic }}</template>
                </span>
              </div>
              <span class="course-improve__comment-score tabular-nums">
                {{ item.compositeScore === null ? '—' : item.compositeScore.toFixed(2) }}
              </span>
            </header>

            <div
              v-for="evaluation in item.supervisorEvaluations"
              :key="`${evaluation.evaluatorType}-${evaluation.evaluatorId}`"
              class="course-improve__comment-body"
            >
              <div class="course-improve__comment-who">
                {{ evaluation.evaluatorName || '督导' }}
                <span class="course-improve__comment-time">{{ formatDate(evaluation.updatedAt) }}</span>
              </div>
              <dl v-if="evaluation.highlights" class="course-improve__comment-row">
                <dt>亮点</dt>
                <dd>{{ evaluation.highlights }}</dd>
              </dl>
              <dl v-if="evaluation.improvements" class="course-improve__comment-row">
                <dt>待改进</dt>
                <dd>{{ evaluation.improvements }}</dd>
              </dl>
              <dl v-if="evaluation.suggestions" class="course-improve__comment-row">
                <dt>建议</dt>
                <dd>{{ evaluation.suggestions }}</dd>
              </dl>
              <dl v-if="evaluation.comment" class="course-improve__comment-row">
                <dt>总体</dt>
                <dd>{{ evaluation.comment }}</dd>
              </dl>
            </div>

            <p
              v-if="item.supervisorEvaluations.length === 0"
              class="course-improve__muted"
            >
              本次课暂无督导评语
            </p>
          </article>
        </template>
        <p v-else class="course-improve__muted">本课程暂无督导评语</p>
      </section>

      <!-- 区块五：智能体提优建议 -->
      <section class="course-improve__block">
        <h2 class="course-improve__block-title">智能体提优建议</h2>
        <AgentSuggestionList :items="suggestions" :loading="improveLoading" />
      </section>

      <!-- 区块六：趋势折线 -->
      <section class="course-improve__block">
        <div class="course-improve__block-head">
          <h2 class="course-improve__block-title">分数趋势</h2>
          <el-radio-group v-model="trendDimension" size="small">
            <el-radio-button
              v-for="option in trendOptions"
              :key="option.key"
              :value="option.key"
            >
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

      <!-- 区块七：与智能体对话 -->
      <section class="course-improve__block">
        <h2 class="course-improve__block-title">与智能体对话</h2>
        <AgentChat
          :messages="messages"
          :streaming="streaming"
          @send="handleAsk"
          @stop="stop"
        />
      </section>
    </template>
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

  &__block {
    padding: var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);

    &:last-child {
      margin-bottom: 0;
    }
  }

  &__block-head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-3);
    margin-bottom: var(--spacing-4);
  }

  &__block-title {
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__block-head &__block-title {
    margin-bottom: 0;
  }

  &__summary {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: var(--spacing-4);

    @media (max-width: 1280px) {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  &__summary-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--spacing-1);
  }

  &__summary-value {
    font-size: var(--font-size-stat);
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

  &__summary-label {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__info {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-6);
    margin-top: var(--spacing-4);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__resource-list {
    margin-top: var(--spacing-4);
  }

  &__comment-item {
    padding: var(--spacing-4);
    margin-bottom: var(--spacing-3);
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);

    &:last-of-type {
      margin-bottom: 0;
    }
  }

  &__comment-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-4);
    margin-bottom: var(--spacing-2);
  }

  &__comment-when {
    display: flex;
    align-items: baseline;
    gap: var(--spacing-2);
  }

  &__comment-date {
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__comment-meta {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__comment-score {
    font-size: var(--font-size-xl);
    font-weight: 600;
    color: var(--color-primary);
  }

  &__comment-body {
    padding-top: var(--spacing-2);
    margin-top: var(--spacing-2);
    border-top: 1px dashed var(--color-divider);
  }

  &__comment-who {
    margin-bottom: var(--spacing-1);
    font-size: var(--font-size-sm);
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  &__comment-time {
    margin-left: var(--spacing-2);
    font-size: var(--font-size-xs);
    font-weight: 400;
    color: var(--color-text-tertiary);
  }

  &__comment-row {
    display: flex;
    gap: var(--spacing-2);
    margin-top: var(--spacing-1);
    font-size: var(--font-size-sm);

    dt {
      flex-shrink: 0;
      width: 52px;
      color: var(--color-text-tertiary);
    }

    dd {
      flex: 1;
      line-height: 1.6;
      color: var(--color-text-secondary);
    }
  }

  &__muted {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
