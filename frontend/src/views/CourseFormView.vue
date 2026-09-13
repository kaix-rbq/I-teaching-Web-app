<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import CourseInfoForm from '@/components/course/CourseInfoForm.vue'
import { createCourseApi, fetchCourseDetailApi, updateCourseApi } from '@/api/course'
import { useDictStore } from '@/stores/dict'
import type { CourseFormModel } from '@/types/course'

const route = useRoute()
const router = useRouter()
const dict = useDictStore()

const isEdit = computed(() => route.name === 'course-edit')
const courseId = computed(() => String(route.params.id ?? ''))

const loading = ref(false)
const submitting = ref(false)

const form = reactive<CourseFormModel>({
  code: '',
  name: '',
  credit: 3,
  hours: 48,
  semester: '',
  department: '',
  teacherId: '',
  description: '',
  status: 'open'
})

let initialSnapshot = ''

function snapshot(): string {
  return JSON.stringify(form)
}

function isDirty(): boolean {
  return snapshot() !== initialSnapshot
}

function fillForm(data: CourseFormModel): void {
  Object.assign(form, data)
  initialSnapshot = snapshot()
}

async function loadDetail(): Promise<void> {
  if (!isEdit.value) return
  loading.value = true
  try {
    const course = await fetchCourseDetailApi(courseId.value)
    fillForm({
      id: course.id,
      code: course.code,
      name: course.name,
      credit: course.credit,
      hours: course.hours,
      semester: course.semester,
      department: course.department,
      teacherId: course.teacherId,
      description: course.description,
      status: course.status
    })
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    loading.value = false
  }
}

async function handleSubmit(value: CourseFormModel): Promise<void> {
  submitting.value = true
  try {
    const payload: Omit<CourseFormModel, 'id'> = {
      code: value.code,
      name: value.name,
      credit: value.credit,
      hours: value.hours,
      semester: value.semester,
      department: value.department,
      teacherId: value.teacherId,
      description: value.description,
      status: value.status
    }
    const result = isEdit.value
      ? await updateCourseApi(courseId.value, payload)
      : await createCourseApi(payload)
    ElMessage.success(isEdit.value ? '课程已更新' : '课程已创建')
    initialSnapshot = snapshot()
    await router.push({ name: 'course-detail', params: { id: result.id } })
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    submitting.value = false
  }
}

async function handleCancel(): Promise<void> {
  if (isDirty()) {
    try {
      await ElMessageBox.confirm('当前修改尚未保存，确定离开吗？', '离开确认', {
        type: 'warning',
        confirmButtonText: '离开',
        cancelButtonText: '继续编辑'
      })
    } catch {
      return
    }
  }
  router.back()
}

onMounted(async () => {
  await dict.load()
  if (!isEdit.value) {
    form.semester = dict.semesters[0] ?? ''
    initialSnapshot = snapshot()
  }
  await loadDetail()
})
</script>

<template>
  <div class="course-form-view">
    <PageHeader :title="isEdit ? '编辑课程' : '新增课程'" subtitle="维护课程基础信息与开课状态">
      <template #actions>
        <el-button @click="handleCancel">取消</el-button>
      </template>
    </PageHeader>

    <div class="course-form-view__body">
      <el-skeleton v-if="loading" :rows="10" animated />
      <CourseInfoForm
        v-else
        v-model="form"
        :mode="isEdit ? 'edit' : 'create'"
        :submitting="submitting"
        @submit="handleSubmit"
        @cancel="handleCancel"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.course-form-view {
  &__body {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }
}
</style>
