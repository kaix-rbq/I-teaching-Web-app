<script setup lang="ts">
import { computed } from 'vue'

type Tone = 'primary' | 'success' | 'warning' | 'info' | 'supervisor'

const props = withDefaults(
  defineProps<{
    label: string
    value: string | number
    unit?: string
    icon?: string
    tone?: Tone
  }>(),
  {
    unit: '',
    icon: 'course',
    tone: 'primary'
  }
)

const ICONS: Record<string, string> = {
  course:
    'M4 19.5A2.5 2.5 0 0 1 6.5 17H20M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z',
  teacher:
    'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 7a4 4 0 1 0 0 .01M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75',
  class: 'm12 2 9 5-9 5-9-5 9-5zM3 12l9 5 9-5M3 17l9 5 9-5',
  student: 'M22 10 12 5 2 10l10 5 10-5zM6 12v5c3 3 9 3 12 0v-5',
  resource:
    'M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.7-.9l-.8-1.2A2 2 0 0 0 7.9 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2z',
  plan: 'M3 4h18a1 1 0 0 1 1 1v15a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1zM16 2v4M8 2v4M2 10h20',
  done: 'M22 11.1V12a10 10 0 1 1-5.9-9.1M9 11l3 3L22 4',
  coverage: 'M12 2a10 10 0 1 1 0 20 10 10 0 0 1 0-20zM12 2v10h10'
}

const path = computed(() => ICONS[props.icon] ?? ICONS.course)
</script>

<template>
  <div class="stat-card">
    <div class="stat-card__icon" :class="`stat-card__icon--${tone}`">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <path :d="path" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </div>
    <div class="stat-card__body">
      <div class="stat-card__value">
        {{ value }}<span v-if="unit" class="stat-card__unit">{{ unit }}</span>
      </div>
      <div class="stat-card__label">{{ label }}</div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.stat-card {
  display: flex;
  align-items: center;
  gap: var(--spacing-4);
  padding: var(--spacing-6);
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  transition: box-shadow 0.2s ease, transform 0.2s ease;

  &:hover {
    box-shadow: var(--shadow-card-hover);
    transform: translateY(-1px);
  }

  &__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    border-radius: var(--radius-lg);

    svg {
      width: 24px;
      height: 24px;
    }

    &--primary {
      color: var(--color-primary);
      background-color: color-mix(in srgb, var(--color-primary) 12%, var(--color-bg-card));
    }

    &--info {
      color: var(--color-info);
      background-color: color-mix(in srgb, var(--color-info) 12%, var(--color-bg-card));
    }

    &--success {
      color: var(--color-success);
      background-color: color-mix(in srgb, var(--color-success) 12%, var(--color-bg-card));
    }

    &--warning {
      color: var(--color-warning);
      background-color: color-mix(in srgb, var(--color-warning) 14%, var(--color-bg-card));
    }

    &--supervisor {
      color: var(--color-supervisor);
      background-color: color-mix(in srgb, var(--color-supervisor) 12%, var(--color-bg-card));
    }
  }

  &__value {
    display: flex;
    align-items: baseline;
    gap: 2px;
    font-size: var(--font-size-stat);
    font-weight: 600;
    line-height: 1.2;
    color: var(--color-text-primary);
    font-variant-numeric: tabular-nums;
  }

  &__unit {
    font-size: var(--font-size-base);
    font-weight: 400;
    color: var(--color-text-tertiary);
  }

  &__label {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
