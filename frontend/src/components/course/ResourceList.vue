<script setup lang="ts">
import type { Resource } from '@/types/resource'
import { RESOURCE_TYPE_LABELS } from '@/constants'
import { formatBytes, formatDateTime } from '@/utils/format'

withDefaults(
  defineProps<{
    resources: Resource[]
    canManage?: boolean
    loading?: boolean
  }>(),
  {
    canManage: false,
    loading: false
  }
)

const emit = defineEmits<{
  (e: 'download', resource: Resource): void
  (e: 'delete', resource: Resource): void
}>()

function asResource(row: unknown): Resource {
  return row as Resource
}
</script>

<template>
  <el-table v-loading="loading" :data="resources" row-key="id" class="resource-list">
    <el-table-column prop="name" label="文件名" min-width="240">
      <template #default="{ row }">
        <div class="resource-list__name">
          <span class="resource-list__badge" :class="`resource-list__badge--${row.type}`">
            {{ RESOURCE_TYPE_LABELS[row.type as keyof typeof RESOURCE_TYPE_LABELS] }}
          </span>
          <span class="resource-list__text">{{ row.name }}</span>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="大小" width="110" align="right">
      <template #default="{ row }">
        <span class="tabular-nums">{{ formatBytes(row.size) }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="uploader" label="上传人" width="110" />
    <el-table-column label="上传时间" width="180">
      <template #default="{ row }">{{ formatDateTime(row.uploadedAt) }}</template>
    </el-table-column>
    <el-table-column label="操作" width="140" fixed="right">
      <template #default="{ row }">
        <el-button link type="primary" @click="emit('download', asResource(row))">下载</el-button>
        <el-button v-if="canManage" link type="danger" @click="emit('delete', asResource(row))">
          删除
        </el-button>
      </template>
    </el-table-column>
    <template #empty>
      <span class="resource-list__empty">暂无课程资源</span>
    </template>
  </el-table>
</template>

<style scoped lang="scss">
.resource-list {
  &__name {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
  }

  &__badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 40px;
    height: 22px;
    padding: 0 var(--spacing-1);
    font-size: var(--font-size-xs);
    color: var(--color-text-secondary);
    background-color: var(--color-divider);
    border-radius: var(--radius-sm);
    flex-shrink: 0;

    &--pdf {
      color: var(--color-error);
      background-color: color-mix(in srgb, var(--color-error) 10%, var(--color-bg-card));
    }

    &--doc {
      color: var(--color-primary);
      background-color: var(--color-primary-bg);
    }

    &--ppt {
      color: var(--color-warning);
      background-color: color-mix(in srgb, var(--color-warning) 12%, var(--color-bg-card));
    }

    &--video {
      color: var(--color-supervisor);
      background-color: var(--color-supervisor-bg);
    }

    &--zip {
      color: var(--color-teacher);
      background-color: var(--color-teacher-bg);
    }
  }

  &__text {
    color: var(--color-text-primary);
  }

  &__empty {
    color: var(--color-text-tertiary);
  }
}
</style>
