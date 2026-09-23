<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RoleTag from '@/components/common/RoleTag.vue'
import TeacherScorePanel from '@/components/teacher/TeacherScorePanel.vue'
import { fetchTeacherEvaluationsApi, fetchTeacherSummaryApi } from '@/api/teacher'
import { useDictStore } from '@/stores/dict'
import type { TeacherSummary, TeacherTimelineItem } from '@/types/teacher'

const route = useRoute()
const router = useRouter()
const dict = useDictStore()

const summary = ref<TeacherSummary | null>(null)
const timeline = ref<TeacherTimelineItem[]>([])
const loading = ref(true)
const timelineLoading = ref(false)
const error = ref(false)
const semester = ref('')

const teacherId = computed(() => String(route.params.id ?? ''))

async function load(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    summary.value = await fetchTeacherSummaryApi(teacherId.value, {
      semester: semester.value || undefined
    })
  } catch {
    error.value = true
    summary.value = null
    loading.value = false
    return
  }
  loading.value = false

  timelineLoading.value = true
  try {
    const result = await fetchTeacherEvaluationsApi(teacherId.value, {
      semester: semester.value || undefined,
      pageSize: 50
    })
    timeline.value = result.list
  } catch {
    timeline.value = []
  } finally {
    timelineLoading.value = false
  }
}

function handleSemesterChange(): void {
  void load()
}

function goBack(): void {
  router.back()
}

function handleSelectCourse(courseId: number): void {
  void router.push({ name: 'course-detail', params: { id: courseId } })
}

onMounted(() => {
  void dict.load()
  void load()
})
</script>

<template>
  <div class="teacher-detail">
    <PageHeader>
      <template #title>
        <div class="teacher-detail__title">
          <el-button link @click="goBack">返回</el-button>
          <span class="teacher-detail__name">{{ summary?.teacherName || '教师评分面板' }}</span>
          <RoleTag role="teacher" />
        </div>
      </template>
      <template #subtitle>综合分、分维度、按课程明细与历次评价时间线</template>
      <template #actions>
        <el-select
          v-model="semester"
          placeholder="学期"
          clearable
          class="teacher-detail__semester"
          @change="handleSemesterChange"
        >
          <el-option
            v-for="item in dict.semesters"
            :key="item"
            :label="item"
            :value="item"
          />
        </el-select>
      </template>
    </PageHeader>

    <el-alert
      class="teacher-detail__ethics"
      type="info"
      :closable="false"
      show-icon
      title="仅用于教学支持，不作为考核依据"
      description="评分用于帮助教师改进教学，不用于绩效与人事评价。请勿外传或作他用。"
    />

    <el-skeleton v-if="loading" :rows="10" animated />

    <EmptyState v-else-if="error || !summary" description="教师评分数据加载失败">
      <template #action>
        <el-button type="primary" @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <TeacherScorePanel
      v-else
      :summary="summary"
      :timeline="timeline"
      :timeline-loading="timelineLoading"
      @select-course="handleSelectCourse"
    />
  </div>
</template>

<style scoped lang="scss">
.teacher-detail {
  &__title {
    display: flex;
    align-items: center;
    gap: var(--spacing-3);
  }

  &__name {
    font-size: var(--font-size-3xl);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__semester {
    width: 180px;
  }

  &__ethics {
    margin-bottom: var(--spacing-4);
  }
}
</style>
