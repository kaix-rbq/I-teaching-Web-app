<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AiBadge from '@/components/common/AiBadge.vue'
import QualityBadge from '@/components/common/QualityBadge.vue'
import QualityCompass from '@/components/evaluation/QualityCompass.vue'
import ScoreDimensionsCard from '@/components/evaluation/ScoreDimensionsCard.vue'
import EvaluationTimeline from '@/components/evaluation/EvaluationTimeline.vue'
import { fetchTeacherEvaluationsApi, fetchTeacherSummaryApi } from '@/api/teacher'
import { fetchAgentSuggestionsApi } from '@/api/agent'
import { useAuthStore } from '@/stores/auth'
import { useSemester } from '@/composables/useSemester'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { AgentSuggestion } from '@/types/agent'
import type { TeacherSummary, TeacherTimelineItem } from '@/types/teacher'

/**
 * 我的质量档案（教师「质量主页」，《前端设计-new》§5.7）。
 * 数据全部来自教师查本人接口（后端已裁剪，无他人数据）；只读。
 * 顶栏全局学期口径驱动（useSemester），与质量驾驶舱同源。
 */
const router = useRouter()
const auth = useAuthStore()
const { semester } = useSemester()

const summary = ref<TeacherSummary | null>(null)
const timeline = ref<TeacherTimelineItem[]>([])
const suggestions = ref<AgentSuggestion[]>([])
const loading = ref(true)
const timelineLoading = ref(false)
const suggestionsLoading = ref(false)
const error = ref(false)

const FLAG_TEXT: Record<string, string> = {
  no_data: '暂无评价',
  sample_insufficient: '样本不足，当前分数代表性有限',
  disjoint: '督导与智能体评价尚未对齐，当前分数代表性有限',
  formula_mixed: '口径版本混杂，请谨慎解读'
}

const flags = computed(() => summary.value?.flags ?? [])

const courseNameById = computed(() => {
  const map = new Map<number, string>()
  for (const course of summary.value?.courses ?? []) {
    map.set(course.courseId, course.courseName)
  }
  return map
})

/** 数字一致性铁律：直接展示后端值，不做前端四舍五入 */
function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

function flagText(flag: string): string {
  return FLAG_TEXT[flag] ?? flag
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

  timelineLoading.value = true
  try {
    const page = await fetchTeacherEvaluationsApi(id, {
      semester: semester.value || undefined,
      pageSize: 50
    })
    timeline.value = page.list
  } catch {
    timeline.value = []
  } finally {
    timelineLoading.value = false
  }

  // AI 提优建议摘要：跨课程聚合，按时间倒序取 3 条（阶段③，当前为演示数据）
  suggestionsLoading.value = true
  try {
    const courses = summary.value?.courses ?? []
    const grouped = await Promise.all(
      courses.map((course) => fetchAgentSuggestionsApi(course.courseId).catch(() => [] as AgentSuggestion[]))
    )
    suggestions.value = grouped
      .flat()
      .sort((a, b) => (a.sessionDate < b.sessionDate ? 1 : -1))
      .slice(0, 3)
  } catch {
    suggestions.value = []
  } finally {
    suggestionsLoading.value = false
  }
}

function goImprove(courseId: number): void {
  void router.push({ name: 'course-improve', params: { id: courseId } })
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
      <!-- 三卡横排：我的综合分 | 五维雷达 | 本学期样本 -->
      <section class="my-quality__top">
        <div class="my-quality__card">
          <h2 class="my-quality__card-title">我的综合分</h2>
          <QualityCompass :summary="summary" size="lg" />
        </div>

        <div class="my-quality__card">
          <ScoreDimensionsCard :summary="summary" />
        </div>

        <div class="my-quality__card">
          <h2 class="my-quality__card-title">本学期样本</h2>
          <dl class="my-quality__facts">
            <div>
              <dt>被评次数</dt>
              <dd class="tabular-nums">{{ summary.sample.evaluatedCount }} 次</dd>
            </div>
            <div>
              <dt>评价来源</dt>
              <dd class="tabular-nums">督导 {{ summary.sample.supervisorCount }} · AI {{ summary.sample.agentCount }}</dd>
            </div>
            <div>
              <dt>双源对齐</dt>
              <dd class="tabular-nums">{{ summary.sample.alignedCount }} 场</dd>
            </div>
            <div>
              <dt>口径版本</dt>
              <dd>{{ summary.formulaVersion || '—' }}</dd>
            </div>
          </dl>
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
          <p v-else class="my-quality__caliber-ok">样本与口径无异常提示</p>
          <p class="my-quality__hint">进步幅度将在历史数据齐备后提供。</p>
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

      <!-- 全部评价流 -->
      <section class="my-quality__block">
        <h2 class="my-quality__title">全部评价流</h2>
        <EvaluationTimeline :items="timeline" :loading="timelineLoading" empty-text="本学期暂无评价记录" />
      </section>

      <!-- AI 提优建议摘要（阶段③；当前为演示数据） -->
      <section class="my-quality__block">
        <h2 class="my-quality__title">
          AI 提优建议摘要
          <el-tag size="small" effect="plain" round>阶段③ · 演示数据</el-tag>
        </h2>
        <el-skeleton v-if="suggestionsLoading" :rows="3" animated />
        <template v-else>
          <ul v-if="suggestions.length" class="my-quality__suggestions">
            <li
              v-for="suggestion in suggestions"
              :key="suggestion.id"
              class="my-quality__suggestion"
            >
              <div class="my-quality__suggestion-head">
                <AiBadge text="AI 建议" />
                <span class="my-quality__suggestion-date tabular-nums">{{ suggestion.sessionDate }}</span>
                <span class="my-quality__suggestion-course">
                  {{ courseNameById.get(suggestion.courseId) || `课程 #${suggestion.courseId}` }}
                  <template v-if="suggestion.topic"> · {{ suggestion.topic }}</template>
                </span>
                <span class="my-quality__suggestion-confidence">
                  置信 {{ Math.round(suggestion.confidence * 100) }}%
                </span>
              </div>
              <p class="my-quality__suggestion-text">{{ suggestion.summary }}</p>
              <el-button
                class="my-quality__improve-btn"
                type="primary"
                plain
                size="small"
                round
                @click="goImprove(suggestion.courseId)"
              >
                进入提优页 →
              </el-button>
            </li>
          </ul>
          <p v-else class="my-quality__muted">暂无 AI 提优建议（本学期课程尚无智能体评价）</p>
        </template>
      </section>
    </template>
  </div>
</template>

<style scoped lang="scss">
.my-quality {
  &__ethics {
    margin-bottom: var(--spacing-4);
  }

  &__top {
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

  &__card {
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

  &__hint {
    margin-top: var(--spacing-2);
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

  &__suggestions {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
    padding: 0;
    margin: 0;
    list-style: none;
  }

  &__suggestion {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    padding: var(--spacing-3) var(--spacing-4);
    background-color: var(--color-ai-bg);
    border: 1px dashed var(--color-ai-bright);
    border-radius: var(--radius-ai, var(--radius-md));
    box-shadow: var(--shadow-ai, none);
  }

  &__suggestion-head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
    font-size: var(--font-size-xs);
    color: var(--color-text-secondary);
  }

  &__suggestion-date {
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__suggestion-course {
    color: var(--color-text-secondary);
  }

  &__suggestion-confidence {
    margin-left: auto;
    color: var(--color-text-tertiary);
  }

  &__suggestion-text {
    line-height: 1.7;
    color: var(--color-text-secondary);
  }

  &__muted {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
