<script setup lang="ts">
import { RESOURCE_ACCEPT, MAX_RESOURCE_SIZE } from '@/constants'
import { formatBytes } from '@/utils/format'
import type { UploadRequestOptions } from 'element-plus'

const props = withDefaults(
  defineProps<{
    accept?: string
    maxSize?: number
    uploading?: boolean
    progress?: number
  }>(),
  {
    accept: RESOURCE_ACCEPT,
    maxSize: MAX_RESOURCE_SIZE,
    uploading: false,
    progress: 0
  }
)

const emit = defineEmits<{
  (e: 'file', file: File): void
}>()

function isAccepted(file: File): boolean {
  const acceptList = props.accept.split(',').map((item) => item.trim().toLowerCase())
  const ext = `.${file.name.slice(file.name.lastIndexOf('.') + 1).toLowerCase()}`
  return acceptList.includes(ext)
}

function handleBeforeUpload(file: File): boolean {
  if (!isAccepted(file)) {
    ElMessage.error(`仅支持 ${props.accept} 格式的文件`)
    return false
  }
  if (file.size > props.maxSize) {
    ElMessage.error(`单个文件不得超过 ${formatBytes(props.maxSize)}`)
    return false
  }
  return true
}

async function handleRequest(options: UploadRequestOptions): Promise<void> {
  emit('file', options.file)
  options.onSuccess?.(undefined)
}
</script>

<template>
  <div class="resource-uploader">
    <el-upload
      class="resource-uploader__drop"
      drag
      :accept="accept"
      :show-file-list="false"
      :multiple="false"
      :before-upload="handleBeforeUpload"
      :http-request="handleRequest"
      :disabled="uploading"
    >
      <div class="resource-uploader__inner">
        <svg
          class="resource-uploader__icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
        >
          <path d="M12 16V4M7 9l5-5 5 5M20 16v4H4v-4" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        <p class="resource-uploader__hint">将文件拖拽到此处，或<em>点击上传</em></p>
        <p class="resource-uploader__tip">
          支持 {{ accept }}，单个文件不超过 {{ formatBytes(maxSize) }}
        </p>
      </div>
    </el-upload>

    <div v-if="uploading" class="resource-uploader__progress">
      <el-progress :percentage="progress" :stroke-width="8" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.resource-uploader {
  &__drop {
    :deep(.el-upload-dragger) {
      padding: var(--spacing-6);
      background-color: var(--color-bg-page);
      border: 1px dashed var(--color-border);
      border-radius: var(--radius-lg);

      &:hover {
        border-color: var(--color-primary);
      }
    }
  }

  &__inner {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  &__icon {
    width: 40px;
    height: 40px;
    color: var(--color-primary);
  }

  &__hint {
    margin-top: var(--spacing-2);
    font-size: var(--font-size-base);
    color: var(--color-text-secondary);

    em {
      font-style: normal;
      color: var(--color-primary);
    }
  }

  &__tip {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__progress {
    margin-top: var(--spacing-3);
  }
}
</style>
