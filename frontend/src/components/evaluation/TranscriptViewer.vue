<script setup lang="ts">
import { computed, ref } from 'vue'
import type { TranscriptDTO } from '@/types/evaluation'

/**
 * 脱敏转写查看器（docs/frontend_AGENTS.md §7.10）。
 * 三态：排队/运行中 → 波形呼吸；失败 → 重试；完成 → 说话人分色气泡（脱敏文案由后端保证）。
 *
 * 卡片长度控制：正文默认只渲染前 PREVIEW_SEGMENTS 条（纯文本按 PREVIEW_CHARS 截断）并限高裁切，
 * 超出时显示「展开全文」；点击后在居中悬浮弹窗（正文可滚动）查看完整转写。
 */
const props = defineProps<{ transcript: TranscriptDTO | null; loading?: boolean }>()
const emit = defineEmits<{ (e: 'retry'): void }>()

const expanded = ref(false)

const waiting = computed(
  () => props.transcript?.status === 'pending' || props.transcript?.status === 'running'
)

/** 说话人分色：教师 = 主色、学生 = 中性青灰；文案已由后端脱敏（「教师 / 学生X」） */
function speakerKind(speaker: string): 'teacher' | 'student' | 'other' {
  if (speaker.includes('教师') || speaker.includes('老师')) return 'teacher'
  if (speaker.includes('学生')) return 'student'
  return 'other'
}

/** 秒 → mm:ss（用于气泡时间戳，便于按时间回看课堂） */
function formatOffset(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '00:00'
  const total = Math.floor(seconds)
  const minutes = Math.floor(total / 60)
  const rest = total % 60
  return `${String(minutes).padStart(2, '0')}:${String(rest).padStart(2, '0')}`
}

/** 预览条数 / 字符数阈值：超过则提供「展开全文」入口 */
const PREVIEW_SEGMENTS = 3
const PREVIEW_CHARS = 220

const hasSegments = computed(() => (props.transcript?.segments.length ?? 0) > 0)
const hasMore = computed(() => {
  const transcript = props.transcript
  if (!transcript) return false
  if (transcript.segments.length > PREVIEW_SEGMENTS) return true
  return transcript.segments.length === 0 && transcript.content.length > PREVIEW_CHARS
})

/** 卡片内只渲染前 PREVIEW_SEGMENTS 条，完整内容留给弹窗 */
const previewSegments = computed(() => (props.transcript?.segments ?? []).slice(0, PREVIEW_SEGMENTS))

/** 无分段时的纯文本预览（按字符截断） */
const previewContent = computed(() => {
  const content = props.transcript?.content ?? ''
  if (content.length <= PREVIEW_CHARS) return content
  return `${content.slice(0, PREVIEW_CHARS)}…`
})

const expandLabel = computed(() =>
  hasSegments.value ? `展开全文（共 ${props.transcript?.segments.length ?? 0} 条）` : '展开全文'
)

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
      <!-- 卡片内只露预览高度，超出部分裁切渐隐 -->
      <div class="transcript-viewer__preview" :class="{ 'transcript-viewer__preview--clipped': hasMore }">
        <div v-if="hasSegments" class="transcript-viewer__segments">
          <p
            v-for="(segment, index) in previewSegments"
            :key="index"
            class="transcript-viewer__bubble"
            :class="`transcript-viewer__bubble--${speakerKind(segment.speaker || '')}`"
          >
            <span class="transcript-viewer__speaker">
              {{ segment.speaker || '发言人' }}
              <em class="transcript-viewer__time tabular-nums">{{ formatOffset(segment.start) }}</em>
            </span>
            <span class="transcript-viewer__text">{{ segment.text }}</span>
          </p>
        </div>
        <p v-else class="transcript-viewer__content">{{ previewContent || '暂无转写内容' }}</p>
      </div>

      <div v-if="hasMore" class="transcript-viewer__more">
        <el-button type="primary" plain size="small" @click="expanded = true">
          {{ expandLabel }}
        </el-button>
      </div>

      <p class="transcript-viewer__privacy">
        转写已自动脱敏，学生一律以「学生X」显示；教师不可回放录音。
      </p>
    </template>

    <!-- 居中悬浮弹窗：正文可滚动查看完整转写 -->
    <el-dialog
      v-model="expanded"
      title="课堂转写全文"
      width="760px"
      align-center
      append-to-body
      class="transcript-viewer__dialog"
    >
      <div class="transcript-viewer__dialog-body">
        <div v-if="hasSegments" class="transcript-viewer__segments">
          <p
            v-for="(segment, index) in transcript.segments"
            :key="`dialog-${index}`"
            class="transcript-viewer__bubble"
            :class="`transcript-viewer__bubble--${speakerKind(segment.speaker || '')}`"
          >
            <span class="transcript-viewer__speaker">
              {{ segment.speaker || '发言人' }}
              <em class="transcript-viewer__time tabular-nums">{{ formatOffset(segment.start) }}</em>
            </span>
            <span class="transcript-viewer__text">{{ segment.text }}</span>
          </p>
        </div>
        <p v-else class="transcript-viewer__content">{{ transcript.content || '暂无转写内容' }}</p>
      </div>
      <p class="transcript-viewer__privacy">
        转写已自动脱敏，学生一律以「学生X」显示；教师不可回放录音。
      </p>
    </el-dialog>
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

  /* 预览区：限制卡片长度（约 4 条气泡），超出裁切并渐隐 */
  &__preview {
    position: relative;

    &--clipped {
      max-height: 260px;
      overflow: hidden;

      &::after {
        position: absolute;
        right: 0;
        bottom: 0;
        left: 0;
        height: 56px;
        content: '';
        background: linear-gradient(180deg, transparent, var(--color-bg-card));
        pointer-events: none;
      }
    }
  }

  &__more {
    display: flex;
    justify-content: center;
  }

  &__dialog-body {
    max-height: 60vh;
    padding-right: var(--spacing-2);
    overflow-y: auto;
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
    display: flex;
    gap: var(--spacing-2);
    align-items: baseline;
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--color-text-secondary);
  }

  &__time {
    font-size: var(--font-size-xs);
    font-style: normal;
    font-weight: 400;
    color: var(--color-text-tertiary);
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
