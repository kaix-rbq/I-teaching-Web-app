<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import EvaluationForm from '@/components/evaluation/EvaluationForm.vue'
import CommentPanel from '@/components/evaluation/CommentPanel.vue'
import EvaluationCompare from '@/components/evaluation/EvaluationCompare.vue'
import ScoreRadar from '@/components/evaluation/ScoreRadar.vue'
import {
  fetchSessionEvaluationApi,
  submitSupervisorEvaluationApi
} from '@/api/session'
import { fetchSessionTranscriptApi, retrySessionTranscriptApi, uploadSessionRecordingApi } from '@/api/session'
import AudioPlayer from '@/components/session/AudioPlayer.vue'
import TranscriptViewer from '@/components/evaluation/TranscriptViewer.vue'
import { useAuthStore } from '@/stores/auth'
import { EVALUATION_DIMENSIONS, getScoreLevel, getSessionStatusMeta } from '@/constants'
import { formatDate } from '@/utils/format'
import type {
  EvaluationDTO,
  EvaluationFormModel,
  SessionEvaluation,
  SupervisorEvaluationPayload
} from '@/types/evaluation'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const data = ref<SessionEvaluation | null>(null)
const loading = ref(true)
const error = ref(false)
const submitting = ref(false)
const activeEvaluation = ref<EvaluationDTO | null>(null)

const sessionId = computed(() => String(route.params.id ?? ''))
const isSupervisor = computed(() => auth.role === 'supervisor')
const readonly = computed(() => !isSupervisor.value)

function emptyForm(): EvaluationFormModel {
  return {
    objective: null,
    content: null,
    interaction: null,
    organization: null,
    frontier: null,
    comment: '',
    highlights: '',
    improvements: '',
    suggestions: ''
  }
}

function fromEvaluation(evaluation: EvaluationDTO): EvaluationFormModel {
  return {
    objective: evaluation.objective,
    content: evaluation.content,
    interaction: evaluation.interaction,
    organization: evaluation.organization,
    frontier: evaluation.frontier,
    comment: evaluation.comment ?? '',
    highlights: evaluation.highlights ?? '',
    improvements: evaluation.improvements ?? '',
    suggestions: evaluation.suggestions ?? ''
  }
}

const form = ref<EvaluationFormModel>(emptyForm())

const statusMeta = computed(() =>
  data.value ? getSessionStatusMeta(data.value.session.status) : null
)

const ownEvaluation = computed<EvaluationDTO | null>(() => {
  if (!isSupervisor.value || !data.value) return null
  return (
    data.value.supervisorScores.find((item) => item.evaluatorId === auth.user?.id) ?? null
  )
})

/** 只读角色展示所选中/最新一次评价；督导展示自己的已提交评价 */
const displayedEvaluation = computed<EvaluationDTO | null>(() =>
  isSupervisor.value ? ownEvaluation.value : activeEvaluation.value
)

const metaItems = computed(() => {
  const session = data.value?.session
  if (!session) return []
  return [
    { label: '授课教师', value: session.teacherName },
    { label: '班级', value: session.className || '未指定' },
    { label: '授课日期', value: formatDate(session.sessionDate) },
    { label: '节次', value: session.period || '—' },
    { label: '主题', value: session.topic || '—' },
    { label: '学期', value: session.semester }
  ]
})

/** 雷达图数据：督导编辑时实时反映表单，其余角色反映所选评价 */
const radarItems = computed(() =>
  EVALUATION_DIMENSIONS.map((dimension) => {
    const score = isSupervisor.value
      ? form.value[dimension.key]
      : displayedEvaluation.value?.[dimension.key] ?? null
    return {
      key: dimension.key,
      name: dimension.shortName,
      score,
      isObservation: dimension.isObservation
    }
  })
)

const dimensionRows = computed(() =>
  EVALUATION_DIMENSIONS.map((dimension) => {
    const score = isSupervisor.value
      ? form.value[dimension.key]
      : displayedEvaluation.value?.[dimension.key] ?? null
    return {
      ...dimension,
      score,
      level: score === null ? '' : getScoreLevel(score)?.level ?? '',
      ratio: score === null ? 0 : (score / 5) * 100
    }
  })
)

const displayTotal = computed(() =>
  displayedEvaluation.value?.totalScore ?? null
)

