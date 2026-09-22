<script setup lang="ts">
import { computed } from 'vue'
import type { TranscriptDTO } from '@/types/evaluation'

const props = defineProps<{ transcript: TranscriptDTO | null; loading?: boolean }>()
const emit = defineEmits<{ (e: 'retry'): void }>()
const waiting = computed(() => props.transcript?.status === 'pending' || props.transcript?.status === 'running')
</script>

<template>
  <el-skeleton v-if="loading" :rows="4" animated />
  <div v-else-if="!transcript" class="transcript-viewer__empty">上传录音后将生成脱敏转写文本。</div>
  <div v-else class="transcript-viewer">
    <div v-if="waiting" class="transcript-viewer__status">
      <el-progress :percentage="transcript.status === 'running' ? 65 : 20" status="success" />
      <span>转写任务{{ transcript.status === 'running' ? '运行中' : '排队中' }}，页面会自动刷新。</span>
    </div>
    <div v-else-if="transcript.status === 'failed'" class="transcript-viewer__status">
      <el-alert type="warning" :title="transcript.errorMessage || '转写失败'" :closable="false" />
      <el-button type="primary" plain @click="emit('retry')">重试转写</el-button>
    </div>
    <template v-else>
      <div v-if="transcript.segments.length" class="transcript-viewer__segments">
        <p v-for="(segment, index) in transcript.segments" :key="index">
          <strong>{{ segment.speaker || '发言人' }}</strong> {{ segment.text }}
        </p>
      </div>
      <p v-else class="transcript-viewer__content">{{ transcript.content || '暂无转写内容' }}</p>
    </template>
  </div>
</template>

<style scoped lang="scss">
.transcript-viewer { display:flex; flex-direction:column; gap:var(--spacing-3); }
.transcript-viewer__status { display:flex; flex-direction:column; gap:var(--spacing-3); font-size:var(--font-size-sm); color:var(--color-text-tertiary); }
.transcript-viewer__segments, .transcript-viewer__content { font-size:var(--font-size-sm); line-height:1.8; color:var(--color-text-secondary); white-space:pre-wrap; }
.transcript-viewer__segments p { margin:0 0 var(--spacing-2); }
.transcript-viewer__empty { color:var(--color-text-tertiary); font-size:var(--font-size-sm); }
</style>
