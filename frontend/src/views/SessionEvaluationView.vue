<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import EvaluationForm from '@/components/evaluation/EvaluationForm.vue'
import EvaluationCompare from '@/components/evaluation/EvaluationCompare.vue'
import ScoreRadar from '@/components/evaluation/ScoreRadar.vue'
import TranscriptViewer from '@/components/evaluation/TranscriptViewer.vue'
import AiBadge from '@/components/common/AiBadge.vue'
import AudioPlayer from '@/components/session/AudioPlayer.vue'
import {
  fetchSessionEvaluationApi,
  fetchSessionTranscriptApi,
  retrySessionTranscriptApi,
  submitSupervisorEvaluationApi,
  uploadSessionRecordingApi
} from '@/api/session'
import type { SessionMedia } from '@/api/session'
import { useAuthStore } from '@/stores/auth'
import { EVALUATION_DIMENSIONS, getScoreLevel, getSessionStatusMeta } from '@/constants'
import { formatDate, scoreTone, scoreToneColor } from '@/utils/format'
import type {
  DimensionKey,
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
const activeSupervisorId = ref<number | null>(null)

const sessionId = computed(() => String(route.params.id ?? ''))
const isSupervisor = computed(() => auth.role === 'supervisor')
const canUpload = computed(() => auth.role === 'supervisor' || auth.role === 'teacher')

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

/** 多督导时按更新时间取最近一条；用于默认对照，不改变逐人只读展示。 */
function latestSupervisor(list: EvaluationDTO[]): EvaluationDTO | null {
  if (!list.length) return null
  return [...list].sort((a, b) => {
    if (a.updatedAt !== b.updatedAt) return a.updatedAt < b.updatedAt ? 1 : -1
    return b.evaluatorId - a.evaluatorId
  })[0]
}

const selectedSupervisor = computed<EvaluationDTO | null>(() => {
  if (!data.value) return null
  if (isSupervisor.value) return ownEvaluation.value
  const found = data.value.supervisorScores.find(
    (item) => item.evaluatorId === activeSupervisorId.value
  )
  return found ?? latestSupervisor(data.value.supervisorScores)
})

const otherSupervisors = computed(() =>
  (data.value?.supervisorScores ?? []).filter(
    (item) => !(isSupervisor.value && item.evaluatorId === auth.user?.id)
  )
)

const agentScore = computed(() => data.value?.agentScore ?? null)
const agentTotalText = computed(() => {
  const total = agentScore.value?.totalScore
  return total === null || total === undefined ? '—' : total.toFixed(2)
})

const supervisorCount = computed(() => data.value?.supervisorScores.length ?? 0)

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

/** 雷达双轨：AI 主序列 + 当前督导（督导编辑时实时反映表单）。 */
const radarItems = computed(() =>
  EVALUATION_DIMENSIONS.map((dimension) => {
    const supervisorValue = isSupervisor.value
      ? form.value[dimension.key]
      : selectedSupervisor.value?.[dimension.key] ?? null
    return {
      key: dimension.key,
      name: dimension.shortName,
      score: agentScore.value?.[dimension.key] ?? null,
      supervisorScore: supervisorValue,
      isObservation: dimension.isObservation
    }
  })
)

const dimensionRows = computed(() =>
  EVALUATION_DIMENSIONS.map((dimension) => {
    const aiScore = agentScore.value?.[dimension.key] ?? null
    const supervisorScore = isSupervisor.value
      ? form.value[dimension.key]
      : selectedSupervisor.value?.[dimension.key] ?? null
    return {
      ...dimension,
      aiScore,
      supervisorScore,
      aiRatio: aiScore === null ? 0 : (aiScore / 5) * 100,
      supervisorRatio: supervisorScore === null ? 0 : (supervisorScore / 5) * 100,
      supervisorLevel:
        supervisorScore === null ? '' : getScoreLevel(supervisorScore)?.level ?? ''
    }
  })
)

const myTotal = computed(() => ownEvaluation.value?.totalScore ?? null)
const myTotalText = computed(() => {
  if (!ownEvaluation.value) return '待提交'
  if (myTotal.value === null) return '仅评语，无总分'
  return myTotal.value.toFixed(2)
})
const myTotalColor = computed(() =>
  myTotal.value === null ? undefined : scoreToneColor(scoreTone(myTotal.value))
)

const formulaVersion = computed(
  () =>
    ownEvaluation.value?.formulaVersion ??
    agentScore.value?.formulaVersion ??
    'v1'
)

const agentTextItems = computed(() => {
  const agent = agentScore.value
  if (!agent) return []
  return [
    { label: '总体评语', value: agent.comment },
    { label: '亮点', value: agent.highlights },
    { label: '待改进', value: agent.improvements },
    { label: '改进建议', value: agent.suggestions }
  ].filter((item) => Boolean(item.value && item.value.trim()))
})

const aiStandardDims = computed(() =>
  EVALUATION_DIMENSIONS.map((dimension) => ({
    key: dimension.key,
    name: dimension.name,
    weight: dimension.weight,
    isObservation: dimension.isObservation
  }))
)

function hasAnyDimension(evaluation: EvaluationDTO): boolean {
  return EVALUATION_DIMENSIONS.some((dimension) => evaluation[dimension.key] !== null)
}

function supervisorTotalText(evaluation: EvaluationDTO): string {
  if (!hasAnyDimension(evaluation)) return '仅评语，无评分'
  if (evaluation.totalScore === null) return '仅评语，无总分'
  return `${evaluation.totalScore.toFixed(2)} / 100`
}

function supervisorTotalTone(evaluation: EvaluationDTO): string | undefined {
  if (!hasAnyDimension(evaluation) || evaluation.totalScore === null) return undefined
  return scoreToneColor(scoreTone(evaluation.totalScore))
}

function supervisorTextBlocks(evaluation: EvaluationDTO): Array<{ label: string; value: string }> {
  return [
    { label: '总体评语', value: evaluation.comment },
    { label: '亮点', value: evaluation.highlights },
    { label: '待改进', value: evaluation.improvements },
    { label: '建议', value: evaluation.suggestions }
  ].filter((item) => Boolean(item.value && item.value.trim()))
}

function scoreOf(evaluation: EvaluationDTO, key: DimensionKey): string {
  const value = evaluation[key]
  return value === null || value === undefined ? '—' : String(value)
}

function selectSupervisor(id: number): void {
  activeSupervisorId.value = id
}

/* ===== 课堂录音与转写 ===== */
const media = ref<SessionMedia>({ recording: null, transcript: null })
const mediaLoading = ref(false)
const uploadProgress = ref(0)
let pollTimer: ReturnType<typeof setTimeout> | undefined

// 优先用带短时票据的 playbackUrl：<audio> 的 Range 请求带不了 Authorization 头。
const recordingURL = computed(() => {
  const url = media.value.recording?.playbackUrl
  return url ? `${import.meta.env.VITE_API_BASE_URL || '/api/v1'}${url}` : ''
})

const transcriptStatusText = computed(() => {
  switch (media.value.transcript?.status) {
    case 'pending':
      return '转写排队中'
    case 'running':
      return '转写进行中'
    case 'done':
      return '转写已完成'
    case 'failed':
      return '转写失败'
    default:
      return '暂无转写'
  }
})

async function loadMedia(): Promise<void> {
  mediaLoading.value = true
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = undefined
  }
  try {
    media.value = await fetchSessionTranscriptApi(sessionId.value)
  } catch {
    // 录音/转写为增强项，任何失败都不得阻断评估数据面板。
    media.value = { recording: null, transcript: null }
  } finally {
    mediaLoading.value = false
  }
  if (
    media.value.transcript &&
    ['pending', 'running'].includes(media.value.transcript.status)
  ) {
    pollTimer = setTimeout(() => {
      void loadMedia()
    }, 2500)
  }
}

