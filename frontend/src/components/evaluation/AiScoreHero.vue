<script setup lang="ts">
import { computed } from 'vue'
import AiBadge from '@/components/common/AiBadge.vue'
import QualityCompass from '@/components/evaluation/QualityCompass.vue'
import { flagText } from '@/constants'
import type { EvaluationDTO } from '@/types/evaluation'
import type { ScoreSummary } from '@/types/teacher'

/**
 * AI 评价总览横带（画像链路首屏）。
 * 左：AI 单环仪表；中：样本口径读数与警示；右：基于真实评价数据的综合分析与提优建议。
 * 分析文本只做模板化归纳，不接入 mock / 不编造 LLM 口吻。
 */
const props = withDefaults(
  defineProps<{
    summary: ScoreSummary | null
    loading?: boolean
    /** 学期口径标注（可选） */
    semester?: string
    /** 最近一条含文本的 AI 评价（来自真实时间线） */
    recentEvaluation?: EvaluationDTO | null
  }>(),
  { loading: false, semester: '', recentEvaluation: null }
)

const flags = computed(() => props.summary?.flags ?? [])

function trimEnd(text: string): string {
  return text.trim().replace(/[。；;，,]+$/, '')
}

/** 最弱维度：仅看非观测项，取 AI 分最低者；缺失分不参与。 */
const weakestDimension = computed(() => {
  const rows = (props.summary?.dimensions ?? [])
    .filter((dimension) => !dimension.isObservation)
    .map((dimension) => ({
      name: dimension.name,
      score: dimension.agentScore ?? dimension.score
    }))
    .filter((row): row is { name: string; score: number } => row.score !== null && row.score !== undefined)
  if (!rows.length) return null
  return rows.reduce((min, row) => (row.score < min.score ? row : min))
})

const strongestDimension = computed(() => {
  const rows = (props.summary?.dimensions ?? [])
    .filter((dimension) => !dimension.isObservation)
    .map((dimension) => ({
      name: dimension.name,
      score: dimension.agentScore ?? dimension.score
    }))
    .filter((row): row is { name: string; score: number } => row.score !== null && row.score !== undefined)
  if (!rows.length) return null
  return rows.reduce((max, row) => (row.score > max.score ? row : max))
})

const analysis = computed(() => {
  const summary = props.summary
  const recent = props.recentEvaluation
  const weakest = weakestDimension.value
  if (!summary || !recent || !weakest || summary.sample.agentCount <= 0) return null

  const sentences = [
    `近 ${summary.sample.agentCount} 次课中，「${weakest.name}」均分最低（${weakest.score.toFixed(2)}）。`
  ]
  const note = trimEnd(recent.improvements || recent.suggestions || recent.highlights || '')
  if (note) {
    sentences.push(`最近一次 AI 评价指出：${note}。`)
  } else if (strongestDimension.value && strongestDimension.value.name !== weakest.name) {
    const gap = Math.max(0, strongestDimension.value.score - weakest.score)
    sentences.push(
      `与本学期最高维度「${strongestDimension.value.name}」（${strongestDimension.value.score.toFixed(2)}）相差 ${gap.toFixed(2)} 分，可作为下一轮听课跟进重点。`
    )
  }
  const action = trimEnd(recent.suggestions || '')
  if (action && action !== note) {
    sentences.push(`建议优先落实：${action}。`)
  }
  return {
    sentences,
    sampleCount: summary.sample.agentCount
  }
})
</script>

<template>
  <section class="ai-hero" :class="{ 'ai-hero--analysis': Boolean(analysis) }">
    <el-skeleton v-if="loading" :rows="6" animated />

    <template v-else-if="summary">
      <div class="ai-hero__gauge">
        <QualityCompass :summary="summary" size="md" />
      </div>

      <div class="ai-hero__body">
        <p class="ai-hero__eyebrow">
          <AiBadge text="AI 评价" />
          <span class="ai-hero__eyebrow-text">本学期综合分</span>
          <span v-if="semester" class="ai-hero__semester tabular-nums">{{ semester }}</span>
        </p>

        <dl class="ai-hero__facts">
          <div class="ai-hero__fact">
            <dt>AI 已评</dt>
            <dd class="tabular-nums">{{ summary.sample.agentCount }} 场</dd>
          </div>
          <div class="ai-hero__fact">
            <dt>口径版本</dt>
            <dd class="tabular-nums">{{ summary.formulaVersion || '—' }}</dd>
          </div>
          <div class="ai-hero__fact">
            <dt>样本</dt>
            <dd>{{ summary.sample.sampleSufficient ? '充足' : '不足' }}</dd>
          </div>
        </dl>

        <div v-if="flags.length" class="ai-hero__flags">
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

      <aside v-if="analysis" class="ai-hero__analysis ai-panel" aria-label="AI 综合分析与提优建议">
        <p class="ai-hero__analysis-title">AI 综合分析与提优建议</p>
        <p v-for="(sentence, index) in analysis.sentences" :key="index" class="ai-hero__analysis-text">
          {{ sentence }}
        </p>
        <p class="ai-hero__analysis-source">
          基于 AI 评价生成，样本 {{ analysis.sampleCount }} 次
        </p>
      </aside>
    </template>

    <p v-else class="ai-hero__empty">暂无 AI 评价数据</p>
  </section>
</template>

<style scoped lang="scss">
.ai-hero {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  column-gap: var(--spacing-6);
  align-items: center;
  padding: var(--spacing-4) var(--spacing-6);
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);

  &--analysis {
    grid-template-columns: auto minmax(0, 1fr) minmax(280px, 0.9fr);
  }

  &__gauge {
    display: flex;
    align-items: center;
    padding-right: var(--spacing-6);
    border-right: 1px solid var(--color-divider);
  }

  &__body {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-3) var(--spacing-6);
    align-items: center;
    min-width: 0;
  }

  &__eyebrow {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__eyebrow-text {
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__semester {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__facts {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-6);
    margin: 0;
  }

  &__fact {
    dt {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    dd {
      margin: 2px 0 0;
      font-size: var(--font-size-lg);
      font-weight: 600;
      line-height: 1.2;
      color: var(--color-text-primary);
    }
  }

  &__flags {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
  }

  &__analysis {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
    align-self: stretch;
    padding: var(--spacing-4);
    margin-left: var(--spacing-2);
  }

  &__analysis-title {
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-ai-deep);
  }

  &__analysis-text {
    font-size: var(--font-size-xs);
    line-height: 1.75;
    color: var(--color-text-secondary);
  }

  &__analysis-source {
    padding-top: var(--spacing-2);
    margin-top: auto;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    border-top: 1px dashed color-mix(in srgb, var(--color-ai) 28%, #ffffff);
  }

  &__empty {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}

@media (max-width: 1200px) {
  .ai-hero--analysis {
    grid-template-columns: auto minmax(0, 1fr);

    .ai-hero__analysis {
      grid-column: 1 / -1;
      margin-left: 0;
    }
  }
}

@media (max-width: 960px) {
  .ai-hero,
  .ai-hero--analysis {
    grid-template-columns: 1fr;
    row-gap: var(--spacing-6);
    padding: var(--spacing-6);
  }

  .ai-hero__gauge {
    justify-content: center;
    padding-right: 0;
    padding-bottom: var(--spacing-6);
    border-right: none;
    border-bottom: 1px solid var(--color-divider);
  }
}
</style>
