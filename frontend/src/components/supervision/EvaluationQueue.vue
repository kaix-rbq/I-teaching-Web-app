<script setup lang="ts">
import { computed } from 'vue'
import dayjs from 'dayjs'
import type { SupervisionPlan } from '@/types/supervision'

/**
 * 督导待评安排台账行。
 * 只负责把一组听评课安排排成时间轨行；今日、未来、补录的分组由工作台决定。
 * 未来课次不开放评估入口，已评估课次不会出现在传入的 plans 中。
 */
const props = withDefaults(
  defineProps<{
    plans: SupervisionPlan[]
    loading?: boolean
    emptyText?: string
  }>(),
  { loading: false, emptyText: '暂无待评安排' }
)

const emit = defineEmits<{
  create: [plan: SupervisionPlan]
  evaluate: [plan: SupervisionPlan]
}>()

const WEEK_LABELS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

const sortedPlans = computed(() =>
  props.plans
    .filter((plan) => !plan.evaluated)
    .slice()
    .sort((left, right) => dayjs(left.plannedDate).valueOf() - dayjs(right.plannedDate).valueOf())
)

function isFuture(plan: SupervisionPlan): boolean {
  return dayjs(plan.plannedDate).isAfter(dayjs(), 'day')
}

function isPast(plan: SupervisionPlan): boolean {
  return dayjs(plan.plannedDate).isBefore(dayjs(), 'day')
}

function dateText(value: string): string {
  const date = dayjs(value)
  return `${date.format('MM-DD')} ${WEEK_LABELS[date.day()]}`
}

function statusText(plan: SupervisionPlan): string {
  if (isFuture(plan)) return '未开始'
  if (isPast(plan)) return '待补录'
  return '待评估'
}

function handleEvaluateClick(plan: SupervisionPlan): void {
  if (isFuture(plan)) return
  if (plan.sessionId === null) {
    emit('create', plan)
  } else {
    emit('evaluate', plan)
  }
}
</script>

<template>
  <div class="evaluation-queue">
    <el-skeleton v-if="loading" :rows="4" animated />

    <template v-else-if="sortedPlans.length">
      <ul class="evaluation-queue__list">
        <li v-for="plan in sortedPlans" :key="plan.id" class="evaluation-queue__item">
          <span class="evaluation-queue__date">{{ dateText(plan.plannedDate) }}</span>
          <span class="evaluation-queue__course" :title="plan.courseName">
            {{ plan.courseName }}
          </span>
          <span class="evaluation-queue__teacher">{{ plan.teacherName }}</span>
          <span class="evaluation-queue__status" :data-state="statusText(plan)">
            <i class="evaluation-queue__dot" aria-hidden="true" />
            {{ statusText(plan) }}
          </span>

          <span v-if="isFuture(plan)" class="evaluation-queue__wait" title="授课日期未到，评估入口在当天开放">
            当天开放
          </span>
          <button
            v-else
            type="button"
            class="evaluation-queue__link"
            @click="handleEvaluateClick(plan)"
          >
            去评估
          </button>
        </li>
      </ul>
    </template>

    <p v-else class="evaluation-queue__empty">{{ emptyText }}</p>
  </div>
</template>

<style scoped lang="scss">
.evaluation-queue {
  width: 100%;

  &__empty {
    padding: var(--spacing-4) 0;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__list {
    display: flex;
    flex-direction: column;
  }

  &__item {
    display: grid;
    grid-template-columns: 92px minmax(0, 1fr) 88px 80px 64px;
    gap: var(--spacing-3);
    align-items: baseline;
    padding: var(--spacing-3) 0;
    border-top: 1px solid var(--color-divider);

    &:first-child {
      border-top: none;
    }
  }

  &__date {
    font-family: var(--font-score);
    font-size: var(--font-size-xs);
    font-variant-numeric: tabular-nums;
    color: var(--color-text-tertiary);
  }

  &__course {
    overflow: hidden;
    font-size: var(--font-size-base);
    font-weight: 500;
    color: var(--color-text-primary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__teacher {
    overflow: hidden;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__status {
    display: flex;
    gap: var(--spacing-1);
    align-items: center;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);

    &[data-state='待评估'],
    &[data-state='待补录'] {
      color: var(--color-warning);
    }
  }

  &__dot {
    width: 6px;
    height: 6px;
    background-color: var(--color-text-disabled);
    border-radius: 50%;

    [data-state='待评估'] &,
    [data-state='待补录'] & {
      background-color: var(--color-warning);
    }
  }

  &__wait {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    text-align: right;
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
  }
}

@media (max-width: 1080px) {
  .evaluation-queue__item {
    grid-template-columns: 84px minmax(0, 1fr) 72px 64px;

    .evaluation-queue__status {
      display: none;
    }
  }
}

@media (max-width: 840px) {
  .evaluation-queue__item {
    grid-template-columns: 76px minmax(0, 1fr) 56px;

    .evaluation-queue__teacher {
      display: none;
    }
  }
}
</style>
