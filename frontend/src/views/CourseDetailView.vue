<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ResourceList from '@/components/course/ResourceList.vue'
import ResourceUploader from '@/components/course/ResourceUploader.vue'
import SessionTable from '@/components/session/SessionTable.vue'
import { fetchCourseDetailApi } from '@/api/course'
import { deleteResourceApi, downloadResourceApi, fetchResourcesApi, uploadResourceApi } from '@/api/resource'
import { createSessionApi, fetchCourseSessionsApi } from '@/api/session'
import { getCourseStatusMeta, RESOURCE_ACCEPT, MAX_RESOURCE_SIZE, PAGE_SIZES } from '@/constants'
import { useAuthStore } from '@/stores/auth'
import type { Course } from '@/types/course'
import type { Resource } from '@/types/resource'
import type { SessionListItem } from '@/types/evaluation'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const course = ref<Course | null>(null)
const loading = ref(true)
const error = ref(false)

const resources = ref<Resource[]>([])
const resourceLoading = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)

const sessions = ref<SessionListItem[]>([])
const sessionsLoading = ref(false)
const sessionsError = ref(false)
const sessionTotal = ref(0)
const sessionPage = ref(1)
const sessionPageSize = ref(10)

const createVisible = ref(false)
const createSubmitting = ref(false)
const createForm = reactive<{
  classId: number | ''
  sessionDate: string
  period: string
  topic: string
}>({
  classId: '',
  sessionDate: '',
  period: '',
  topic: ''
})

const activeTab = ref<'basic' | 'classes' | 'resource' | 'sessions'>('basic')

const courseId = computed(() => String(route.params.id ?? ''))

const canManageResources = computed(
  () => auth.role === 'teacher' && course.value?.teacherId === auth.user?.id
)

const canCreateSession = computed(() => auth.role === 'supervisor')

const statusMeta = computed(() =>
  course.value ? getCourseStatusMeta(course.value.status) : null
)

const summary = computed(() => {
  if (!course.value) return []
  return [
    { label: '学分', value: course.value.credit, unit: '分' },
    { label: '学时', value: course.value.hours, unit: '学时' },
    { label: '班级数', value: course.value.classes?.length ?? course.value.classCount, unit: '个' },
    { label: '学生人次', value: course.value.studentCount, unit: '人次' },
    { label: '课程资源', value: course.value.resourceCount, unit: '份' }
  ]
})

async function loadCourse(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    course.value = await fetchCourseDetailApi(courseId.value)
  } catch {
    error.value = true
    course.value = null
  } finally {
    loading.value = false
  }
}

async function loadResources(): Promise<void> {
  resourceLoading.value = true
  try {
    resources.value = await fetchResourcesApi(courseId.value)
  } catch {
    resources.value = []
  } finally {
    resourceLoading.value = false
  }
}

async function loadSessions(): Promise<void> {
  sessionsLoading.value = true
  sessionsError.value = false
  try {
    const result = await fetchCourseSessionsApi(courseId.value, {
      page: sessionPage.value,
      pageSize: sessionPageSize.value
    })
    sessions.value = result.list
    sessionTotal.value = result.total
  } catch {
    sessionsError.value = true
    sessions.value = []
    sessionTotal.value = 0
  } finally {
    sessionsLoading.value = false
  }
}

function handleSessionPageChange(value: number): void {
  sessionPage.value = value
  void loadSessions()
}

function handleSessionPageSizeChange(value: number): void {
  sessionPageSize.value = value
  sessionPage.value = 1
  void loadSessions()
}

function handleSessionView(session: SessionListItem): void {
  void router.push({ name: 'session-evaluation', params: { id: session.id } })
}

function disabledFuture(date: Date): boolean {
  return date.getTime() > Date.now()
}

function openCreateSession(): void {
  createForm.classId = ''
  createForm.sessionDate = ''
  createForm.period = ''
  createForm.topic = ''
  createVisible.value = true
}

async function submitCreateSession(): Promise<void> {
  if (!createForm.sessionDate) {
    ElMessage.error('请选择授课日期')
    return
  }
  if (!createForm.period.trim()) {
    ElMessage.error('请填写节次')
    return
  }
  createSubmitting.value = true
  try {
    await createSessionApi({
      courseId: Number(courseId.value),
      classId: createForm.classId === '' ? 0 : createForm.classId,
      sessionDate: createForm.sessionDate,
      period: createForm.period.trim(),
      topic: createForm.topic.trim()
    })
    ElMessage.success('授课记录已创建')
    createVisible.value = false
    sessionPage.value = 1
    await loadSessions()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    createSubmitting.value = false
  }
}

