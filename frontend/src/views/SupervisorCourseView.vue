<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { fetchCourseDetailApi, fetchCourseEvaluationSummaryApi } from '@/api/course'
import { fetchCourseSessionsApi, uploadSessionRecordingApi } from '@/api/session'
import { useSemester } from '@/composables/useSemester'
import { getCourseStatusMeta, PAGE_SIZES } from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { Course, CourseEvaluationSummary } from '@/types/course'
import type { SessionListItem } from '@/types/evaluation'

/**
 * 督导「课程评分与授课记录」页（课程列表 → 具体课程）。
 * 三段式：
 *   1. 抬头：课程名与 AI 评价分一行；
 *   2. 授课记录：时间轨行，督导未评去评估、督导已评查看/修改；
 *   3. 录音：收进每行的文本链，转写与查看在评估页完成。
 */
const route = useRoute()
const router = useRouter()
const { semester } = useSemester()

const courseId = computed(() => String(route.params.id ?? ''))

const course = ref<Course | null>(null)
const summary = ref<CourseEvaluationSummary | null>(null)
const loading = ref(true)
const error = ref(false)

const sessions = ref<SessionListItem[]>([])
const sessionsLoading = ref(false)
const sessionsError = ref(false)
const sessionTotal = ref(0)
const sessionPage = ref(1)
const sessionPageSize = ref(10)
const uploadingSessionId = ref<number | null>(null)

const statusMeta = computed(() =>
  course.value ? getCourseStatusMeta(course.value.status) : null
)

/** 页面只展示 AI 评价分（AI 未评时回退后端综合分，缺失显示 —） */
const aiScore = computed(() => summary.value?.agentScore ?? summary.value?.compositeScore ?? null)
const aiColor = computed(() => scoreToneColor(scoreTone(aiScore.value)))

const infoItems = computed(() => {
  if (!course.value) return []
  return [
    { label: '教研室', value: course.value.department },
    { label: '授课教师', value: course.value.teacherName },
    { label: '学期', value: course.value.semester },
    { label: '开课班级', value: `${course.value.classes?.length ?? course.value.classCount} 个` },
    { label: '学生人次', value: `${course.value.studentCount} 人次` }
  ]
})

/** 数字一致性铁律：直接展示后端值，不做前端四舍五入 */
function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

function scoreColor(score: number | null | undefined): string {
  return scoreToneColor(scoreTone(score))
}

async function loadCourse(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    course.value = await fetchCourseDetailApi(courseId.value)
  } catch {
    error.value = true
    course.value = null
    loading.value = false
    return
  }
  loading.value = false
  summary.value = await fetchCourseEvaluationSummaryApi(courseId.value, {
    semester: semester.value || undefined
  }).catch(() => null)
}

async function loadSessions(): Promise<void> {
  sessionsLoading.value = true
  sessionsError.value = false
  try {
    const result = await fetchCourseSessionsApi(courseId.value, {
      semester: semester.value || undefined,
      page: sessionPage.value,
      pageSize: sessionPageSize.value
    })
    sessions.value = result.list
    sessionTotal.value = result.total
  } catch {
    sessionsError.value = true
    sessions.value = []
    sessionTotal.value = 0
  } finally {
    sessionsLoading.value = false
  }
}

function reload(): void {
  void loadCourse()
  sessionPage.value = 1
  void loadSessions()
}

function goSession(row: SessionListItem): void {
  void router.push({ name: 'session-evaluation', params: { id: row.id } })
}

/** el-table 插槽行类型为 DefaultRow，取用时显式收敛为 SessionListItem */
function asSession(row: unknown): SessionListItem {
  return row as SessionListItem
}

/**
 * 督导课程页状态徽章：以 supervisorEvaluated 判断督导是否已评，
 * status 只用于区分「AI 已评 / 已录音 / 待录音」。data-status 仍沿用原样式类。
 */
function sessionBadge(row: SessionListItem): { label: string; dataStatus: string } {
  if (row.supervisorEvaluated === true) {
    return { label: '督导已评', dataStatus: 'evaluated' }
  }
  if (row.status === 'evaluated') {
    return { label: '待督导评估', dataStatus: 'recorded' }
  }
  if (row.status === 'recorded') {
    return { label: '待 AI 评析', dataStatus: 'recorded' }
  }
  return { label: '待录音', dataStatus: 'scheduled' }
}

