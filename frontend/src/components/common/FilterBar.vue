<script setup lang="ts">
defineEmits<{
  (e: 'search'): void
  (e: 'reset'): void
}>()
</script>

<template>
  <div class="filter-bar">
    <div class="filter-bar__fields">
      <slot />
    </div>
    <div class="filter-bar__actions">
      <slot name="actions" />
      <el-button type="primary" @click="$emit('search')">查询</el-button>
      <el-button @click="$emit('reset')">重置</el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.filter-bar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-4);
  padding: var(--spacing-4) var(--spacing-6);
  margin-bottom: var(--spacing-4);
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);

  &__fields {
    display: flex;
    flex: 1;
    flex-wrap: wrap;
    gap: var(--spacing-3);
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    gap: var(--spacing-2);
  }
}
</style>
