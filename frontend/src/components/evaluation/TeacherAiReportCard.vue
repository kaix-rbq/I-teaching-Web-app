<script setup lang="ts">
import { computed } from 'vue'
import AiBadge from '@/components/common/AiBadge.vue'
import { formatDateTime } from '@/utils/format'
import type { AiReportEnvelope } from '@/types/agent'

const props = withDefaults(
  defineProps<{
    envelope: AiReportEnvelope | null
    loading?: boolean
    error?: boolean
  }>(),
  {
    loading: false,
    error: false
  }
)

const emit = defineEmits<{ (e: 'reload'): void }>()

const LOW_CONFIDENCE = 0.4

const report = computed(() => props.envelope?.report ?? null)
const status = computed(() => props.envelope?.status ?? 'pending')
const hasContent = computed(
  () =>
    Boolean(
      report.value &&
        (report.value.summary ||
          report.value.highlights ||
          report.value.trendComment ||
          report.value.directions.length ||
          report.value.suggestions.length)
    )
)
const pending = computed(() => !props.error && status.value === 'pending')
const running = computed(() => status.value === 'running')
const failed = computed(() => status.value === 'failed')
const confidence = computed(() => report.value?.confidence ?? null)
const lowConfidence = computed(
  () => confidence.value !== null && confidence.value < LOW_CONFIDENCE
)
const confidencePercent = computed(() => {
  if (confidence.value === null) return '—'
  return `${Math.round(Math.max(0, Math.min(1, confidence.value)) * 100)}%`
})
</script>

<template>
  <el-skeleton v-if="loading" :rows="4" animated />

  <div v-else-if="error" class="ai-report-card__state">
    <p class="ai-report-card__state-title">AI 综合总结加载失败</p>
    <el-button size="small" @click="emit('reload')">重新加载</el-button>
  </div>

  <div v-else-if="failed" class="ai-report-card__state">
    <p class="ai-report-card__state-title">
      {{ report?.errorMessage || envelope?.errorMessage || '生成失败，请稍后刷新' }}
    </p>
    <el-button size="small" @click="emit('reload')">重新加载</el-button>
  </div>

  <div v-else-if="pending" class="ai-report-card__state">
    <p class="ai-report-card__state-title">AI 综合总结待生成</p>
    <p class="ai-report-card__state-desc">完成课次 AI 评析并满足样本要求后自动生成</p>
  </div>

  <div v-else-if="running && !hasContent" class="ai-report-card__state">
    <p class="ai-report-card__state-title">AI 综合总结正在生成</p>
    <p class="ai-report-card__state-desc">生成完成后会自动刷新</p>
  </div>

  <template v-else-if="report">
    <header class="ai-report-card__head">
      <div class="ai-report-card__title-wrap">
        <AiBadge text="AI 总结" />
        <h3 class="ai-report-card__title">AI 综合总结</h3>
      </div>
      <el-tag v-if="running" type="warning" size="small" effect="light" round>更新中</el-tag>
    </header>

    <p class="ai-report-card__summary">{{ report.summary || '暂无总结' }}</p>

    <div v-if="report.highlights" class="ai-report-card__block">
      <h4 class="ai-report-card__label">亮点</h4>
      <p class="ai-report-card__text">{{ report.highlights }}</p>
    </div>

    <div v-if="report.trendComment" class="ai-report-card__block">
      <h4 class="ai-report-card__label">趋势解读</h4>
      <p class="ai-report-card__text">{{ report.trendComment }}</p>
    </div>

    <div v-if="report.directions.length" class="ai-report-card__block">
      <h4 class="ai-report-card__label">可提优方向</h4>
      <ol class="ai-report-card__list">
        <li v-for="(item, index) in report.directions" :key="`direction-${index}`">{{ item }}</li>
      </ol>
    </div>

    <div v-if="report.suggestions.length" class="ai-report-card__block">
      <h4 class="ai-report-card__label">提优建议</h4>
      <ol class="ai-report-card__list">
        <li v-for="(item, index) in report.suggestions" :key="`suggestion-${index}`">{{ item }}</li>
      </ol>
    </div>

    <footer class="ai-report-card__foot">
      <span>已评 {{ report.evaluatedCount }} / {{ report.sessionCount }} 场</span>
      <span>置信度 {{ confidencePercent }}</span>
      <span v-if="lowConfidence" class="ai-report-card__warning">低置信，仅供参考</span>
      <span>生成于 {{ formatDateTime(report.generatedAt) }}</span>
      <span class="ai-report-card__meta">
        {{ report.modelVersion || '未知模型' }} · {{ report.promptVersion || '未知提示词版本' }}
      </span>
    </footer>
  </template>

  <div v-else class="ai-report-card__state">
    <p class="ai-report-card__state-title">AI 综合总结待生成</p>
    <p class="ai-report-card__state-desc">完成课次 AI 评析并满足样本要求后自动生成</p>
  </div>
</template>

<style scoped lang="scss">
.ai-report-card {
  &__head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2) var(--spacing-3);
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-3);
  }

  &__title-wrap {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
  }

  &__title {
    font-size: var(--font-size-lg);
    color: var(--color-text-primary);
  }

  &__summary {
    font-size: var(--font-size-sm);
    line-height: 1.8;
    color: var(--color-text-secondary);
  }

  &__block {
    margin-top: var(--spacing-4);
  }

  &__label {
    margin-bottom: var(--spacing-2);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__text {
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-secondary);
  }

  &__list {
    padding-left: 1.25rem;
    font-size: var(--font-size-sm);
    line-height: 1.8;
    color: var(--color-text-secondary);
    list-style: decimal;

    li + li {
      margin-top: var(--spacing-1);
    }
  }

  &__foot {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2) var(--spacing-4);
    align-items: baseline;
    padding-top: var(--spacing-4);
    margin-top: var(--spacing-4);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    border-top: 1px dashed var(--color-divider);
  }

  &__warning {
    color: var(--color-warning);
  }

  &__meta {
    margin-left: auto;
  }

  &__state {
    padding: var(--spacing-4) 0;
  }

  &__state-title {
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-secondary);
  }

  &__state-desc {
    margin: var(--spacing-2) 0 var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }
}
</style>