/** 督导已评 → 「查看/修改评估」；否则「去评估」（评估页可覆盖提交） */
function sessionActionLabel(row: SessionListItem): string {
  return row.supervisorEvaluated === true ? '查看/修改评估' : '去评估'
}

async function handleRecording(row: SessionListItem, file: File): Promise<boolean> {
  uploadingSessionId.value = row.id
  try {
    await uploadSessionRecordingApi(row.id, file)
    ElMessage.success('录音已上传，转写完成后可在评估页查看')
    await loadSessions()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    uploadingSessionId.value = null
  }
  return false
}

function handlePageChange(value: number): void {
  sessionPage.value = value
  void loadSessions()
}

function handlePageSizeChange(value: number): void {
  sessionPageSize.value = value
  sessionPage.value = 1
  void loadSessions()
}

function goBack(): void {
  router.back()
}

watch(courseId, () => reload())
watch(semester, () => reload())

onMounted(() => reload())
</script>

<template>
  <div class="supervisor-course">
    <el-skeleton v-if="loading" :rows="10" animated />

    <EmptyState v-else-if="error || !course" description="课程信息加载失败">
      <template #action>
        <el-button type="primary" @click="reload">重新加载</el-button>
      </template>
    </EmptyState>

    <template v-else>
      <PageHeader>
        <template #title>
          <div class="supervisor-course__title">
            <el-button link @click="goBack">返回</el-button>
            <span class="supervisor-course__name">{{ course.name }}</span>
            <span class="supervisor-course__code">{{ course.code }}</span>
            <span v-if="statusMeta" class="supervisor-course__status">{{ statusMeta.label }}</span>
          </div>
        </template>
        <template #actions>
          <div class="supervisor-course__ai">
            <span class="supervisor-course__ai-label">AI 评价</span>
            <strong class="supervisor-course__ai-score score-num" :style="{ color: aiColor }">
              {{ formatScore(aiScore) }}
            </strong>
            <span class="supervisor-course__ai-sample tabular-nums">
              已评 {{ summary?.sample.evaluatedCount ?? 0 }} / {{ summary?.sample.sessionCount ?? 0 }} 场
            </span>
          </div>
        </template>
      </PageHeader>

      <div class="supervisor-course__info">
        <div v-for="item in infoItems" :key="item.label" class="supervisor-course__info-item">
          <span class="supervisor-course__info-label">{{ item.label }}</span>
          <span class="supervisor-course__info-value">{{ item.value }}</span>
        </div>
        <span v-if="summary && !summary.sample.sampleSufficient" class="supervisor-course__flag">
          样本不足，分数代表性有限
        </span>
      </div>

      <!-- 授课记录：时间轨行 -->
      <section class="supervisor-course__panel">
        <div class="supervisor-course__panel-head">
          <h2 class="supervisor-course__section-title">授课记录</h2>
          <span class="supervisor-course__section-count tabular-nums">{{ sessionTotal }} 节</span>
        </div>

        <div v-if="sessionsError" class="supervisor-course__state">
          <EmptyState description="授课记录加载失败">
            <template #action>
              <el-button type="primary" @click="loadSessions">重新加载</el-button>
            </template>
          </EmptyState>
        </div>

        <template v-else>
          <el-skeleton v-if="sessionsLoading && !sessions.length" :rows="5" animated />

          <ol v-else-if="sessions.length" class="course-sessions">
            <li v-for="row in sessions" :key="row.id" class="course-sessions__row">
              <div class="course-sessions__rail">
                <span class="course-sessions__date tabular-nums">{{ row.sessionDate }}</span>
                <span class="course-sessions__period">{{ row.period || '节次待定' }}</span>
              </div>
              <div class="course-sessions__body">
                <p class="course-sessions__topic">{{ row.topic || '未填写主题' }}</p>
                <p class="course-sessions__meta">
                  <span class="course-sessions__ai" :style="{ color: scoreColor(row.agentScore) }">
                    AI {{ formatScore(row.agentScore) }}
                  </span>
                  <span>督导 {{ formatScore(row.supervisorScore) }}</span>
                  <span>已评 {{ row.evaluationCount }} 次</span>
                </p>
              </div>
              <span
                class="course-sessions__status"
                :data-status="sessionBadge(asSession(row)).dataStatus"
              >
                <i class="course-sessions__dot" aria-hidden="true" />
                {{ sessionBadge(asSession(row)).label }}
              </span>
              <button type="button" class="course-sessions__link" @click="goSession(asSession(row))">
                {{ sessionActionLabel(asSession(row)) }}
              </button>
              <el-upload
                class="course-sessions__upload"
                :show-file-list="false"
                accept=".mp3,.wav,.m4a"
                :before-upload="(file) => handleRecording(asSession(row), file)"
              >
                <button
                  type="button"
                  class="course-sessions__link course-sessions__link--quiet"
                  :disabled="uploadingSessionId === row.id"
                >
                  {{ uploadingSessionId === row.id ? '上传中' : '上传录音' }}
                </button>
              </el-upload>
            </li>
          </ol>

          <EmptyState v-else description="本课程暂无授课记录，可前往工作台新增授课记录" />

          <div v-if="sessionTotal > sessionPageSize" class="supervisor-course__pagination">
            <el-pagination
              :current-page="sessionPage"
              :page-size="sessionPageSize"
              :page-sizes="PAGE_SIZES"
              :total="sessionTotal"
              layout="total, sizes, prev, pager, next"
              background
              @current-change="handlePageChange"
              @size-change="handlePageSizeChange"
            />
          </div>
        </template>
      </section>
    </template>
  </div>
