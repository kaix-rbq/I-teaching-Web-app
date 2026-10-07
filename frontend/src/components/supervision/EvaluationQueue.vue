<script setup lang="ts">
import { computed, ref } from 'vue'
import dayjs from 'dayjs'
import type { SupervisionPlan } from '@/types/supervision'
import { formatDate } from '@/utils/format'
import EmptyState from '@/components/common/EmptyState.vue'

/**
 * 督导待评课队列（工作台核心组件）。
 * 通过「今日 / 本周 / 本月 / 未评」按钮切换卡片内容：
 *   - 评估入口按授课日期闸门控制：**未来课次仅预览（不开放评估）**；
 *     到达/超过授课日期后自动暴露「去评估」（未建档则「创建授课记录并评估」）；
 *   - 「未评」专门列出**已过授课日期且尚未评价**的记录（不含今日），便于补录过去课程评分；
 *   - **已评估的听评课安排一律不再出现**：提交评分后授课记录状态变为 evaluated，
 *     从本队列清除，改由「课程列表 → 课程综合评分 → 历史授课记录」查看。
 */
const props = withDefaults(
  defineProps<{
    plans: SupervisionPlan[]
    loading?: boolean
  }>(),
  { loading: false }
)

const emit = defineEmits<{
  create: [plan: SupervisionPlan]
  evaluate: [plan: SupervisionPlan]
}>()

type RangeKey = 'day' | 'week' | 'month' | 'pending'

const range = ref<RangeKey>('day')

const RANGES: { key: RangeKey; label: string }[] = [
  { key: 'day', label: '今日' },
  { key: 'week', label: '本周' },
  { key: 'month', label: '本月' },
  { key: 'pending', label: '未评' }
]

const rangedPlans = computed<SupervisionPlan[]>(() => {
  const today = dayjs()
  return props.plans
    .filter((plan) => {
      // 已评估 → 已从待评课队列清除（不在主工作台展示）
      if (plan.evaluated) return false
      const date = dayjs(plan.plannedDate)
      if (range.value === 'day') return date.isSame(today, 'day')
      if (range.value === 'week') return date.isSame(today, 'week')
      if (range.value === 'month') return date.isSame(today, 'month')
      // 未评：已过授课日期（严格早于今天，不与「今日」重合）且尚未评估
      return date.isBefore(today, 'day')
    })
    .sort((a, b) => dayjs(a.plannedDate).valueOf() - dayjs(b.plannedDate).valueOf())
})

const pendingCount = computed(() => rangedPlans.value.length)

const emptyText = computed(() => {
  if (range.value === 'pending') return '暂无已过日期且未评课的记录'
  return `${RANGES.find((r) => r.key === range.value)?.label}暂无待评课安排`
})

/** 授课日期是否已到达/超过（date <= 今天）——未来课程不开放评估入口 */
function isReached(plan: SupervisionPlan): boolean {
  return !dayjs(plan.plannedDate).isAfter(dayjs(), 'day')
}

function isFuture(plan: SupervisionPlan): boolean {
  return dayjs(plan.plannedDate).isAfter(dayjs(), 'day')
}

/** 到达/超过授课日期且未评估：暴露「去评估」入口（未来课次仅预览） */
function canEvaluateEntry(plan: SupervisionPlan): boolean {
  return isReached(plan) && !plan.evaluated
}

/** 无授课记录时先创建再评估；已有记录直接进入评估页 */
function handleEvaluateClick(plan: SupervisionPlan): void {
  if (plan.sessionId === null) {
    emit('create', plan)
  } else {
    emit('evaluate', plan)
  }
}
</script>

<template>
  <div class="evaluation-queue">
    <div class="evaluation-queue__tabs">
      <button
        v-for="item in RANGES"
        :key="item.key"
        type="button"
        class="evaluation-queue__tab"
        :class="{ 'evaluation-queue__tab--active': range === item.key }"
        @click="range = item.key"
      >
        {{ item.label }}
      </button>
      <span class="evaluation-queue__summary">待评 {{ pendingCount }} 条</span>
    </div>

    <el-skeleton v-if="loading" :rows="5" animated />

    <template v-else-if="rangedPlans.length">
      <ul class="evaluation-queue__list">
        <li v-for="plan in rangedPlans" :key="plan.id" class="evaluation-queue__item">
          <span class="evaluation-queue__date">
            {{ formatDate(plan.plannedDate, 'MM-DD') }}
          </span>
          <span class="evaluation-queue__course" :title="plan.courseName">
            {{ plan.courseName }}
          </span>
          <span class="evaluation-queue__teacher">{{ plan.teacherName }}</span>
          <el-tag
            class="evaluation-queue__status"
            size="small"
            :type="isFuture(plan) ? 'info' : 'warning'"
            effect="plain"
          >
            {{ isFuture(plan) ? '未开始' : '待评估' }}
          </el-tag>

          <span class="evaluation-queue__actions">
            <el-button
              v-if="canEvaluateEntry(plan)"
              size="small"
              type="primary"
              @click="handleEvaluateClick(plan)"
            >
              去评估
            </el-button>
            <span v-else-if="isFuture(plan)" class="evaluation-queue__preview">
              未到授课日期 · 仅预览
            </span>
          </span>
        </li>
      </ul>
    </template>

    <EmptyState v-else :description="emptyText" />
  </div>
</template>

<style scoped lang="scss">
.evaluation-queue {
  width: 100%;

  &__tabs {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    margin-bottom: var(--spacing-4);
  }

  &__tab {
    padding: 4px 14px;
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
    cursor: pointer;
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: 999px;
    transition: all var(--duration-fast) ease;

    &:hover {
      color: var(--color-primary);
    }

    &--active {
      font-weight: 600;
      color: #ffffff;
      background-color: var(--color-primary);
      border-color: var(--color-primary);
    }
  }

  &__summary {
    margin-left: auto;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
  }

  &__item {
    display: flex;
    gap: var(--spacing-3);
    align-items: center;
    padding: var(--spacing-3);
    background-color: var(--color-bg-page);
    border-radius: var(--radius-md);
    transition: box-shadow var(--duration-fast) ease;

    &:hover {
      box-shadow: var(--shadow-card-hover);
    }
  }

  &__date {
    flex-shrink: 0;
    font-family: var(--font-score);
    font-size: var(--font-size-sm);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    color: var(--color-primary);
  }

  &__course {
    overflow: hidden;
    flex: 1;
    min-width: 0;
    font-size: var(--font-size-base);
    font-weight: 500;
    color: var(--color-text-primary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__teacher {
    flex-shrink: 0;
    max-width: 140px;
    overflow: hidden;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__status {
    flex-shrink: 0;
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    gap: var(--spacing-2);
    align-items: center;
    margin-left: auto;
  }

  &__preview {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }
}
</style>
