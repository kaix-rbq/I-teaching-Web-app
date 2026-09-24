<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AiBadge from '@/components/common/AiBadge.vue'
import ResourceList from '@/components/course/ResourceList.vue'
import ResourceUploader from '@/components/course/ResourceUploader.vue'
import QualityCompass from '@/components/evaluation/QualityCompass.vue'
import ScoreDimensionsCard from '@/components/evaluation/ScoreDimensionsCard.vue'
import EvaluationTimeline from '@/components/evaluation/EvaluationTimeline.vue'
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
const chatVisible = ref(false)

const resources = ref<Resource[]>([])
const resourceLoading = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)

const { messages, streaming, send, stop } = useAgentChat()

const FLAG_TEXT: Record<string, string> = {
  no_data: '暂无评价',
  sample_insufficient: '样本不足，当前分数代表性有限',
  disjoint: '督导与智能体评价尚未对齐，当前分数代表性有限',
  formula_mixed: '口径版本混杂，请谨慎解读'
}

function flagText(flag: string): string {
  return FLAG_TEXT[flag] ?? flag
}

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
      <template #subtitle>课程质量驾驶舱 · 督导评分与 AI 建议双源驱动</template>
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
      <!-- 驾驶舱层：本课程质量第一眼（罗盘 | 雷达 | 样本口径） -->
      <section class="course-improve__cockpit">
        <div class="course-improve__cockpit-card">
          <h2 class="course-improve__card-title">本课程综合分 · 双源罗盘</h2>
          <QualityCompass :summary="courseSummary" :loading="scoreLoading" size="lg" />
        </div>

        <div class="course-improve__cockpit-card">
          <ScoreDimensionsCard :summary="courseSummary" title="本课程五维评分 · 双源叠加" />
        </div>

        <div class="course-improve__cockpit-card">
          <h2 class="course-improve__card-title">样本与口径</h2>
          <template v-if="courseSummary">
            <dl class="course-improve__facts">
              <div>
                <dt>本课程已评</dt>
                <dd class="tabular-nums">
                  {{ courseSummary.sample.evaluatedCount }} / {{ courseSummary.sample.sessionCount }} 场
                </dd>
              </div>
              <div>
                <dt>评价来源</dt>
                <dd class="tabular-nums">
                  督导 {{ courseSummary.sample.supervisorCount }} · AI
                  {{ courseSummary.sample.agentCount }}
                </dd>
              </div>
              <div>
                <dt>双源对齐</dt>
                <dd class="tabular-nums">{{ courseSummary.sample.alignedCount }} 场</dd>
              </div>
              <div>
                <dt>融合权重</dt>
                <dd>
                  督导 {{ Math.round(courseSummary.weights.supervisor * 100) }}% / 智能体
                  {{ Math.round(courseSummary.weights.agent * 100) }}%
                </dd>
              </div>
              <div>
                <dt>口径版本</dt>
                <dd>{{ courseSummary.formulaVersion || '—' }}</dd>
              </div>
            </dl>
            <div v-if="courseSummary.flags.length" class="course-improve__flags">
              <el-alert
                v-for="flag in courseSummary.flags"
                :key="flag"
                :title="flagText(flag)"
                type="warning"
                :closable="false"
                show-icon
              />
            </div>
            <p v-else class="course-improve__caliber-ok">样本与口径无异常提示</p>
          </template>
          <p v-else class="course-improve__muted">本课程暂无评价</p>
        </div>
      </section>

      <!-- 明细层：左 = 我的表现 + 督导评语流；右 = 资源 + 基本信息（折叠卡） -->
      <div class="course-improve__detail">
        <div class="course-improve__detail-main">
          <section class="course-improve__block">
            <h2 class="course-improve__block-title">我的教学表现（本学期全部课程口径）</h2>
            <ScoreDimensionsCard :summary="teacherSummary" title="五维评分 · 双源叠加" />
            <p class="course-improve__muted course-improve__compare">
              进步幅度将在历史数据齐备后提供。
            </p>
          </section>

          <section class="course-improve__block">
            <h2 class="course-improve__block-title">督导评语流</h2>
            <EvaluationTimeline
              :items="courseTimeline"
              :loading="commentLoading"
              empty-text="本课程暂无督导评语"
            />
          </section>
        </div>

        <div class="course-improve__detail-side">
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

          <section class="course-improve__block course-improve__block--flush">
            <el-collapse class="course-improve__info-collapse">
              <el-collapse-item name="info">
                <template #title>
                  <span class="course-improve__block-title course-improve__block-title--inline">
                    课程基本信息
                  </span>
                </template>
                <div class="course-improve__summary">
                  <div
                    v-for="item in summaryItems"
                    :key="item.label"
                    class="course-improve__summary-item"
                  >
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
              </el-collapse-item>
            </el-collapse>
          </section>
        </div>
      </div>

      <!-- AI 层：建议卡 + 趋势折线 + 对话抽屉入口 -->
      <div class="course-improve__ai">
        <section class="course-improve__block">
          <div class="course-improve__block-head">
            <h2 class="course-improve__block-title course-improve__block-title--inline">
              <AiBadge text="AI 建议" />
              智能体提优建议
            </h2>
            <el-button
              type="primary"
              plain
              round
              size="small"
              @click="chatVisible = true"
            >
              与 AI 助手对话 →
            </el-button>
          </div>
          <AgentSuggestionList :items="suggestions" :loading="improveLoading" />
        </section>

        <section class="course-improve__block">
          <div class="course-improve__block-head">
            <h2 class="course-improve__block-title course-improve__block-title--inline">
              <AiBadge text="AI 趋势" />
              分数趋势
            </h2>
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
      </div>
    </template>

    <!-- 右下角常驻提优助手（四芒星徽标，点开 420px 抽屉流式对话） -->
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

    <el-drawer
      v-model="chatVisible"
      title="课程提优助手"
      direction="rtl"
      size="420px"
      append-to-body
    >
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

  &__cockpit {
    display: grid;
    grid-template-columns: minmax(260px, 1fr) minmax(320px, 1.5fr) minmax(260px, 1fr);
    gap: var(--spacing-4);
    margin-bottom: var(--spacing-4);

    @media (max-width: 1280px) {
      grid-template-columns: 1fr 1fr;
    }

    @media (max-width: 960px) {
      grid-template-columns: 1fr;
    }
  }

  &__cockpit-card {
    display: flex;
    flex-direction: column;
    padding: var(--spacing-5);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__card-title {
    margin-bottom: var(--spacing-3);
    padding-bottom: var(--spacing-2);
    font-size: var(--font-size-base);
    color: var(--color-text-primary);
    border-bottom: 1px solid var(--color-divider);
  }

  &__facts {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);

    dt {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    dd {
      font-size: var(--font-size-sm);
      font-weight: 500;
      color: var(--color-text-primary);
    }
  }

  &__flags {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    margin-top: var(--spacing-4);
  }

  &__caliber-ok {
    margin-top: var(--spacing-4);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__detail {
    display: grid;
    grid-template-columns: minmax(0, 1.55fr) minmax(0, 1fr);
    gap: var(--spacing-4);
    align-items: start;
    margin-bottom: var(--spacing-4);

    @media (max-width: 1280px) {
      grid-template-columns: 1fr;
    }
  }

  &__detail-main,
  &__detail-side {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
    min-width: 0;
  }

  &__ai {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
  }

  &__block {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);

    &--flush {
      padding: var(--spacing-3) var(--spacing-5);
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
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);

    &--inline {
      margin-bottom: 0;
      font-size: var(--font-size-lg);
    }
  }

  &__compare {
    margin-top: var(--spacing-3);
  }

  &__summary {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: var(--spacing-4);
    padding: var(--spacing-4) 0;

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

  &__info-collapse {
    border: none;

    :deep(.el-collapse-item__header) {
      font-size: var(--font-size-lg);
      font-weight: 600;
      color: var(--color-text-primary);
    }

    :deep(.el-collapse-item__wrap) {
      background-color: transparent;
    }

    :deep(.el-collapse-item__content) {
      padding-bottom: var(--spacing-2);
    }
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

  &__muted {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
