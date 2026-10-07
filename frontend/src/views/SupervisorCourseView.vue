<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ScoreDimensionsCard from '@/components/evaluation/ScoreDimensionsCard.vue'
import { fetchCourseDetailApi, fetchCourseEvaluationSummaryApi } from '@/api/course'
import { fetchCourseSessionsApi, uploadSessionRecordingApi } from '@/api/session'
import { useSemester } from '@/composables/useSemester'
import { getCourseStatusMeta, getSessionStatusMeta, PAGE_SIZES } from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { Course, CourseEvaluationSummary } from '@/types/course'
import type { SessionListItem } from '@/types/evaluation'

/**
 * 督导「课程综合评分」页（课程列表 → 具体课程）。
 * 一块页面承载三件事：
 *   1. 本课程当前综合评分（督导 × AI 双源融合，缺失分一律 —）；
 *   2. 过去授课记录评估入口：未评价去评估、已评价可查看/修改（跳当堂课评估页，督导可覆盖提交）；
 *   3. 课堂录音上传入口：逐次授课记录直接传音频，转写与查看在评估页完成。
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

const compositeScore = computed(() => summary.value?.compositeScore ?? null)
const compositeColor = computed(() => scoreToneColor(scoreTone(compositeScore.value)))

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

/** 未评价 → 「去评估」；已评价 → 「查看/修改评估」（评估页可覆盖提交） */
function sessionActionLabel(row: SessionListItem): string {
  return row.status === 'evaluated' ? '查看/修改评估' : '去评估'
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
            <el-tag effect="plain">{{ course.code }}</el-tag>
            <el-tag v-if="statusMeta" :type="statusMeta.tag" effect="light" round>
              {{ statusMeta.label }}
            </el-tag>
          </div>
        </template>
        <template #subtitle>课程综合评分 · 督导负责评估 · 历史授课记录与课堂录音</template>
      </PageHeader>

      <div class="supervisor-course__info">
        <div v-for="item in infoItems" :key="item.label" class="supervisor-course__info-item">
          <span class="supervisor-course__info-label">{{ item.label }}</span>
          <span class="supervisor-course__info-value">{{ item.value }}</span>
        </div>
      </div>

      <!-- 当前综合评分 -->
      <section class="supervisor-course__score">
        <div class="supervisor-course__score-main">
          <h2 class="supervisor-course__section-title">本课程当前综合评分</h2>
          <p class="supervisor-course__section-sub">
            督导评分与 AI 智能体评分双源融合；缺失分显示「—」，不以 0 冒充。
          </p>
          <div class="supervisor-course__tips">
            <span>综合分</span>
            <strong class="score-num" :style="{ color: compositeColor }">
              {{ formatScore(compositeScore) }}
            </strong>
            <span class="supervisor-course__muted">
              督导 {{ formatScore(summary?.supervisorScore) }} · AI {{ formatScore(summary?.agentScore) }}
            </span>
          </div>
        </div>

        <div class="supervisor-course__score-dims">
          <ScoreDimensionsCard :summary="summary" title="五维评分 · 双源叠加" />
        </div>

        <aside class="supervisor-course__score-side">
          <div class="supervisor-course__sample">
            <span class="supervisor-course__sample-label">样本</span>
            <span class="supervisor-course__sample-value tabular-nums">
              已评 {{ summary?.sample.evaluatedCount ?? 0 }} / {{ summary?.sample.sessionCount ?? 0 }} 场
            </span>
            <span class="supervisor-course__muted">
              督导 {{ summary?.sample.supervisorCount ?? 0 }} 场 · AI {{ summary?.sample.agentCount ?? 0 }} 场
            </span>
            <span v-if="summary && !summary.sample.sampleSufficient" class="supervisor-course__flag">
              样本不足，分数代表性有限
            </span>
          </div>
        </aside>
      </section>

      <!-- 历史授课记录：评估入口 + 录音上传 -->
      <section class="supervisor-course__panel">
        <div class="supervisor-course__panel-head">
          <div>
            <h2 class="supervisor-course__section-title">
              历史授课记录（{{ sessionTotal }}）
            </h2>
            <p class="supervisor-course__section-sub">
              点击「去评估」进入当堂课评估页打分；已评价记录可查看/修改并覆盖提交；可在此上传课堂录音。
            </p>
          </div>
        </div>

        <div v-if="sessionsError" class="supervisor-course__state">
          <EmptyState description="授课记录加载失败">
            <template #action>
              <el-button type="primary" @click="loadSessions">重新加载</el-button>
            </template>
          </EmptyState>
        </div>

        <template v-else>
          <el-table
            v-if="sessionsLoading || sessions.length"
            v-loading="sessionsLoading"
            :data="sessions"
            row-key="id"
            stripe
          >
            <el-table-column label="日期" width="130">
              <template #default="{ row }">
                <span class="tabular-nums">{{ row.sessionDate }}</span>
              </template>
            </el-table-column>
            <el-table-column label="节次" width="100">
              <template #default="{ row }">{{ row.period || '—' }}</template>
            </el-table-column>
            <el-table-column label="主题" min-width="180">
              <template #default="{ row }">{{ row.topic || '—' }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="getSessionStatusMeta(row.status).tag" effect="light" round>
                  {{ getSessionStatusMeta(row.status).label }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="督导分" width="100" align="right">
              <template #default="{ row }">
                <span class="tabular-nums">{{ formatScore(row.supervisorScore) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="智能体分" width="110" align="right">
              <template #default="{ row }">
                <span class="tabular-nums supervisor-course__muted">
                  {{ formatScore(row.agentScore) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="已评次数" width="100" align="center">
              <template #default="{ row }">
                <span class="tabular-nums">{{ row.evaluationCount }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="240" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="goSession(asSession(row))">
                  {{ sessionActionLabel(asSession(row)) }}
                </el-button>
                <el-upload
                  class="supervisor-course__upload"
                  :show-file-list="false"
                  accept=".mp3,.wav,.m4a"
                  :before-upload="(file) => handleRecording(asSession(row), file)"
                >
                  <el-button
                    link
                    type="primary"
                    :loading="uploadingSessionId === row.id"
                  >
                    上传录音
                  </el-button>
                </el-upload>
              </template>
            </el-table-column>
            <template #empty>
              <span class="supervisor-course__muted">暂无授课记录</span>
            </template>
          </el-table>

          <EmptyState v-else description="本课程暂无授课记录，可前往工作台新增授课记录" />

          <div class="supervisor-course__pagination">
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
    align-items: center;
    gap: var(--spacing-3);
  }

  &__name {
    font-size: var(--font-size-3xl);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__info {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-6);
    padding: var(--spacing-3) var(--spacing-6);
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

  &__score {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr) minmax(220px, 0.8fr);
    gap: var(--spacing-4);
    padding: var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);

    @media (max-width: 1280px) {
      grid-template-columns: 1fr;
    }
  }

  &__score-main {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
  }

  &__score-dims {
    padding-left: var(--spacing-6);
    border-left: 1px solid var(--color-divider);

    @media (max-width: 1280px) {
      padding-left: 0;
      border-left: none;
    }
  }

  &__score-side {
    padding-left: var(--spacing-6);
    border-left: 1px solid var(--color-divider);

    @media (max-width: 1280px) {
      padding-left: 0;
      border-left: none;
    }
  }

  &__tips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: baseline;
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);

    strong {
      font-size: var(--font-size-score-xl);
      font-weight: 600;
    }
  }

  &__sample {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-1);
  }

  &__sample-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__sample-value {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__flag {
    margin-top: var(--spacing-2);
    font-size: var(--font-size-xs);
    color: var(--color-warning);
  }

  &__panel {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__panel-head {
    display: flex;
    gap: var(--spacing-4);
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__section-title {
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__section-sub {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-xs);
    line-height: 1.6;
    color: var(--color-text-tertiary);
  }

  &__state {
    padding: var(--spacing-6);
  }

  &__muted {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__upload {
    display: inline-block;
    margin-left: var(--spacing-3);
  }

  &__pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--spacing-4);
  }
}
</style>