async function handleRecording(file: File): Promise<boolean> {
  try {
    await uploadSessionRecordingApi(sessionId.value, file, (value) => {
      uploadProgress.value = value
    })
    ElMessage.success('录音上传成功，转写完成后将触发 AI 评析')
    await loadMedia()
  } catch {
    // 错误提示由 interceptor 统一处理
  }
  return false
}

async function retryTranscript(): Promise<void> {
  try {
    await retrySessionTranscriptApi(sessionId.value)
    await loadMedia()
  } catch {
    // 错误提示由 interceptor 统一处理
  }
}

/* ===== 数据加载与提交 ===== */
async function load(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    const result = await fetchSessionEvaluationApi(sessionId.value)
    data.value = result
    const own = isSupervisor.value
      ? result.supervisorScores.find((item) => item.evaluatorId === auth.user?.id) ?? null
      : null
    form.value = own ? fromEvaluation(own) : emptyForm()

    if (!isSupervisor.value) {
      const exists = result.supervisorScores.some(
        (item) => item.evaluatorId === activeSupervisorId.value
      )
      if (!exists) {
        activeSupervisorId.value = latestSupervisor(result.supervisorScores)?.evaluatorId ?? null
      }
    }
    await loadMedia()
  } catch {
    error.value = true
    data.value = null
  } finally {
    loading.value = false
  }
}