async function handleUpload(file: File): Promise<void> {
  uploading.value = true
  uploadProgress.value = 0
  try {
    await uploadResourceApi(courseId.value, file, (percent) => {
      uploadProgress.value = percent
    })
    ElMessage.success('资源上传成功')
    await loadResources()
    if (course.value) {
      course.value.resourceCount = resources.value.length
    }
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    uploading.value = false
    uploadProgress.value = 0
  }
}

async function handleDelete(resource: Resource): Promise<void> {
  try {
    await ElMessageBox.confirm(`确定删除资源「${resource.name}」吗？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  await deleteResourceApi(resource.id)
  ElMessage.success('资源已删除')
  await loadResources()
  if (course.value) {
    course.value.resourceCount = resources.value.length
  }
}

async function handleDownload(resource: Resource): Promise<void> {
  try {
    const blob = await downloadResourceApi(resource.id)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = resource.name
    link.click()
    URL.revokeObjectURL(url)
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  }
}

function goEdit(): void {
  void router.push({ name: 'course-edit', params: { id: courseId.value } })
}

function goBack(): void {
  router.back()
}

watch(
  () => route.query.tab,
  (tab) => {
    if (tab === 'resource' || tab === 'classes' || tab === 'basic' || tab === 'sessions') {
      activeTab.value = tab
    }
  },
  { immediate: true }
)

onMounted(async () => {
  await loadCourse()
  await loadResources()
  await loadSessions()
})
</script>

<template>
  <div class="course-detail">
    <PageHeader v-if="course">
      <template #title>
        <div class="course-detail__title">
          <el-button link @click="goBack">返回</el-button>
          <span class="course-detail__name">{{ course.name }}</span>
          <el-tag effect="plain">{{ course.code }}</el-tag>
          <el-tag v-if="statusMeta" :type="statusMeta.tag" effect="light" round>
            {{ statusMeta.label }}
          </el-tag>
        </div>
      </template>
      <template #actions>
        <el-button v-if="auth.role === 'director'" type="primary" @click="goEdit">
          编辑课程
        </el-button>
        <el-button
          v-if="canManageResources"
          type="primary"
          @click="activeTab = 'resource'"
        >
          上传资源
        </el-button>
      </template>
    </PageHeader>

    <el-skeleton v-if="loading" :rows="10" animated />

    <EmptyState v-else-if="error" description="课程信息加载失败">
      <template #action>
        <el-button type="primary" @click="loadCourse">重新加载</el-button>
      </template>
    </EmptyState>

    <template v-else-if="course">
      <div class="course-detail__summary">
        <div v-for="item in summary" :key="item.label" class="course-detail__summary-item">
          <span class="course-detail__summary-value tabular-nums">
            {{ item.value }}<em>{{ item.unit }}</em>
          </span>
          <span class="course-detail__summary-label">{{ item.label }}</span>
        </div>
      </div>

      <div class="course-detail__info">
        <span>所属教研室：{{ course.department }}</span>
        <span>授课教师：{{ course.teacherName }}</span>
        <span>学期：{{ course.semester }}</span>
      </div>

      <el-tabs v-model="activeTab" class="course-detail__tabs">
        <el-tab-pane label="基本信息" name="basic">
          <div class="course-detail__section">
            <h3 class="course-detail__section-title">课程简介</h3>
            <p class="course-detail__paragraph">{{ course.description || '暂无课程简介' }}</p>
          </div>
          <div v-if="course.objective" class="course-detail__section">
            <h3 class="course-detail__section-title">培养目标</h3>
            <p class="course-detail__paragraph">{{ course.objective }}</p>
          </div>
          <div v-if="course.major" class="course-detail__section">
            <h3 class="course-detail__section-title">适用专业</h3>
            <p class="course-detail__paragraph">{{ course.major }}</p>
          </div>
        </el-tab-pane>

        <el-tab-pane
          :label="`开课信息（${course.classes?.length ?? 0}）`"
          name="classes"
        >
          <el-table :data="course.classes ?? []" row-key="id" stripe>
            <el-table-column prop="className" label="班级名称" min-width="160" />
            <el-table-column prop="schedule" label="上课时间" min-width="140" />
            <el-table-column prop="location" label="上课地点" min-width="140" />
            <el-table-column label="学生数" width="110" align="right">
              <template #default="{ row }">
                <span class="tabular-nums">{{ row.studentCount }}</span>
              </template>
            </el-table-column>
            <template #empty>
              <span class="course-detail__empty-text">暂无开课班级信息</span>
            </template>
          </el-table>
        </el-tab-pane>

        <el-tab-pane :label="`课程资源（${resources.length}）`" name="resource">
          <ResourceUploader
            v-if="canManageResources"
            :accept="RESOURCE_ACCEPT"
            :max-size="MAX_RESOURCE_SIZE"
            :uploading="uploading"
            :progress="uploadProgress"
            @file="handleUpload"
          />
          <div class="course-detail__resource-list">
            <ResourceList
              :resources="resources"
              :can-manage="canManageResources"
              :loading="resourceLoading"
              @download="handleDownload"
              @delete="handleDelete"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane :label="`历史授课记录（${sessionTotal}）`" name="sessions">
          <div class="course-detail__session-head">
            <span class="course-detail__session-hint">
              点击任意记录进入当堂课质量评估；未评价侧显示「—」
            </span>
            <el-button
              v-if="canCreateSession"
              type="primary"
              @click="openCreateSession"
            >
              新增授课记录
            </el-button>
          </div>

          <div v-if="sessionsError" class="course-detail__session-state">
            <EmptyState description="授课记录加载失败">
              <template #action>
                <el-button type="primary" @click="loadSessions">重新加载</el-button>
              </template>
            </EmptyState>
          </div>
          <template v-else>
            <SessionTable
              :rows="sessions"
              :loading="sessionsLoading"
              @view="handleSessionView"
            />
            <EmptyState
              v-if="!sessionsLoading && sessions.length === 0 && canCreateSession"
              description="还没有授课记录，去创建"
            >
              <template #action>
                <el-button type="primary" @click="openCreateSession">
                  新增授课记录
                </el-button>
              </template>
            </EmptyState>
            <EmptyState
              v-else-if="!sessionsLoading && sessions.length === 0"
              description="暂无授课记录"
            />
            <div class="course-detail__pagination">
              <el-pagination
                :current-page="sessionPage"
                :page-size="sessionPageSize"
                :page-sizes="PAGE_SIZES"
                :total="sessionTotal"
                layout="total, sizes, prev, pager, next"
                background
                @current-change="handleSessionPageChange"
                @size-change="handleSessionPageSizeChange"
              />
            </div>
          </template>
        </el-tab-pane>
      </el-tabs>

      <el-dialog v-model="createVisible" title="新增授课记录" width="480px">
        <el-form label-width="88px">
          <el-form-item label="授课日期" required>
            <el-date-picker
              v-model="createForm.sessionDate"
              type="date"
              placeholder="不得晚于今天"
              value-format="YYYY-MM-DD"
              :disabled-date="disabledFuture"
              class="course-detail__dialog-field"
            />
          </el-form-item>
          <el-form-item label="节次" required>
            <el-input
              v-model="createForm.period"
              placeholder="如 3-4 节"
              maxlength="32"
              class="course-detail__dialog-field"
            />
          </el-form-item>
          <el-form-item label="主题">
            <el-input
              v-model="createForm.topic"
              placeholder="本次课主题（选填）"
              maxlength="128"
              class="course-detail__dialog-field"
            />
          </el-form-item>
          <el-form-item label="班级">
            <el-select
              v-model="createForm.classId"
              placeholder="不指定"
              clearable
              class="course-detail__dialog-field"
            >
              <el-option
                v-for="item in course?.classes ?? []"
                :key="item.id"
                :label="item.className"
                :value="item.id"
              />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="createVisible = false">取消</el-button>
          <el-button type="primary" :loading="createSubmitting" @click="submitCreateSession">
            创建
          </el-button>
        </template>
      </el-dialog>
    </template>
  </div>
</template>

<style scoped lang="scss">
.course-detail {
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

  &__summary {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: var(--spacing-4);
    padding: var(--spacing-6);
    margin-bottom: var(--spacing-4);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);

    @media (max-width: 1280px) {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  &__summary-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--spacing-1);
  }

  &__summary-value {
    font-size: var(--font-size-stat);
    font-weight: 600;
    color: var(--color-text-primary);

    em {
      margin-left: 2px;
      font-size: var(--font-size-sm);
      font-style: normal;
      font-weight: 400;
      color: var(--color-text-tertiary);
    }
  }

  &__summary-label {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__info {
    display: flex;
    gap: var(--spacing-6);
    padding: var(--spacing-3) var(--spacing-6);
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
  }

  &__tabs {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__section {
    margin-bottom: var(--spacing-6);

    &:last-child {
      margin-bottom: 0;
    }
  }

  &__section-title {
    margin-bottom: var(--spacing-2);
    font-size: var(--font-size-lg);
    color: var(--color-text-primary);
  }

  &__paragraph {
    font-size: var(--font-size-base);
    line-height: 1.8;
    color: var(--color-text-secondary);
  }

  &__resource-list {
    margin-top: var(--spacing-4);
  }

  &__empty-text {
    color: var(--color-text-tertiary);
  }

  &__session-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-4);
    margin-bottom: var(--spacing-4);
  }

  &__session-hint {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__session-state {
    padding: var(--spacing-6);
  }

  &__pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--spacing-4);
  }

  &__dialog-field {
    width: 100%;
  }
}
</style>
