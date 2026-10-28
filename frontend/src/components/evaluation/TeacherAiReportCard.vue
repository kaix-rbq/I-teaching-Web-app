<script setup lang="ts">
import { computed, ref } from 'vue'
import AiBadge from '@/components/common/AiBadge.vue'
import { formatDateTime } from '@/utils/format'
import type { AiReportEnvelope } from '@/types/agent'

type ReportTabKey = 'highlights' | 'trend' | 'directions' | 'suggestions'

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

const TABS: Array<{ key: ReportTabKey; label: string }> = [
  { key: 'highlights', label: '亮点' },
  { key: 'trend', label: '趋势解读' },
  { key: 'directions', label: '可提优方向' },
  { key: 'suggestions', label: '提优建议' }
]

const activeTab = ref<ReportTabKey>('highlights')
const tablist = ref<HTMLElement | null>(null)
const uid = `ai-report-tabs-${Math.random().toString(36).slice(2, 8)}`
const activeTabId = computed(() => `${uid}-tab-${activeTab.value}`)
const activePanelId = computed(() => `${uid}-panel-${activeTab.value}`)

function selectTab(key: ReportTabKey): void {
  activeTab.value = key
}

function focusTab(index: number): void {
  const buttons = tablist.value?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
  buttons?.[index]?.focus()
}

function onTabKeydown(event: KeyboardEvent, index: number): void {
  let next = index
  if (event.key === 'ArrowRight') next = (index + 1) % TABS.length
  else if (event.key === 'ArrowLeft') next = (index - 1 + TABS.length) % TABS.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = TABS.length - 1
  else return
  event.preventDefault()
  activeTab.value = TABS[next].key
  focusTab(next)
}

const activeEmpty = computed(() => {
  if (!report.value) return true
  switch (activeTab.value) {
    case 'highlights':
      return !report.value.highlights.trim()
    case 'trend':
      return !report.value.trendComment.trim()
    case 'directions':
      return report.value.directions.length === 0
    case 'suggestions':
      return report.value.suggestions.length === 0
    default:
      return true
  }
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

    <!-- 固定摘要：切换标签不移动，保持一眼可见 -->
    <p class="ai-report-card__summary">
      {{ report.summary || '本次暂无该项内容' }}
    </p>

    <div class="ai-report-card__metrics">
      <span>已评 {{ report.evaluatedCount }} / {{ report.sessionCount }} 场</span>
      <span>置信度 {{ confidencePercent }}</span>
      <span v-if="lowConfidence" class="ai-report-card__warning">低置信，仅供参考</span>
    </div>

    <div class="ai-report-card__tabs-wrap">
      <div
        ref="tablist"
        class="ai-report-card__tabs"
        role="tablist"
        aria-label="AI 综合总结内容"
      >
        <button
          v-for="(tab, index) in TABS"
          :id="`${uid}-tab-${tab.key}`"
          :key="tab.key"
          type="button"
          role="tab"
          class="ai-report-card__tab"
          :class="{ 'ai-report-card__tab--active': activeTab === tab.key }"
          :aria-selected="activeTab === tab.key"
          :aria-controls="`${uid}-panel-${tab.key}`"
          :tabindex="activeTab === tab.key ? 0 : -1"
          @click="selectTab(tab.key)"
          @keydown="onTabKeydown($event, index)"
        >
          {{ tab.label }}
        </button>
      </div>
    </div>

    <div
      :id="activePanelId"
      class="ai-report-card__panel"
      role="tabpanel"
      :aria-labelledby="activeTabId"
      tabindex="0"
    >
      <p v-if="activeEmpty" class="ai-report-card__empty">本次暂无该项内容</p>

      <p v-else-if="activeTab === 'highlights'" class="ai-report-card__text">
        {{ report.highlights }}
      </p>

      <p v-else-if="activeTab === 'trend'" class="ai-report-card__text">
        {{ report.trendComment }}
      </p>

      <ol v-else-if="activeTab === 'directions'" class="ai-report-card__list">
        <li v-for="(item, index) in report.directions" :key="`direction-${index}`">{{ item }}</li>
      </ol>

      <ol v-else class="ai-report-card__list">
        <li v-for="(item, index) in report.suggestions" :key="`suggestion-${index}`">{{ item }}</li>
      </ol>
    </div>

    <footer class="ai-report-card__foot">
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

  &__metrics {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2) var(--spacing-4);
    align-items: baseline;
    margin-top: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__warning {
    color: var(--color-warning);
  }

  &__tabs-wrap {
    margin-top: var(--spacing-4);
    border-bottom: 1px solid var(--color-divider);
  }

  /* 移动端不换行，保持横向滚动；键盘焦点使用原生 focus-visible */
  &__tabs {
    display: flex;
    flex-wrap: nowrap;
    gap: var(--spacing-1);
    overflow-x: auto;
    scrollbar-width: thin;
    -webkit-overflow-scrolling: touch;
  }

  &__tab {
    flex: 0 0 auto;
    padding: var(--spacing-2) var(--spacing-3);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
    cursor: pointer;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    transition:
      color var(--duration-fast) var(--ease-out-soft),
      border-color var(--duration-fast) var(--ease-out-soft);

    &:hover {
      color: var(--color-text-secondary);
    }

    &:focus-visible {
      outline: 2px solid var(--color-primary);
      outline-offset: -2px;
    }

    &--active {
      font-weight: 600;
      color: var(--color-ai-deep);
      border-bottom-color: var(--color-ai);
    }
  }

  &__panel {
    min-height: 88px;
    padding-top: var(--spacing-3);

    &:focus-visible {
      outline: 2px solid var(--color-primary);
      outline-offset: 2px;
    }
  }

  &__text {
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-secondary);
  }

  &__empty {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
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
    padding-top: var(--spacing-3);
    margin-top: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    border-top: 1px dashed var(--color-divider);
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
