<script setup lang="ts">
import { ref } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  DIMENSION_ANCHORS,
  EVALUATION_DIMENSIONS,
  getDimensionAnchor,
  getScoreLevel,
  SCORE_LEVELS
} from '@/constants'
import type { DimensionKey, EvaluationFormModel } from '@/types/evaluation'

const props = withDefaults(
  defineProps<{
    /** 只读：主任（本室）/ 教师（本人）进入时禁用并隐藏提交 */
    readonly?: boolean
    submitting?: boolean
  }>(),
  {
    readonly: false,
    submitting: false
  }
)

const model = defineModel<EvaluationFormModel>({ required: true })

const emit = defineEmits<{ (e: 'submit'): void }>()

const formRef = ref<FormInstance>()

const rules: FormRules<EvaluationFormModel> = {
  objective: [{ required: true, message: '请选择「教学目标与内容准确性」评分', trigger: 'change' }],
  content: [{ required: true, message: '请选择「内容质量与深度」评分', trigger: 'change' }],
  interaction: [{ required: true, message: '请选择「学生互动与参与」评分', trigger: 'change' }],
  organization: [{ required: true, message: '请选择「课堂组织与节奏」评分', trigger: 'change' }],
  frontier: [{ required: true, message: '请选择「前沿与交叉学科」评分', trigger: 'change' }]
}

function anchorContent(dimension: DimensionKey, score: number): string {
  const level = getScoreLevel(score)
  return `${score} 分 · ${level?.level ?? ''}：${getDimensionAnchor(dimension, score)}`
}

function weightLabel(weight: number): string {
  return `${Math.round(weight * 100)}%`
}

function setDimension(key: DimensionKey, value: string | number | boolean | undefined): void {
  model.value[key] = typeof value === 'number' ? value : null
}

async function handleSubmit(): Promise<void> {
  if (props.readonly || props.submitting) return
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  emit('submit')
}
</script>

<template>
  <el-form
    ref="formRef"
    :model="model"
    :rules="rules"
    label-position="top"
    :disabled="readonly"
    class="evaluation-form"
  >
    <div
      v-for="dimension in EVALUATION_DIMENSIONS"
      :key="dimension.key"
      class="evaluation-form__dimension"
    >
      <el-form-item :prop="dimension.key" class="evaluation-form__item">
        <template #label>
          <div class="evaluation-form__label">
            <span class="evaluation-form__name">{{ dimension.name }}</span>
            <el-tag v-if="dimension.isObservation" type="warning" size="small" effect="light" round>
              观测项
            </el-tag>
            <el-tag v-else type="info" size="small" effect="plain" round>
              权重 {{ weightLabel(dimension.weight) }}
            </el-tag>
            <el-tooltip placement="top" :content="dimension.description">
              <span class="evaluation-form__hint">?</span>
            </el-tooltip>
          </div>
        </template>

        <el-radio-group
          :model-value="model[dimension.key] ?? undefined"
          class="evaluation-form__scores"
          @update:model-value="(value) => setDimension(dimension.key, value)"
        >
          <el-tooltip
            v-for="level in SCORE_LEVELS"
            :key="level.value"
            placement="top"
            :content="anchorContent(dimension.key, level.value)"
          >
            <el-radio :value="level.value" border>{{ level.value }}</el-radio>
          </el-tooltip>
        </el-radio-group>
      </el-form-item>

      <p class="evaluation-form__anchors">
        <span v-for="level in SCORE_LEVELS" :key="level.value" class="evaluation-form__anchor">
          {{ level.value }} {{ DIMENSION_ANCHORS[dimension.key][level.value] }}
        </span>
      </p>
    </div>

    <el-divider content-position="left">结构化评语</el-divider>

    <el-form-item label="总体评语（选填）">
      <el-input
        v-model="model.comment"
        type="textarea"
        :rows="2"
        maxlength="2000"
        show-word-limit
        placeholder="当堂课的总体印象与结论"
      />
    </el-form-item>

    <el-form-item label="亮点">
      <el-input
        v-model="model.highlights"
        type="textarea"
        :rows="2"
        maxlength="1000"
        show-word-limit
        placeholder="值得肯定、可推广的做法"
      />
    </el-form-item>

    <el-form-item label="待改进">
      <el-input
        v-model="model.improvements"
        type="textarea"
        :rows="2"
        maxlength="1000"
        show-word-limit
        placeholder="本次课暴露出的短板"
      />
    </el-form-item>

    <el-form-item label="建议">
      <el-input
        v-model="model.suggestions"
        type="textarea"
        :rows="2"
        maxlength="1000"
        show-word-limit
        placeholder="针对性的改进建议"
      />
    </el-form-item>

    <div v-if="!readonly" class="evaluation-form__actions">
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        提交评分
      </el-button>
      <span class="evaluation-form__tip">提交后再次提交将覆盖你上次的评分</span>
    </div>

    <el-alert
      v-else
      class="evaluation-form__readonly"
      type="info"
      :closable="false"
      show-icon
      title="你以只读身份查看本次评估，评分与评语仅教学督导可填写"
    />
  </el-form>
</template>

<style scoped lang="scss">
.evaluation-form {
  &__dimension {
    padding-bottom: var(--spacing-3);
    margin-bottom: var(--spacing-3);
    border-bottom: 1px dashed var(--color-divider);

    &:last-of-type {
      border-bottom: none;
    }
  }

  &__label {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__name {
    font-weight: 600;
  }

  &__hint {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    cursor: help;
    border: 1px solid var(--color-border);
    border-radius: 50%;
  }

  &__scores {
    display: flex;
    gap: var(--spacing-2);
  }

  &__anchors {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin-top: var(--spacing-2);
    font-size: var(--font-size-xs);
    line-height: 1.6;
    color: var(--color-text-tertiary);
  }

  &__anchor {
    display: block;
  }

  &__actions {
    display: flex;
    align-items: center;
    gap: var(--spacing-3);
  }

  &__tip {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__readonly {
    margin-top: var(--spacing-2);
  }

  :deep(.el-form-item__label) {
    padding-bottom: var(--spacing-2);
    line-height: 1.4;
  }

  :deep(.el-radio.is-bordered) {
    margin-right: 0;
  }
}
</style>
