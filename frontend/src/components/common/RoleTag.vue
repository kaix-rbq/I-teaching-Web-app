<script setup lang="ts">
import { computed } from 'vue'
import { getRoleMeta } from '@/constants'
import type { Role } from '@/types/user'

const props = defineProps<{ role?: Role }>()

const meta = computed(() => getRoleMeta(props.role))
const modifier = computed(() => (props.role ? `role-tag--${props.role}` : ''))
</script>

<template>
  <span v-if="meta" class="role-tag" :class="modifier">{{ meta.label }}</span>
</template>

<style scoped lang="scss">
.role-tag {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 var(--spacing-2);
  font-size: var(--font-size-xs);
  line-height: 1;
  border-radius: var(--radius-sm);

  &--director {
    color: var(--color-director);
    background-color: var(--color-director-bg);
  }

  &--teacher {
    color: var(--color-teacher);
    background-color: var(--color-teacher-bg);
  }

  &--supervisor {
    color: var(--color-supervisor);
    background-color: var(--color-supervisor-bg);
  }
}
</style>
