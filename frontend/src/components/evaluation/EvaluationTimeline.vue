<script setup lang="ts">
import AiBadge from '@/components/common/AiBadge.vue'
import { formatDate, scoreTone, scoreToneColor } from '@/utils/format'
import type { EvaluationDTO } from '@/types/evaluation'
import type { TeacherTimelineItem } from '@/types/teacher'

/**
 * 竖线评价流（教师画像 / 我的质量档案共用，《前端设计-new》§5.6/§5.7）。
 * 竖线 + 课次徽章 + 分数色阶节点；督导评语分色缩进；AI 评价以 AI 语法显形。
 */
withDefaults(
  defineProps<{
    items: TeacherTimelineItem[]
    loading?: boolean
    /** 空态文案 */
    emptyText?: string
  }>(),
  {
    loading: false,
    emptyText: '本学期暂无评价记录'
  }
)

const LOW_CONFIDENCE = 0.4

/** 数字一致性铁律：直接展示后端值，不做前端四舍五入 */
function formatScore(score: number | null | undefined): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

function lowConfidence(evaluation: EvaluationDTO): boolean {
  return evaluation.aiConfidence !== null && evaluation.aiConfidence < LOW_CONFIDENCE
}

function confidencePercent(evaluation: EvaluationDTO): number {
  return Math.round((evaluation.aiConfidence ?? 0) * 100)
}
</script>

<template>
  <el-skeleton v-if="loading" :rows="4" animated />
  <template v-else>
    <div v-if="items.length" class="evaluation-timeline">
      <article v-for="item in items" :key="item.sessionId" class="evaluation-timeline__item">
        <span
          class="evaluation-timeline__node"
          :style="{ backgroundColor: scoreToneColor(scoreTone(item.compositeScore)) }"
        />
        <div class="evaluation-timeline__main">
          <header class="evaluation-timeline__head">
            <span class="evaluation-timeline__date">{{ formatDate(item.sessionDate) }}</span>
            <el-tag size="small" effect="plain" round>{{ item.courseName }}</el-tag>
            <span class="evaluation-timeline__meta">
              {{ item.period || '—' }}
              <template v-if="item.topic"> · {{ item.topic }}</template>
            </span>
            <span
              class="evaluation-timeline__score score-num"
              :style="{ color: scoreToneColor(scoreTone(item.compositeScore)) }"
            >
              {{ formatScore(item.compositeScore) }}
            </span>
          </header>

          <div
            v-for="evaluation in item.supervisorEvaluations"
            :key="`${evaluation.evaluatorType}-${evaluation.evaluatorId}`"
            class="evaluation-timeline__comment"
          >
            <div class="evaluation-timeline__who">
              {{ evaluation.evaluatorName || '督导' }}
              <span class="evaluation-timeline__time">{{ formatDate(evaluation.updatedAt) }}</span>
            </div>
            <dl v-if="evaluation.highlights" class="evaluation-timeline__row">
              <dt class="evaluation-timeline__label evaluation-timeline__label--highlight">亮点</dt>
              <dd>{{ evaluation.highlights }}</dd>
            </dl>
            <dl v-if="evaluation.improvements" class="evaluation-timeline__row">
              <dt class="evaluation-timeline__label evaluation-timeline__label--improve">待改进</dt>
              <dd>{{ evaluation.improvements }}</dd>
            </dl>
            <dl v-if="evaluation.suggestions" class="evaluation-timeline__row">
              <dt class="evaluation-timeline__label evaluation-timeline__label--suggest">建议</dt>
              <dd>{{ evaluation.suggestions }}</dd>
            </dl>
            <dl v-if="evaluation.comment" class="evaluation-timeline__row">
              <dt class="evaluation-timeline__label">总体</dt>
              <dd>{{ evaluation.comment }}</dd>
            </dl>
          </div>

          <!-- AI 评价：AI 琥珀语法显形（低置信标警示，不用 0 分冒充） -->
          <div
            v-if="item.agentEvaluation"
            class="evaluation-timeline__comment evaluation-timeline__comment--ai"
          >
            <div class="evaluation-timeline__who">
              <AiBadge text="AI 评价" />
              <el-tag
                v-if="lowConfidence(item.agentEvaluation)"
                type="warning"
                size="small"
                effect="light"
                round
              >
                低置信 {{ confidencePercent(item.agentEvaluation) }}%
              </el-tag>
              <el-tag v-else type="info" size="small" effect="plain" round>
                置信 {{ confidencePercent(item.agentEvaluation) }}%
              </el-tag>
            </div>
            <dl v-if="item.agentEvaluation.highlights" class="evaluation-timeline__row">
              <dt class="evaluation-timeline__label evaluation-timeline__label--highlight">亮点</dt>
              <dd>{{ item.agentEvaluation.highlights }}</dd>
            </dl>
            <dl v-if="item.agentEvaluation.improvements" class="evaluation-timeline__row">
              <dt class="evaluation-timeline__label evaluation-timeline__label--improve">待改进</dt>
              <dd>{{ item.agentEvaluation.improvements }}</dd>
            </dl>
            <dl v-if="item.agentEvaluation.suggestions" class="evaluation-timeline__row">
              <dt class="evaluation-timeline__label evaluation-timeline__label--suggest">建议</dt>
              <dd>{{ item.agentEvaluation.suggestions }}</dd>
            </dl>
          </div>

          <p
            v-if="item.supervisorEvaluations.length === 0 && !item.agentEvaluation"
            class="evaluation-timeline__muted"
          >
            本次课暂无评价内容
          </p>
        </div>
      </article>
    </div>
    <p v-else class="evaluation-timeline__muted">{{ emptyText }}</p>
  </template>