function buildEvaluationPayload(): SupervisorEvaluationPayload | null {
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
  const payload = buildEvaluationPayload()
  if (!payload) {
    ElMessage.error('请先完成五个维度的评分')
    return
  }
  if (ownEvaluation.value) {
    try {
      await ElMessageBox.confirm(
        '你已提交过本次课的评分，继续将覆盖上次的五维评分与评语。',
        '覆盖确认',
        {
          type: 'warning',
          confirmButtonText: '覆盖提交',
          cancelButtonText: '取消'
        }
      )
    } catch {
      return
    }
  }
  submitting.value = true
  try {
    await submitSupervisorEvaluationApi(sessionId.value, payload)
    ElMessage.success('督导评分已提交')
    await load()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    submitting.value = false
  }
}

function goBack(): void {
  router.back()
}

watch(sessionId, () => {
  activeSupervisorId.value = null
  void load()
})

onMounted(() => {
  void load()
})

onUnmounted(() => {
  if (pollTimer) clearTimeout(pollTimer)
})
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
      </PageHeader>

      <!-- 场次信息 -->
      <div class="session-evaluation__meta">
        <div v-for="item in metaItems" :key="item.label" class="session-evaluation__meta-item">
          <span class="session-evaluation__meta-label">{{ item.label }}</span>
          <span class="session-evaluation__meta-value">{{ item.value }}</span>
        </div>
      </div>

      <!-- 课堂媒体全宽置顶：先看/听课堂，再评分 -->
      <section class="session-evaluation__card session-evaluation__media" aria-label="课堂录音与转写">
        <header class="session-evaluation__card-head">
          <div>
            <h2 class="session-evaluation__card-title">课堂录音与转写</h2>
            <p class="session-evaluation__muted">课堂录音仅本人与教学督导可回放</p>
          </div>
          <div v-if="media.recording" class="session-evaluation__media-actions">
            <el-tag size="small" effect="plain" round>{{ transcriptStatusText }}</el-tag>
            <el-upload
              v-if="canUpload"
              :show-file-list="false"
              accept=".mp3,.wav,.m4a"
              :before-upload="handleRecording"
            >
              <el-button size="small" plain>替换录音</el-button>
            </el-upload>
          </div>
        </header>

        <el-skeleton v-if="mediaLoading" :rows="3" animated />

        <template v-else-if="media.recording">
          <AudioPlayer
            v-if="recordingURL"
            :src="recordingURL"
            :name="media.recording.originalName"
          />
          <el-alert
            v-else
            type="info"
            :closable="false"
            show-icon
            title="当前账号无课堂录音回放权限，可查看下方脱敏转写。"
          />
          <el-progress
            v-if="uploadProgress > 0 && uploadProgress < 100"
            :percentage="uploadProgress"
          />
          <TranscriptViewer
            :transcript="media.transcript"
            :loading="mediaLoading"
            @retry="retryTranscript"
          />
        </template>

        <div v-else class="session-evaluation__media-empty">
          <div>
            <h3 class="session-evaluation__media-empty-title">本节课还没有课堂录音</h3>
            <p class="session-evaluation__muted">
              上传录音后自动转写，转写完成后会触发 AI 评析。
            </p>
          </div>
          <el-upload
            v-if="canUpload"
            :show-file-list="false"
            accept=".mp3,.wav,.m4a"
            :before-upload="handleRecording"
          >
            <el-button type="primary" size="large">上传课堂录音</el-button>
          </el-upload>
          <span v-else class="session-evaluation__muted">暂无课堂录音</span>
          <el-progress
            v-if="uploadProgress > 0 && uploadProgress < 100"
            :percentage="uploadProgress"
          />
        </div>
      </section>

      <!-- 督导：左侧评分与观察，右侧仅智能体参考 -->
      <div v-if="isSupervisor" class="session-evaluation__grid">
        <div class="session-evaluation__main">
          <section class="session-evaluation__card session-evaluation__supervisor-card">
            <header class="session-evaluation__card-head">
              <div>
                <h2 class="session-evaluation__card-title">督导评分与观察</h2>
                <p class="session-evaluation__muted">
                  五维原始分 1–5；我的督导总分为 0–100。督导分不参与 AI 综合分。
                </p>
              </div>
              <div class="session-evaluation__panel-tags">
                <AiBadge text="AI 参考" />
                <span class="session-evaluation__muted">口径 {{ formulaVersion }}</span>
              </div>
            </header>

            <div class="session-evaluation__viz">
              <div class="session-evaluation__viz-radar">
                <ScoreRadar :items="radarItems" :max="5" tone="ai" />
              </div>

              <div class="session-evaluation__bars" aria-label="五维双轨分数条">
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
                    <span class="session-evaluation__bar-value">
                      <em v-if="row.supervisorLevel">督导 {{ row.supervisorLevel }}</em>
                    </span>
                  </div>

                  <div class="session-evaluation__bar-line">
                    <span class="session-evaluation__bar-track-label">AI</span>
                    <div class="session-evaluation__bar-track">
                      <div
                        class="session-evaluation__bar-fill session-evaluation__bar-fill--ai"
                        :class="{ 'session-evaluation__bar-fill--observation': row.isObservation }"
                        :style="{ width: `${row.aiRatio}%` }"
                      />
                    </div>
                    <span class="session-evaluation__bar-num tabular-nums">
                      {{ row.aiScore === null ? '—' : row.aiScore }}
                    </span>
                  </div>

                  <div class="session-evaluation__bar-line">
                    <span
                      class="session-evaluation__bar-track-label session-evaluation__bar-track-label--supervisor"
                    >
                      督导
                    </span>
                    <div class="session-evaluation__bar-track">
                      <div
                        class="session-evaluation__bar-fill session-evaluation__bar-fill--supervisor"
                        :style="{ width: `${row.supervisorRatio}%` }"
                      />
                    </div>
                    <span class="session-evaluation__bar-num tabular-nums">
                      {{ row.supervisorScore === null ? '—' : row.supervisorScore }}
                    </span>
                  </div>
                </div>
                <p class="session-evaluation__bar-note">
                  五维为 1–5 原始分；督导分不参与 AI 综合分。「前沿与交叉学科」为观测项，不计入加权总分。
                </p>
              </div>
            </div>

            <div class="session-evaluation__summary">
              <div class="session-evaluation__total">
                <span class="session-evaluation__total-label">我的督导总分</span>
                <span
                  class="session-evaluation__total-value score-num"
                  :style="{ color: myTotalColor }"
                >
                  {{ myTotalText }}
                </span>
                <span class="session-evaluation__muted">满分 100，与五维 1–5 原始分分开显示</span>
              </div>
              <dl class="session-evaluation__facts">
                <div>
                  <dt>已评督导</dt>
                  <dd class="tabular-nums">{{ supervisorCount }} 人</dd>
                </div>
                <div>
                  <dt>智能体参考</dt>
                  <dd>
                    <AiBadge v-if="agentScore" text="AI 参考" />
                    <span v-else class="session-evaluation__muted">暂无 AI 评价</span>
                  </dd>
                </div>
                <div>
                  <dt>AI 总分</dt>
                  <dd class="tabular-nums">{{ agentTotalText }}</dd>
                </div>
              </dl>
            </div>

            <EvaluationForm
              v-model="form"
              :readonly="false"
              :submitting="submitting"
              @submit="handleSubmit"
            />

            <div v-if="otherSupervisors.length" class="session-evaluation__others">
              <el-collapse>
                <el-collapse-item
                  :title="`其他督导记录（${otherSupervisors.length}）`"
                  name="other-supervisors"
                >
                  <ul class="session-evaluation__other-list">
                    <li
                      v-for="item in otherSupervisors"
                      :key="item.evaluatorId"
                      class="session-evaluation__other-item"
                    >
                      <div class="session-evaluation__other-head">
                        <strong>{{ item.evaluatorName || `督导 #${item.evaluatorId}` }}</strong>
                        <span
                          class="session-evaluation__other-total tabular-nums"
                          :style="{ color: supervisorTotalTone(item) }"
                        >
                          {{ supervisorTotalText(item) }}
                        </span>
                      </div>

                      <div v-if="hasAnyDimension(item)" class="session-evaluation__dim-chips">
                        <span
                          v-for="dimension in EVALUATION_DIMENSIONS"
                          :key="`${item.evaluatorId}-${dimension.key}`"
                          class="session-evaluation__dim-chip"
                        >
                          {{ dimension.shortName }}
                          <strong class="tabular-nums">{{ scoreOf(item, dimension.key) }}</strong>
                        </span>
                      </div>

                      <dl
                        v-if="supervisorTextBlocks(item).length"
                        class="session-evaluation__other-text"
                      >
                        <div
                          v-for="block in supervisorTextBlocks(item)"
                          :key="`${item.evaluatorId}-${block.label}`"
                        >
                          <dt>{{ block.label }}</dt>
                          <dd>{{ block.value }}</dd>
                        </div>
                      </dl>
                    </li>
                  </ul>
                </el-collapse-item>
              </el-collapse>
            </div>
          </section>
        </div>

        <aside class="session-evaluation__side">
          <section class="session-evaluation__card">
            <header class="session-evaluation__card-head">
              <h2 class="session-evaluation__card-title">智能体参考</h2>
              <AiBadge v-if="agentScore" text="AI 参考" />
            </header>

            <div v-if="agentScore" class="session-evaluation__agent-total">
              <span class="session-evaluation__muted">AI 总分（0–100）</span>
              <strong class="score-num">{{ agentTotalText }}</strong>
            </div>

            <EvaluationCompare :agent="agentScore" hide-supervisor />

            <dl v-if="agentTextItems.length" class="session-evaluation__agent-text">
              <div v-for="item in agentTextItems" :key="item.label">
                <dt>{{ item.label }}</dt>
                <dd>{{ item.value }}</dd>
              </div>
            </dl>
          </section>
        </aside>
      </div>

      <!-- 教师/主任只读：督导评分逐人展示 + AI 参考 -->
      <template v-else>
        <section class="session-evaluation__card" aria-label="督导评分">
          <header class="session-evaluation__card-head">
            <div>
              <h2 class="session-evaluation__card-title">督导评分</h2>
              <p class="session-evaluation__muted">
                AI 综合分仅采用 AI 数据；督导评分作为人工参考，不参与综合分。
              </p>
            </div>
            <span class="session-evaluation__muted tabular-nums">共 {{ supervisorCount }} 位督导</span>
          </header>

          <template v-if="data.supervisorScores.length">
            <div class="session-evaluation__teacher-radar">
              <div class="session-evaluation__teacher-radar-head">
                <span>
                  雷达对照：AI + 当前督导
                  <strong>{{ selectedSupervisor?.evaluatorName || '—' }}</strong>
                </span>
                <span class="session-evaluation__muted">五维原始分 1–5</span>
              </div>
              <ScoreRadar :items="radarItems" :max="5" tone="ai" />
            </div>

            <ul class="session-evaluation__supervisor-list">
              <li
                v-for="item in data.supervisorScores"
                :key="item.evaluatorId"
                class="session-evaluation__supervisor-item"
                :class="{
                  'session-evaluation__supervisor-item--active':
                    item.evaluatorId === selectedSupervisor?.evaluatorId
                }"
              >
                <div class="session-evaluation__supervisor-item-head">
                  <div>
                    <strong>{{ item.evaluatorName || `督导 #${item.evaluatorId}` }}</strong>
                    <span
                      class="session-evaluation__supervisor-total tabular-nums"
                      :style="{ color: supervisorTotalTone(item) }"
                    >
                      {{ supervisorTotalText(item) }}
                    </span>
                  </div>
                  <el-button
                    size="small"
                    :type="item.evaluatorId === selectedSupervisor?.evaluatorId ? 'primary' : 'default'"
                    plain
                    @click="selectSupervisor(item.evaluatorId)"
                  >
                    {{ item.evaluatorId === selectedSupervisor?.evaluatorId ? '当前对照' : '对照此督导' }}
                  </el-button>
                </div>

                <div v-if="hasAnyDimension(item)" class="session-evaluation__dim-chips">
                  <span
                    v-for="dimension in EVALUATION_DIMENSIONS"
                    :key="`${item.evaluatorId}-${dimension.key}`"
                    class="session-evaluation__dim-chip"
                  >
                    {{ dimension.shortName }}
                    <strong class="tabular-nums">{{ scoreOf(item, dimension.key) }}</strong>
                  </span>
                </div>
                <p v-else class="session-evaluation__muted">仅评语，无评分</p>

                <dl
                  v-if="supervisorTextBlocks(item).length"
                  class="session-evaluation__other-text"
                >
                  <div
                    v-for="block in supervisorTextBlocks(item)"
                    :key="`${item.evaluatorId}-${block.label}`"
                  >
                    <dt>{{ block.label }}</dt>
                    <dd>{{ block.value }}</dd>
                  </div>
                </dl>
              </li>
            </ul>
          </template>

          <EmptyState v-else description="暂无督导评分" />
        </section>

        <section class="session-evaluation__card session-evaluation__ai-standard">
          <header class="session-evaluation__card-head">
            <div>
              <h2 class="session-evaluation__card-title">AI 评分标准与参考</h2>
              <p class="session-evaluation__muted">
                AI 仅依据已脱敏的课堂转写计分；督导评分不参与 AI 综合分。
              </p>
            </div>
            <div class="session-evaluation__panel-tags">
              <AiBadge text="AI 评价" />
              <span class="session-evaluation__muted">口径 {{ formulaVersion }}</span>
            </div>
          </header>

          <div v-if="agentScore" class="session-evaluation__agent-total">
            <span class="session-evaluation__muted">AI 总分（0–100）</span>
            <strong class="score-num">{{ agentTotalText }}</strong>
          </div>

          <ul class="session-evaluation__weights" aria-label="AI 评分维度与权重">
            <li v-for="dim in aiStandardDims" :key="dim.key" class="session-evaluation__weight">
              <span class="session-evaluation__weight-name">{{ dim.name }}</span>
              <span class="session-evaluation__weight-value tabular-nums">
                {{ dim.isObservation ? '观测项' : `${Math.round(dim.weight * 100)}%` }}
              </span>
            </li>
          </ul>

          <EvaluationCompare :agent="agentScore" hide-supervisor />

          <dl v-if="agentTextItems.length" class="session-evaluation__agent-text">
            <div v-for="item in agentTextItems" :key="item.label">
              <dt>{{ item.label }}</dt>
              <dd>{{ item.value }}</dd>
            </div>
          </dl>
        </section>
      </template>
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

    @media (max-width: 720px) {
      grid-template-columns: repeat(2, 1fr);
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

  &__card {
    padding: var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__card-head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-3);
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__card-title {
    font-size: var(--font-size-xl);
    color: var(--color-text-primary);
  }

  &__muted {
    font-size: var(--font-size-xs);
    font-weight: 400;
    line-height: 1.7;
    color: var(--color-text-tertiary);
  }

  &__panel-tags {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
  }

  &__media {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
  }

  &__media-actions {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
  }

  &__media-empty {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-4);
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-5);
    background-color: color-mix(in srgb, var(--color-primary) 5%, #ffffff);
    border: 1px dashed color-mix(in srgb, var(--color-primary) 35%, #ffffff);
    border-radius: var(--radius-md);
  }

  &__media-empty-title {
    margin-bottom: var(--spacing-1);
    font-size: var(--font-size-base);
    color: var(--color-text-primary);
  }

  &__grid {
    display: grid;
    grid-template-columns: minmax(0, 2fr) minmax(280px, 1fr);
    gap: var(--spacing-4);
    align-items: start;

    @media (max-width: 1180px) {
      grid-template-columns: 1fr;
    }
  }

  &__main,
  &__side {
    min-width: 0;
  }

  &__side {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
  }

  &__viz {
    display: grid;
    grid-template-columns: minmax(240px, 320px) minmax(0, 1fr);
    gap: var(--spacing-6);
    align-items: center;
    padding-bottom: var(--spacing-4);
    margin-bottom: var(--spacing-4);
    border-bottom: 1px dashed var(--color-divider);

    @media (max-width: 900px) {
      grid-template-columns: 1fr;
    }
  }

  &__viz-radar {
    min-width: 0;
  }

  &__bars {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
  }

  &__bar + &__bar {
    margin-top: var(--spacing-1);
  }

  &__bar-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-2);
    margin-bottom: var(--spacing-1);
  }

  &__bar-name {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--spacing-2);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__bar-value {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);

    em {
      font-style: normal;
    }
  }

  &__bar-line {
    display: grid;
    grid-template-columns: 34px minmax(0, 1fr) 24px;
    gap: var(--spacing-2);
    align-items: center;
    margin-top: 4px;
  }

  &__bar-track-label {
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--color-ai-deep);

    &--supervisor {
      color: var(--color-teacher);
    }
  }

  &__bar-track {
    height: 7px;
    overflow: hidden;
    background-color: var(--color-divider);
    border-radius: var(--radius-sm);
  }

  &__bar-fill {
    height: 100%;
    border-radius: var(--radius-sm);
    transition: width 0.3s ease;

    &--ai {
      background-color: var(--color-ai);
    }

    &--supervisor {
      background-color: var(--color-teacher);
    }

    &--observation {
      opacity: 0.6;
    }
  }

  &__bar-num {
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--color-text-primary);
    text-align: right;
  }

  &__bar-note {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-xs);
    line-height: 1.6;
    color: var(--color-text-tertiary);
  }

  &__summary {
    display: grid;
    grid-template-columns: minmax(180px, 240px) minmax(0, 1fr);
    gap: var(--spacing-6);
    align-items: start;
    padding: var(--spacing-4);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-page);
    border-radius: var(--radius-md);

    @media (max-width: 720px) {
      grid-template-columns: 1fr;
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
    flex-wrap: wrap;
    gap: var(--spacing-3) var(--spacing-6);

    > div {
      display: flex;
      flex-direction: column;
      gap: 2px;
      font-size: var(--font-size-sm);
    }

    dt {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    dd {
      font-weight: 500;
      color: var(--color-text-primary);
    }
  }

  &__others {
    margin-top: var(--spacing-5);
    border-top: 1px dashed var(--color-divider);
  }

  &__other-list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-4);
  }

  &__other-item {
    padding: var(--spacing-3);
    background-color: var(--color-bg-page);
    border-radius: var(--radius-md);
  }

  &__other-head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: var(--spacing-2);
  }

  &__other-total {
    font-size: var(--font-size-sm);
    font-weight: 600;
  }

  &__dim-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-1) var(--spacing-2);
    margin-bottom: var(--spacing-2);
  }

  &__dim-chip {
    display: inline-flex;
    gap: 4px;
    align-items: baseline;
    padding: 2px 8px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: 999px;

    strong {
      color: var(--color-text-primary);
    }
  }

  &__other-text {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-1);

    > div {
      display: grid;
      grid-template-columns: 72px minmax(0, 1fr);
      gap: var(--spacing-2);
    }

    dt {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    dd {
      margin: 0;
      font-size: var(--font-size-sm);
      line-height: 1.7;
      color: var(--color-text-secondary);
      white-space: pre-wrap;
    }
  }

  &__agent-total {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--spacing-3);
    padding: var(--spacing-3) var(--spacing-4);
    margin-bottom: var(--spacing-3);
    background-color: var(--color-ai-bg);
    border-radius: var(--radius-md);

    strong {
      font-size: var(--font-size-2xl);
      color: var(--color-ai-deep);
    }
  }

  &__agent-text {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    padding-top: var(--spacing-3);
    margin-top: var(--spacing-3);
    border-top: 1px dashed color-mix(in srgb, var(--color-ai) 24%, #ffffff);

    dt {
      font-size: var(--font-size-xs);
      font-weight: 600;
      color: var(--color-ai-deep);
    }

    dd {
      margin: 2px 0 0;
      font-size: var(--font-size-sm);
      line-height: 1.7;
      color: var(--color-text-secondary);
      white-space: pre-wrap;
    }
  }

  &__teacher-radar {
    padding: var(--spacing-4);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-page);
    border-radius: var(--radius-md);
  }

  &__teacher-radar-head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: var(--spacing-2);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__supervisor-list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
  }

  &__supervisor-item {
    padding: var(--spacing-4);
    background-color: var(--color-bg-page);
    border: 1px solid transparent;
    border-radius: var(--radius-md);

    &--active {
      background-color: color-mix(in srgb, var(--color-primary) 5%, #ffffff);
      border-color: color-mix(in srgb, var(--color-primary) 28%, #ffffff);
    }
  }

  &__supervisor-item-head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-2);
  }

  &__supervisor-total {
    margin-left: var(--spacing-3);
    font-size: var(--font-size-sm);
    font-weight: 600;
  }

  &__ai-standard {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
  }

  &__weights {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
  }

  &__weight {
    display: inline-flex;
    gap: var(--spacing-2);
    align-items: baseline;
    padding: 4px 10px;
    font-size: var(--font-size-xs);
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: 999px;
  }

  &__weight-name {
    color: var(--color-text-secondary);
  }

  &__weight-value {
    font-weight: 600;
    color: var(--color-ai-deep);
  }
}
</style>
