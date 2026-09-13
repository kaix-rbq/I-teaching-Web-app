<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string
    subtitle?: string
  }>(),
  {
    title: '',
    subtitle: ''
  }
)
</script>

<template>
  <div class="page-header">
    <div class="page-header__main">
      <h1 class="page-header__title">
        <slot name="title">{{ title }}</slot>
      </h1>
      <p v-if="subtitle" class="page-header__subtitle">{{ subtitle }}</p>
    </div>
    <div class="page-header__actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-4);
  margin-bottom: var(--spacing-6);

  &__title {
    font-size: var(--font-size-3xl);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__subtitle {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-base);
    color: var(--color-text-tertiary);
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    gap: var(--spacing-2);
  }
}
</style>
