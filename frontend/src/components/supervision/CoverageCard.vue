<script setup lang="ts">
import { computed } from 'vue'
import type { CoverageStat } from '@/types/supervision'
import { formatRate } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    overall: CoverageStat | null
    loading?: boolean
  }>(),
  {
    loading: false
  }
)

const RADIUS = 52
const CIRCUMFERENCE = 2 * Math.PI * RADIUS

const rate = computed(() => props.overall?.rate ?? 0)
const dashOffset = computed(() => CIRCUMFERENCE * (1 - rate.value))
const circumference = computed(() => CIRCUMFERENCE)
</script>

<template>
  <div class="coverage-card">
    <el-skeleton v-if="loading" :rows="5" animated />

    <template v-else>
      <div class="coverage-card__ring">
        <svg viewBox="0 0 120 120" class="coverage-card__svg">
          <circle
            cx="60"
            cy="60"
            :r="RADIUS"
            fill="none"
            stroke="var(--color-divider)"
            stroke-width="10"
          />
          <circle
            cx="60"
            cy="60"
            :r="RADIUS"
            fill="none"
            stroke="var(--color-primary)"
            stroke-width="10"
            stroke-linecap="round"
            :stroke-dasharray="circumference"
            :stroke-dashoffset="dashOffset"
            transform="rotate(-90 60 60)"
          />
        </svg>
        <div class="coverage-card__ring-text">
          <span class="coverage-card__rate">{{ formatRate(rate) }}</span>
          <span class="coverage-card__rate-label">督导覆盖率</span>
        </div>
      </div>

      <div class="coverage-card__summary">
        <p>
          已完成听评课
          <strong>{{ overall?.supervisedCourses ?? 0 }}</strong>
          / {{ overall?.totalCourses ?? 0 }} 门
        </p>
      </div>

      <div class="coverage-card__departments">
        <h4 class="coverage-card__subtitle">按教研室覆盖率</h4>
        <div
          v-for="item in overall?.byDepartment ?? []"
          :key="item.department"
          class="coverage-card__dept"
        >
          <div class="coverage-card__dept-head">
            <span>{{ item.department }}</span>
            <span class="tabular-nums">{{ formatRate(item.rate) }}</span>
          </div>
          <el-progress
            :percentage="Math.round(item.rate * 100)"
            :show-text="false"
            :stroke-width="8"
          />
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.coverage-card {
  padding: var(--spacing-6);
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);

  &__ring {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 160px;
    height: 160px;
    margin: 0 auto;
  }

  &__svg {
    width: 160px;
    height: 160px;
  }

  &__ring-text {
    position: absolute;
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  &__rate {
    font-size: var(--font-size-stat);
    font-weight: 600;
    color: var(--color-text-primary);
    font-variant-numeric: tabular-nums;
  }

  &__rate-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__summary {
    margin-top: var(--spacing-3);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
    text-align: center;

    strong {
      font-size: var(--font-size-lg);
      color: var(--color-primary);
    }
  }

  &__departments {
    padding-top: var(--spacing-4);
    margin-top: var(--spacing-4);
    border-top: 1px solid var(--color-divider);
  }

  &__subtitle {
    margin-bottom: var(--spacing-3);
    font-size: var(--font-size-base);
    color: var(--color-text-primary);
  }

  &__dept {
    margin-bottom: var(--spacing-3);

    &:last-child {
      margin-bottom: 0;
    }
  }

  &__dept-head {
    display: flex;
    justify-content: space-between;
    margin-bottom: var(--spacing-1);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }
}
</style>
