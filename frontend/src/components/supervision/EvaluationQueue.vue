<script setup lang="ts">
import { computed } from 'vue'
import dayjs from 'dayjs'
import type { SupervisionPlan } from '@/types/supervision'
import { formatDate } from '@/utils/format'
import EmptyState from '@/components/common/EmptyState.vue'

/**
 * 督导待评课队列（《前端设计-new》§5.3 / §5.4）——「下一步该评哪节课」无需思考。
 * 按今日 / 本周 / 更早 / 已完成分组。
 * 已知数据缺口：supervision_plans 与 teaching_sessions 无外键关联，
 * 计划行无法直达评估页，故主动作为「创建授课记录并评估」（父级实现 POST /sessions 流）。
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
  'view-course': [plan: SupervisionPlan]
}>()

interface QueueGroup {
  key: string
  title: string
  items: SupervisionPlan[]
}

const groups = computed<QueueGroup[]>(() => {
  const today = dayjs()
  const pending = props.plans.filter((item) => item.status === 'planned')
  const done = props.plans.filter((item) => item.status === 'completed')

  const ofToday = pending.filter((item) => dayjs(item.plannedDate).isSame(today, 'day'))
  const rest = pending.filter((item) => !dayjs(item.plannedDate).isSame(today, 'day'))
  const ofThisWeek = rest.filter((item) => dayjs(item.plannedDate).isSame(today, 'week'))
  const earlier = rest.filter((item) => !dayjs(item.plannedDate).isSame(today, 'week'))

  const sortByDate = (list: SupervisionPlan[]): SupervisionPlan[] =>
    [...list].sort((a, b) => dayjs(a.plannedDate).valueOf() - dayjs(b.plannedDate).valueOf())

  return [
    { key: 'today', title: '今日待评', items: sortByDate(ofToday) },
    { key: 'week', title: '本周待评', items: sortByDate(ofThisWeek) },
    { key: 'earlier', title: '更早计划', items: sortByDate(earlier) },
    { key: 'done', title: '已完成', items: sortByDate(done).reverse() }
  ].filter((group) => group.items.length > 0)
})

const pendingCount = computed(() => props.plans.filter((item) => item.status === 'planned').length)
</script>

<template>
  <div class="evaluation-queue">
    <el-skeleton v-if="loading" :rows="5" animated />

    <template v-else-if="groups.length">
      <p class="evaluation-queue__summary">
        共 {{ pendingCount }} 条待评计划 · 下一步动作已按日期排好
      </p>

      <section v-for="group in groups" :key="group.key" class="evaluation-queue__group">
        <h4 class="evaluation-queue__group-title">
          {{ group.title }}
          <span class="evaluation-queue__count">{{ group.items.length }}</span>
        </h4>

        <ul class="evaluation-queue__list">
          <li v-for="plan in group.items" :key="plan.id" class="evaluation-queue__item">
            <span class="evaluation-queue__date">
              {{ formatDate(plan.plannedDate, 'MM-DD') }}
            </span>
            <span class="evaluation-queue__course" :title="plan.courseName">
              {{ plan.courseName }}
            </span>
            <span class="evaluation-queue__teacher">{{ plan.teacherName }}</span>

            <span class="evaluation-queue__actions">
              <el-button
                v-if="plan.status === 'planned'"
                size="small"
                type="primary"
                plain
                @click="emit('create', plan)"
              >
                创建授课记录并评估
              </el-button>
              <el-button v-else size="small" text type="primary" @click="emit('view-course', plan)">
                查看授课记录
              </el-button>
            </span>
          </li>
        </ul>
      </section>
    </template>

    <EmptyState v-else description="本学期暂无听课计划，可前往课堂评估页排期" />
  </div>
</template>

<style scoped lang="scss">
.evaluation-queue {
  width: 100%;

  &__summary {
    margin-bottom: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__group {
    &-title {
      display: flex;
      gap: var(--spacing-2);
      align-items: center;
      margin: var(--spacing-4) 0 var(--spacing-2);
      font-size: var(--font-size-sm);
      color: var(--color-text-secondary);
    }
  }

  &__count {
    padding: 0 6px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    background-color: var(--color-bg-page);
    border-radius: 999px;
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
    padding: var(--spacing-2) var(--spacing-3);
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
    max-width: 160px;
    overflow: hidden;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    gap: var(--spacing-2);
    margin-left: auto;
  }
}
</style>