</template>

<style scoped lang="scss">
.supervisor-course {
  &__title {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--spacing-3);
  }

  &__name {
    font-size: var(--font-size-3xl);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__code {
    font-family: var(--font-score);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__status {
    font-size: var(--font-size-xs);
    color: var(--color-text-secondary);
  }

  &__ai {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2) var(--spacing-3);
    align-items: baseline;
  }

  &__ai-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__ai-score {
    font-size: var(--font-size-score-lg);
    line-height: 1;
  }

  &__ai-sample {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__info {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-3) var(--spacing-6);
    align-items: baseline;
    padding: var(--spacing-4) var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
  }

  &__info-item {
    display: flex;
    gap: var(--spacing-2);
    align-items: baseline;
  }

  &__info-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__info-value {
    font-size: var(--font-size-sm);
    font-weight: 500;
    color: var(--color-text-primary);
  }

  &__flag {
    font-size: var(--font-size-xs);
    color: var(--color-warning);
  }

  &__panel {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
  }

  &__panel-head {
    display: flex;
    gap: var(--spacing-3);
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: var(--spacing-2);
  }

  &__section-title {
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__section-count {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__state {
    padding: var(--spacing-6);
  }

  &__pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--spacing-4);
  }
}

.course-sessions {
  display: flex;
  flex-direction: column;

  &__row {
    display: grid;
    grid-template-columns: 112px minmax(0, 1fr) 76px 88px 72px;
    gap: var(--spacing-4);
    align-items: center;
    padding: var(--spacing-4) 0;
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

  &__topic {
    overflow: hidden;
    font-size: var(--font-size-base);
    font-weight: 500;
    color: var(--color-text-primary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-3);
    margin-top: 2px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__status {
    display: flex;
    gap: var(--spacing-1);
    align-items: center;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);

    &[data-status='evaluated'] {
      color: var(--color-success);
    }
  }

  &__dot {
    width: 6px;
    height: 6px;
    background-color: var(--color-text-disabled);
    border-radius: 50%;

    [data-status='evaluated'] & {
      background-color: var(--color-success);
    }

    [data-status='scheduled'] &,
    [data-status='recorded'] & {
      background-color: var(--color-warning);
    }
  }

  &__link {
    padding: 0;
    font: inherit;
    font-size: var(--font-size-sm);
    color: var(--color-primary);
    text-align: right;
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

    &--quiet {
      color: var(--color-text-tertiary);
    }

    &:disabled {
      color: var(--color-text-disabled);
      cursor: default;
      text-decoration: none;
    }
  }

  &__upload {
    display: inline-block;
  }
}

@media (max-width: 1080px) {
  .course-sessions__row {
    grid-template-columns: 100px minmax(0, 1fr) 72px 64px;

    .course-sessions__upload {
      display: none;
    }
  }
}

@media (max-width: 840px) {
  .course-sessions__row {
    grid-template-columns: 88px minmax(0, 1fr) 56px;

    .course-sessions__status {
      display: none;
    }
  }
}
</style>
