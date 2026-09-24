<script setup lang="ts">
import { computed } from 'vue'
import type { TranscriptDTO } from '@/types/evaluation'

/**
 * 脱敏转写查看器（《前端设计-new》§5.5-4）。
 * 三态：排队/运行中 → 波形呼吸；失败 → 重试；完成 → 说话人分色气泡（脱敏文案由后端保证）。
 */
const props = defineProps<{ transcript: TranscriptDTO | null; loading?: boolean }>()
const emit = defineEmits<{ (e: 'retry'): void }>()

const waiting = computed(
  () => props.transcript?.status === 'pending' || props.transcript?.status === 'running'
)

/** 说话人分色：教师 = 主色、学生 = 中性青灰；文案已由后端脱敏（「教师 / 学生X」） */
function speakerKind(speaker: string): 'teacher' | 'student' | 'other' {
  if (speaker.includes('教师') || speaker.includes('老师')) return 'teacher'
  if (speaker.includes('学生')) return 'student'
  return 'other'
}

const WAVE_BARS = [10, 18, 26, 18, 10, 22, 14]
</script>

<template>
  <el-skeleton v-if="loading" :rows="4" animated />
  <div v-else-if="!transcript" class="transcript-viewer__empty">
    上传录音后将生成脱敏转写文本。
  </div>
  <div v-else class="transcript-viewer">
    <div v-if="waiting" class="transcript-viewer__status">
      <span class="transcript-viewer__wave" aria-hidden="true">
        <i
          v-for="(height, index) in WAVE_BARS"
          :key="index"
          class="transcript-viewer__wave-bar anim-breathe"
          :style="{ height: `${height}px`, animationDelay: `${index * 0.12}s` }"
        />
      </span>
      <span>
        转写任务{{ transcript.status === 'running' ? '运行中' : '排队中' }}，完成后自动刷新。
      </span>
    </div>

    <div v-else-if="transcript.status === 'failed'" class="transcript-viewer__status">
      <el-alert type="warning" :title="transcript.errorMessage || '转写失败'" :closable="false" />
      <el-button type="primary" plain @click="emit('retry')">重试转写</el-button>
    </div>

    <template v-else>
      <div v-if="transcript.segments.length" class="transcript-viewer__segments">
        <p
          v-for="(segment, index) in transcript.segments"
          :key="index"
          class="transcript-viewer__bubble"
          :class="`transcript-viewer__bubble--${speakerKind(segment.speaker || '')}`"
        >
          <span class="transcript-viewer__speaker">{{ segment.speaker || '发言人' }}</span>
          <span class="transcript-viewer__text">{{ segment.text }}</span>
        </p>
      </div>
      <p v-else class="transcript-viewer__content">{{ transcript.content || '暂无转写内容' }}</p>
      <p class="transcript-viewer__privacy">
        转写已自动脱敏，学生一律以「学生X」显示；教师不可回放录音。
      </p>
    </template>
  </div>
</template>

<style scoped lang="scss">
.transcript-viewer {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);

  &__status {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
    align-items: flex-start;
    padding: var(--spacing-3);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
    background-color: var(--color-bg-page);
    border-radius: var(--radius-md);
  }

  &__wave {
    display: flex;
    gap: 4px;
    align-items: center;
    height: 30px;

    &-bar {
      width: 4px;
      background: var(--gradient-ai);
      border-radius: 2px;
    }
  }

  &__segments {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
  }

  &__bubble {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: var(--spacing-2) var(--spacing-3);
    background-color: var(--color-bg-page);
    border-left: 3px solid var(--color-divider);
    border-radius: var(--radius-md);

    &--teacher {
      border-left-color: var(--color-primary);
    }

    &--student {
      border-left-color: var(--color-teacher);
    }

    &--other {
      border-left-color: var(--color-score-void);
    }
  }

  &__speaker {
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--color-text-secondary);
  }

  &__text,
  &__content {
    font-size: var(--font-size-sm);
    line-height: 1.8;
    color: var(--color-text-secondary);
    white-space: pre-wrap;
  }

  &__content {
    margin: 0;
  }

  &__privacy {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__empty {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
