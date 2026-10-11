<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AiBadge from '@/components/common/AiBadge.vue'
import QualityBadge from '@/components/common/QualityBadge.vue'
import QualityCompass from '@/components/evaluation/QualityCompass.vue'
import ScoreDimensionsCard from '@/components/evaluation/ScoreDimensionsCard.vue'
import { fetchTeacherEvaluationsApi, fetchTeacherSummaryApi } from '@/api/teacher'
import { useAuthStore } from '@/stores/auth'
import { useSemester } from '@/composables/useSemester'
import { flagText } from '@/constants'
import { scoreTone, scoreToneColor } from '@/utils/format'
import type { TeacherSummary, TeacherTimelineItem } from '@/types/teacher'

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
const loading = ref(true)
const error = ref(false)

const flags = computed(() => summary.value?.flags ?? [])

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
        <h2 class="my-quality__title">授课快照</h2>
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
}
</style>
