<script setup lang="ts">
withDefaults(
  defineProps<{
    description?: string
  }>(),
  {
    description: '暂无数据'
  }
)
</script>

<template>
  <div class="empty-state">
    <svg
      class="empty-state__icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.4"
    >
      <path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.7-.9l-.8-1.2A2 2 0 0 0 7.9 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2z" />
    </svg>
    <p class="empty-state__description">{{ description }}</p>
    <div class="empty-state__action">
      <slot name="action" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-8) var(--spacing-6);
  text-align: center;

  &__icon {
    width: 56px;
    height: 56px;
    color: var(--color-text-disabled);
  }

  &__description {
    margin-top: var(--spacing-3);
    font-size: var(--font-size-base);
    color: var(--color-text-tertiary);
  }

  &__action {
    margin-top: var(--spacing-4);
  }
}
</style>