const supervisorCount = computed(() => data.value?.supervisorScores.length ?? 0)
const agentScore = computed(() => data.value?.agentScore ?? null)
const formulaVersion = computed(
  () => displayedEvaluation.value?.formulaVersion ?? data.value?.supervisorScores[0]?.formulaVersion ?? 'v1'
)
const media = ref<Awaited<ReturnType<typeof fetchSessionTranscriptApi>>>({ recording: null, transcript: null })
const mediaLoading = ref(false)
const uploadProgress = ref(0)
let pollTimer: ReturnType<typeof setTimeout> | undefined
const recordingURL = computed(() => media.value.recording ? `${import.meta.env.VITE_API_BASE_URL || '/api/v1'}${media.value.recording.streamUrl}` : '')

async function loadMedia(): Promise<void> {
  mediaLoading.value = true
  try {
    media.value = await fetchSessionTranscriptApi(sessionId.value)
  } catch {
    // 录音/转写为阶段②增强项，任何失败都不得阻断评估数据面板。
    media.value = { recording: null, transcript: null }
  } finally {
    mediaLoading.value = false
  }
  if (media.value.transcript && ['pending', 'running'].includes(media.value.transcript.status)) {
    pollTimer = setTimeout(() => { void loadMedia() }, 2500)
  }
}
async function handleRecording(file: File): Promise<boolean> {
  try { await uploadSessionRecordingApi(sessionId.value, file, (v) => { uploadProgress.value = v }); ElMessage.success('录音上传成功'); await loadMedia() } catch { /* interceptor */ }
  return false
}
async function retryTranscript(): Promise<void> { try { await retrySessionTranscriptApi(sessionId.value); await loadMedia() } catch { /* interceptor */ } }

async function load(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    const result = await fetchSessionEvaluationApi(sessionId.value)
    data.value = result
    const own = isSupervisor.value
      ? result.supervisorScores.find((item) => item.evaluatorId === auth.user?.id) ?? null
      : null
    activeEvaluation.value = own ?? latestFrom(result)
    if (own) {
      form.value = fromEvaluation(own)
    } else if (!isSupervisor.value && activeEvaluation.value) {
      form.value = fromEvaluation(activeEvaluation.value)
    } else {
      form.value = emptyForm()
    }
    await loadMedia()
  } catch {
    error.value = true
    data.value = null
  } finally {
    loading.value = false
  }
}

function latestFrom(result: SessionEvaluation): EvaluationDTO | null {
  const list = result.supervisorScores
  if (list.length === 0) return null
  return [...list].sort((a, b) => (a.updatedAt < b.updatedAt ? 1 : -1))[0]
}

function buildPayload(): SupervisorEvaluationPayload | null {
  const value = form.value
  if (
    value.objective === null ||
    value.content === null ||
    value.interaction === null ||
    value.organization === null ||
    value.frontier === null
  ) {
    return null
  }
  return {
    objective: value.objective,
    content: value.content,
    interaction: value.interaction,
    organization: value.organization,
    frontier: value.frontier,
    comment: value.comment,
    highlights: value.highlights,
    improvements: value.improvements,
    suggestions: value.suggestions
  }
}

async function handleSubmit(): Promise<void> {
  const payload = buildPayload()
  if (!payload) {
    ElMessage.error('请先完成五个维度的评分')
    return
  }
  if (ownEvaluation.value) {
    try {
      await ElMessageBox.confirm('你已提交过本次课的评分，继续将覆盖上次的评分与评语。', '覆盖确认', {
        type: 'warning',
        confirmButtonText: '覆盖提交',
        cancelButtonText: '取消'
      })
    } catch {
      return
    }
  }
  submitting.value = true
  try {
    await submitSupervisorEvaluationApi(sessionId.value, payload)
    ElMessage.success('评分已提交')
    await load()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    submitting.value = false
  }
}

function handleSelectEvaluation(evaluation: EvaluationDTO): void {
  activeEvaluation.value = evaluation
  if (!isSupervisor.value) {
    form.value = fromEvaluation(evaluation)
  }
}

function goBack(): void {
  router.back()
}

watch(sessionId, () => {
  void load()
})

onMounted(() => {
  void load()
})
onUnmounted(() => { if (pollTimer) clearTimeout(pollTimer) })
</script>

