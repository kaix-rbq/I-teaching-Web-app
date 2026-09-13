<script setup lang="ts">
import { computed, ref } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { COURSE_STATUS } from '@/constants'
import { useDictStore } from '@/stores/dict'
import type { CourseFormModel } from '@/types/course'
import type { DepartmentOption } from '@/types/dict'

const props = withDefaults(
  defineProps<{
    mode: 'create' | 'edit'
    submitting?: boolean
    /** 可选教研室；主任场景由页面收敛为本室 */
    departmentOptions?: DepartmentOption[]
  }>(),
  {
    mode: 'create',
    submitting: false,
    departmentOptions: () => []
  }
)

const model = defineModel<CourseFormModel>({ required: true })

const emit = defineEmits<{
  (e: 'submit', value: CourseFormModel): void
  (e: 'cancel'): void
}>()

const dict = useDictStore()
const formRef = ref<FormInstance>()

const isEdit = computed(() => props.mode === 'edit')

const teacherOptions = computed(() => dict.teachersOfDepartment(model.value.departmentId))

const rules: FormRules<CourseFormModel> = {
  code: [
    { required: true, message: '请输入课程编码', trigger: 'blur' },
    {
      pattern: /^[A-Z]{2,4}\d{4}$/,
      message: '编码格式为 2-4 位大写字母 + 4 位数字，如 SE3101',
      trigger: 'blur'
    }
  ],
  name: [{ required: true, message: '请输入课程名称', trigger: 'blur' }],
  departmentId: [{ required: true, message: '请选择所属教研室', trigger: 'change' }],
  teacherId: [{ required: true, message: '请选择授课教师', trigger: 'change' }],
  semester: [{ required: true, message: '请选择学期', trigger: 'change' }],
  credit: [{ required: true, message: '请输入学分', trigger: 'change' }],
  hours: [{ required: true, message: '请输入学时', trigger: 'change' }],
  status: [{ required: true, message: '请选择课程状态', trigger: 'change' }]
}

function handleDepartmentChange(): void {
  const valid = teacherOptions.value.some((teacher) => teacher.id === model.value.teacherId)
  if (!valid) {
    model.value.teacherId = ''
  }
}

async function handleSubmit(): Promise<void> {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  emit('submit', { ...model.value })
}

function handleCancel(): void {
  emit('cancel')
}
</script>

<template>
  <el-form
    ref="formRef"
    :model="model"
    :rules="rules"
    label-width="100px"
    label-position="right"
    class="course-form"
  >
    <el-form-item label="课程编码" prop="code">
      <el-input
        v-model="model.code"
        :disabled="isEdit"
        placeholder="如 SE3101"
        maxlength="8"
        class="course-form__code"
      />
    </el-form-item>

    <el-form-item label="课程名称" prop="name">
      <el-input v-model="model.name" placeholder="请输入课程名称" maxlength="50" />
    </el-form-item>

    <el-form-item label="所属教研室" prop="departmentId">
      <el-select v-model="model.departmentId" placeholder="请选择教研室" @change="handleDepartmentChange">
        <el-option
          v-for="department in props.departmentOptions"
          :key="department.id"
          :label="department.name"
          :value="department.id"
        />
      </el-select>
    </el-form-item>

    <el-form-item label="授课教师" prop="teacherId">
      <el-select
        v-model="model.teacherId"
        placeholder="请先选择教研室"
        :disabled="!model.departmentId"
      >
        <el-option
          v-for="teacher in teacherOptions"
          :key="teacher.id"
          :label="teacher.name"
          :value="teacher.id"
        />
      </el-select>
    </el-form-item>

    <el-form-item label="学期" prop="semester">
      <el-select v-model="model.semester" placeholder="请选择学期">
        <el-option
          v-for="semester in dict.semesters"
          :key="semester"
          :label="semester"
          :value="semester"
        />
      </el-select>
    </el-form-item>

    <el-form-item label="学分" prop="credit">
      <el-input-number v-model="model.credit" :min="1" :max="6" :step="0.5" />
    </el-form-item>

    <el-form-item label="学时" prop="hours">
      <el-input-number v-model="model.hours" :min="1" :max="200" :step="1" />
    </el-form-item>

    <el-form-item label="课程简介" prop="description">
      <el-input
        v-model="model.description"
        type="textarea"
        :rows="4"
        maxlength="500"
        show-word-limit
        placeholder="请输入课程简介（不超过 500 字）"
      />
    </el-form-item>

    <el-form-item label="状态" prop="status">
      <el-radio-group v-model="model.status">
        <el-radio v-for="item in COURSE_STATUS" :key="item.value" :value="item.value">
          {{ item.label }}
        </el-radio>
      </el-radio-group>
    </el-form-item>

    <el-form-item>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      <el-button @click="handleCancel">取消</el-button>
    </el-form-item>
  </el-form>
</template>

<style scoped lang="scss">
.course-form {
  max-width: 640px;

  &__code {
    max-width: 220px;
  }

  :deep(.el-form-item__label) {
    color: var(--color-text-secondary);
  }
}
</style>