</template>

<style scoped lang="scss">
.evaluation-timeline {
  position: relative;
  padding-left: var(--spacing-5);

  &::before {
    position: absolute;
    top: 6px;
    bottom: 6px;
    left: 5px;
    width: 2px;
    content: '';
    background-color: var(--color-divider);
    border-radius: var(--radius-sm);
  }

  &__item {
    position: relative;
    display: flex;
    padding-bottom: var(--spacing-4);
    margin-bottom: var(--spacing-3);

    &:last-child {
      padding-bottom: 0;
      margin-bottom: 0;
    }
  }

  &__node {
    position: absolute;
    top: 6px;
    left: calc(-1 * var(--spacing-5) + 1px);
    width: 10px;
    height: 10px;
    border: 2px solid var(--color-bg-card);
    border-radius: 50%;
  }

  &__main {
    flex: 1;
    padding: var(--spacing-3) var(--spacing-4);
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
  }

  &__head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
    margin-bottom: var(--spacing-2);
  }

  &__date {
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__meta {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__score {
    margin-left: auto;
    font-size: var(--font-size-lg);
    font-weight: 600;
  }

  &__comment {
    padding-top: var(--spacing-2);
    margin-top: var(--spacing-2);
    border-top: 1px dashed var(--color-divider);

    &--ai {
      padding: var(--spacing-2) var(--spacing-3);
      border: 1px dashed var(--color-ai-bright);
      border-radius: var(--radius-ai, var(--radius-md));
      background-color: var(--color-ai-bg);
      box-shadow: var(--shadow-ai, none);
    }
  }

  &__who {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
    margin-bottom: var(--spacing-1);
    font-size: var(--font-size-sm);
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  &__time {
    font-size: var(--font-size-xs);
    font-weight: 400;
    color: var(--color-text-tertiary);
  }

  &__row {
    display: flex;
    gap: var(--spacing-2);
    margin-top: var(--spacing-1);
    font-size: var(--font-size-sm);

    dt {
      flex-shrink: 0;
      width: 52px;
      font-weight: 500;
      color: var(--color-text-tertiary);
    }

    dd {
      flex: 1;
      line-height: 1.6;
      color: var(--color-text-secondary);
    }
  }

  &__label {
    &--highlight {
      color: var(--color-score-5);
    }

    &--improve {
      color: var(--color-score-2);
    }

    &--suggest {
      color: var(--color-primary);
    }
  }

  &__muted {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}
</style>