<template>
  <div class="session-evaluation">
    <el-skeleton v-if="loading" :rows="10" animated />

    <EmptyState v-else-if="error" description="评估数据加载失败">
      <template #action>
        <el-button type="primary" @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <template v-else-if="data">
      <PageHeader>
        <template #title>
          <div class="session-evaluation__title">
            <el-button link @click="goBack">返回</el-button>
            <span class="session-evaluation__name">{{ data.session.courseName }}</span>
            <el-tag effect="plain">{{ data.session.courseCode }}</el-tag>
            <el-tag v-if="statusMeta" :type="statusMeta.tag" effect="light" round>
              {{ statusMeta.label }}
            </el-tag>
          </div>
        </template>
        <template #subtitle>当堂课质量评估 · {{ isSupervisor ? '可评分' : '只读' }}</template>
      </PageHeader>

      <!-- 场次信息 -->
      <div class="session-evaluation__meta">
        <div v-for="item in metaItems" :key="item.label" class="session-evaluation__meta-item">
          <span class="session-evaluation__meta-label">{{ item.label }}</span>
          <span class="session-evaluation__meta-value">{{ item.value }}</span>
        </div>
      </div>

      <!-- 评分数据面板：直观可视 + 信息全面 -->
      <section class="session-evaluation__panel">
        <header class="session-evaluation__panel-head">
          <h2 class="session-evaluation__panel-title">评分数据面板</h2>
          <span class="session-evaluation__panel-sub">
            {{ isSupervisor ? '实时反映当前评分' : '当前展示所选评价' }}
          </span>
        </header>

        <div class="session-evaluation__panel-body">
          <div class="session-evaluation__viz">
            <ScoreRadar :items="radarItems" :max="5" />
          </div>

          <div class="session-evaluation__bars">
            <div
              v-for="row in dimensionRows"
              :key="row.key"
              class="session-evaluation__bar"
            >
              <div class="session-evaluation__bar-head">
                <span class="session-evaluation__bar-name">
                  {{ row.name }}
                  <el-tag
                    v-if="row.isObservation"
                    type="warning"
                    size="small"
                    effect="light"
                    round
                  >
                    观测项
                  </el-tag>
                  <el-tag v-else type="info" size="small" effect="plain" round>
                    {{ Math.round(row.weight * 100) }}%
                  </el-tag>
                </span>
                <span class="session-evaluation__bar-value tabular-nums">
                  {{ row.score === null ? '—' : row.score }}
                  <em v-if="row.level">{{ row.level }}</em>
                </span>
              </div>
              <div class="session-evaluation__bar-track">
                <div
                  class="session-evaluation__bar-fill"
                  :class="{ 'session-evaluation__bar-fill--observation': row.isObservation }"
                  :style="{ width: `${row.ratio}%` }"
                />
              </div>
            </div>
            <p class="session-evaluation__bar-note">
              「前沿与交叉学科」为观测项，仅作亮点展示，不计入加权总分；基础课不因无前沿内容扣分。
            </p>
          </div>

          <div class="session-evaluation__summary">
            <div class="session-evaluation__total">
              <span class="session-evaluation__total-label">
                {{ isSupervisor ? '我的督导总分' : '督导总分' }}
              </span>
              <span class="session-evaluation__total-value tabular-nums">
                {{ displayTotal === null ? (isSupervisor ? '待提交' : '暂无') : displayTotal.toFixed(2) }}
              </span>
            </div>
            <dl class="session-evaluation__facts">
              <div>
                <dt>已评督导</dt>
                <dd class="tabular-nums">{{ supervisorCount }} 人</dd>
              </div>
              <div>
                <dt>智能体参考</dt>
                <dd>
                  <el-tag v-if="agentScore" type="info" effect="light" round>
                    {{ agentScore.totalScore?.toFixed(2) ?? '已接入' }}
                  </el-tag>
                  <span v-else class="session-evaluation__muted">阶段②接入</span>
                </dd>
              </div>
              <div>
                <dt>口径版本</dt>
                <dd>{{ formulaVersion }}</dd>
              </div>
            </dl>
          </div>
        </div>
      </section>

      <div class="session-evaluation__grid">
        <div class="session-evaluation__main">
          <section class="session-evaluation__card">
            <h2 class="session-evaluation__card-title">督导评分</h2>
            <EvaluationForm
              v-model="form"
              :readonly="readonly"
              :submitting="submitting"
              @submit="handleSubmit"
            />
          </section>
        </div>

        <aside class="session-evaluation__side">
          <section class="session-evaluation__card">
            <h2 class="session-evaluation__card-title">督导评语</h2>
            <CommentPanel
              :items="data.supervisorScores"
              :loading="false"
              @select="handleSelectEvaluation"
            />
          </section>

          <section class="session-evaluation__card">
            <h2 class="session-evaluation__card-title">智能体参考</h2>
            <EvaluationCompare :supervisor="displayedEvaluation" :agent="agentScore" />
          </section>

          <section class="session-evaluation__card">
            <h2 class="session-evaluation__card-title">课堂录音与转写</h2>
            <template v-if="isSupervisor">
              <el-upload :show-file-list="false" accept=".mp3,.wav,.m4a" :before-upload="handleRecording">
                <el-button type="primary" plain>上传课堂录音</el-button>
              </el-upload>
              <el-progress v-if="uploadProgress > 0 && uploadProgress < 100" :percentage="uploadProgress" />
              <AudioPlayer v-if="media.recording" :src="recordingURL" :name="media.recording.originalName" />
            </template>
            <TranscriptViewer :transcript="media.transcript" :loading="mediaLoading" @retry="retryTranscript" />
          </section>
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.session-evaluation {
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

  &__meta {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: var(--spacing-4);
    padding: var(--spacing-4) var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);

    @media (max-width: 1280px) {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  &__meta-item {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-1);
  }

  &__meta-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__meta-value {
    font-size: var(--font-size-base);
    font-weight: 500;
    color: var(--color-text-primary);
  }

  &__panel {
    padding: var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__panel-head {
    display: flex;
    align-items: baseline;
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

  &__panel-body {
    display: grid;
    grid-template-columns: 320px 1fr 240px;
    gap: var(--spacing-6);
    align-items: center;

    @media (max-width: 1280px) {
      grid-template-columns: 1fr;
    }
  }

  &__viz {
    min-width: 0;
  }

  &__bar {
    margin-bottom: var(--spacing-4);

    &:last-of-type {
      margin-bottom: var(--spacing-2);
    }
  }

  &__bar-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-1);
  }

  &__bar-name {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__bar-value {
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);

    em {
      margin-left: var(--spacing-1);
      font-size: var(--font-size-xs);
      font-style: normal;
      font-weight: 400;
      color: var(--color-text-tertiary);
    }
  }

  &__bar-track {
    height: 8px;
    overflow: hidden;
    background-color: var(--color-divider);
    border-radius: var(--radius-sm);
  }

  &__bar-fill {
    height: 100%;
    background-color: var(--color-primary);
    border-radius: var(--radius-sm);
    transition: width 0.3s ease;

    &--observation {
      background-color: var(--color-warning);
    }
  }

  &__bar-note {
    margin-top: var(--spacing-2);
    font-size: var(--font-size-xs);
    line-height: 1.6;
    color: var(--color-text-tertiary);
  }

  &__summary {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
    padding-left: var(--spacing-6);
    border-left: 1px solid var(--color-divider);

    @media (max-width: 1280px) {
      padding-left: 0;
      border-left: none;
    }
  }

  &__total {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-1);
  }

  &__total-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__total-value {
    font-size: var(--font-size-stat);
    font-weight: 600;
    line-height: 1.1;
    color: var(--color-primary);
  }

  &__facts {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);

    > div {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: var(--spacing-2);
      font-size: var(--font-size-sm);
    }

    dt {
      color: var(--color-text-tertiary);
    }

    dd {
      font-weight: 500;
      color: var(--color-text-primary);
    }
  }

  &__muted {
    font-size: var(--font-size-xs);
    font-weight: 400;
    color: var(--color-text-tertiary);
  }

  &__grid {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: var(--spacing-6);
    align-items: start;

    @media (max-width: 1280px) {
      grid-template-columns: 1fr;
    }
  }

  &__side {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
  }

  &__card {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__card-title {
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__placeholder {
    font-size: var(--font-size-sm);
    line-height: 1.6;
    color: var(--color-text-tertiary);
  }
}
</style>
