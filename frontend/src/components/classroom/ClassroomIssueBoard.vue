<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import EmptyState from '@/components/common/EmptyState.vue'
import {
  confirmClassroomIssueApi,
  createClassroomIssueApi,
  fetchClassroomIssuesApi,
  handleClassroomIssueApi
} from '@/api/classroom'
import { useAuthStore } from '@/stores/auth'
import { CLASSROOM_ISSUE_TYPES } from '@/types/classroom'
import type { ClassroomIssue, ClassroomIssueType } from '@/types/classroom'

/**
 * 课堂问题登记板：教师登记/确认，督导与主任处理。
 * 同一条问题由教师发起并闭环，督导/主任只处理、不删除。
 */
const auth = useAuthStore()
const isTeacher = computed(() => auth.role === 'teacher')

const loading = ref(true)
const error = ref(false)
const items = ref<ClassroomIssue[]>([])
const actingId = ref<number | null>(null)

const openCount = computed(() => items.value.filter((item) => item.status === 'open').length)
const pendingCount = computed(() => items.value.filter((item) => item.status === 'handled').length)

const TYPE_LABELS: Record<ClassroomIssueType, string> = {
  equipment: '设备故障',
  absence: '学生缺勤',
  progress: '进度问题',
  material: '材料缺失',
  other: '其他'
}

function typeLabel(type: ClassroomIssueType): string {
  return TYPE_LABELS[type] ?? '其他'
}

function statusLabel(item: ClassroomIssue): string {
  if (item.status === 'handled') return isTeacher.value ? '待教师确认' : '已处理，待教师确认'
  return '进行中'
}

async function load(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    items.value = (await fetchClassroomIssuesApi()) ?? []
  } catch {
    error.value = true
    items.value = []
  } finally {
    loading.value = false
  }
}

/* ===== 教师：登记 ===== */
const createVisible = ref(false)
const createSubmitting = ref(false)
const createForm = reactive<{ issueType: ClassroomIssueType; content: string }>({
  issueType: 'equipment',
  content: ''
})

function openCreate(): void {
  createForm.issueType = 'equipment'
  createForm.content = ''
  createVisible.value = true
}

async function submitCreate(): Promise<void> {
  const content = createForm.content.trim()
  if (!content) {
    ElMessage.error('请描述课堂中遇到的问题')
    return
  }
  createSubmitting.value = true
  try {
    await createClassroomIssueApi({ issueType: createForm.issueType, content })
    ElMessage.success('课堂问题已登记')
    createVisible.value = false
    await load()
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    createSubmitting.value = false
  }
}

/* ===== 督导 / 主任：处理 ===== */
const handleVisible = ref(false)
const handleSubmitting = ref(false)
const handleTarget = ref<ClassroomIssue | null>(null)
const handleNote = ref('')

function openHandle(item: ClassroomIssue): void {
  handleTarget.value = item
  handleNote.value = ''
  handleVisible.value = true
}

async function submitHandle(): Promise<void> {
  if (!handleTarget.value) return
  const note = handleNote.value.trim()
  if (!note) {
    ElMessage.error('请填写处理说明')
    return
  }
  handleSubmitting.value = true
  try {
    await handleClassroomIssueApi(handleTarget.value.id, note)
    ElMessage.success('已处理，等待教师确认')
    handleVisible.value = false
    await load()
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    handleSubmitting.value = false
  }
}

/* ===== 教师：确认解决 / 反馈未解决 ===== */
async function confirmResolved(item: ClassroomIssue): Promise<void> {
  try {
    await ElMessageBox.confirm('确认该问题已解决？确认后记录将被删除。', '确认解决', {
      type: 'warning',
      confirmButtonText: '已解决',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  actingId.value = item.id
  try {
    await confirmClassroomIssueApi(item.id, true)
    ElMessage.success('已确认解决')
    await load()
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    actingId.value = null
  }
}

async function feedbackUnresolved(item: ClassroomIssue): Promise<void> {
  actingId.value = item.id
  try {
    await confirmClassroomIssueApi(item.id, false)
    ElMessage.success('已反馈未解决，问题回到进行中')
    await load()
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    actingId.value = null
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <section class="issue-board" aria-label="课堂问题登记">
    <header class="issue-board__head">
      <div>
        <h2 class="issue-board__title">课堂问题登记</h2>
        <p class="issue-board__sub">
          <template v-if="isTeacher">记录设备、进度或材料问题，由督导或主任跟进处理。</template>
          <template v-else>教师登记的问题在这里集中处理，处理后等待教师确认。</template>
        </p>
      </div>
      <div class="issue-board__head-side">
        <span class="issue-board__stat tabular-nums">{{ openCount }} 进行中</span>
        <span v-if="pendingCount" class="issue-board__stat issue-board__stat--pending tabular-nums">
          {{ pendingCount }} 待确认
        </span>
        <el-button v-if="isTeacher" plain @click="openCreate">登记问题</el-button>
      </div>
    </header>

    <el-skeleton v-if="loading" :rows="3" animated />

    <EmptyState v-else-if="error" description="课堂问题加载失败">
      <template #action>
        <el-button size="small" @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <p v-else-if="!items.length" class="issue-board__empty">暂无课堂问题。</p>

    <ul v-else class="issue-board__list">
      <li v-for="item in items" :key="item.id" class="issue-row">
        <span class="issue-row__marker" :class="`issue-row__marker--${item.issueType}`" aria-hidden="true" />
        <div class="issue-row__main">
          <div class="issue-row__meta">
            <el-tag size="small" effect="plain" round>{{ typeLabel(item.issueType) }}</el-tag>
            <span class="issue-row__who">{{ item.teacherName || '教师' }}</span>
            <span class="issue-row__time tabular-nums">{{ item.createdAt.slice(0, 10) }}</span>
            <span
              class="issue-row__status"
              :class="{ 'issue-row__status--pending': item.status === 'handled' }"
            >
              {{ statusLabel(item) }}
            </span>
          </div>
          <p class="issue-row__content">{{ item.content }}</p>
          <div v-if="item.handleNote" class="issue-row__note">
            <span class="issue-row__note-label">
              {{ item.handlerName || '处理人' }}：
            </span>
            {{ item.handleNote }}
          </div>
        </div>

        <div class="issue-row__actions">
          <template v-if="!isTeacher && item.status === 'open'">
            <el-button
              size="small"
              plain
              :aria-label="`处理课堂问题：${item.content}`"
              @click="openHandle(item)"
            >
              处理
            </el-button>
          </template>
          <template v-if="isTeacher && item.status === 'handled'">
            <el-button
              size="small"
              :loading="actingId === item.id"
              @click="confirmResolved(item)"
            >
              确认解决
            </el-button>
            <el-button
              size="small"
              text
              :loading="actingId === item.id"
              @click="feedbackUnresolved(item)"
            >
              未解决
            </el-button>
          </template>
        </div>
      </li>
    </ul>

    <!-- 教师登记 -->
    <el-dialog v-model="createVisible" title="登记课堂问题" width="480px" append-to-body>
      <el-form label-width="72px">
        <el-form-item label="类型" required>
          <el-radio-group v-model="createForm.issueType">
            <el-radio v-for="type in CLASSROOM_ISSUE_TYPES" :key="type.value" :value="type.value">
              {{ type.label }}
            </el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="描述" required>
          <el-input
            v-model="createForm.content"
            type="textarea"
            :rows="4"
            maxlength="2000"
            show-word-limit
            placeholder="描述问题发生的时间、地点和影响"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createSubmitting" @click="submitCreate">提交</el-button>
      </template>
    </el-dialog>

    <!-- 督导 / 主任处理 -->
    <el-dialog v-model="handleVisible" title="处理课堂问题" width="480px" append-to-body>
      <p v-if="handleTarget" class="issue-board__target">{{ handleTarget.content }}</p>
      <el-form label-width="72px">
        <el-form-item label="处理说明" required>
          <el-input
            v-model="handleNote"
            type="textarea"
            :rows="4"
            maxlength="1000"
            show-word-limit
            placeholder="说明采取的措施和处理结果"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="handleVisible = false">取消</el-button>
        <el-button type="primary" :loading="handleSubmitting" @click="submitHandle">提交处理</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped lang="scss">
.issue-board {
  padding: var(--spacing-5) var(--spacing-6);
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);

  &__head {
    display: flex;
    gap: var(--spacing-4);
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__title {
    font-size: var(--font-size-lg);
  }

  &__sub {
    margin-top: 4px;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__head-side {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
  }

  &__stat {
    padding: 2px 10px;
    font-size: var(--font-size-xs);
    color: var(--color-text-secondary);
    background-color: var(--color-bg-page);
    border-radius: 999px;

    &--pending {
      color: #8a5a00;
      background-color: color-mix(in srgb, var(--color-warning) 14%, #ffffff);
    }
  }

  &__empty {
    padding: var(--spacing-4) 0;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__target {
    padding: var(--spacing-3);
    margin-bottom: var(--spacing-4);
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-secondary);
    background-color: var(--color-bg-page);
    border-radius: var(--radius-md);
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
  }
}

.issue-row {
  display: flex;
  gap: var(--spacing-3);
  align-items: flex-start;
  padding: var(--spacing-3) var(--spacing-4);
  background-color: var(--color-bg-page);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-md);

  &__marker {
    flex-shrink: 0;
    width: 4px;
    height: 36px;
    border-radius: 2px;
    background-color: var(--color-primary);

    &--equipment { background-color: var(--color-error); }
    &--absence { background-color: var(--color-warning); }
    &--progress { background-color: var(--color-info); }
    &--material { background-color: var(--color-ai); }
    &--other { background-color: var(--color-text-disabled); }
  }

  &__main {
    flex: 1;
    min-width: 0;
  }

  &__meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
  }

  &__who,
  &__time {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__status {
    padding: 1px 8px;
    font-size: var(--font-size-xs);
    color: var(--color-primary);
    background-color: var(--color-primary-bg);
    border-radius: 999px;

    &--pending {
      color: #8a5a00;
      background-color: color-mix(in srgb, var(--color-warning) 16%, #ffffff);
    }
  }

  &__content {
    margin-top: var(--spacing-2);
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-primary);
  }

  &__note {
    padding: var(--spacing-2) var(--spacing-3);
    margin-top: var(--spacing-2);
    font-size: var(--font-size-xs);
    line-height: 1.7;
    color: var(--color-text-secondary);
    background-color: var(--color-bg-card);
    border-left: 2px solid var(--color-divider);
    border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  }

  &__note-label {
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    gap: var(--spacing-1);
    align-items: center;
  }
}

@media (max-width: 960px) {
  .issue-board__head {
    flex-direction: column;
  }

  .issue-row__actions {
    margin-left: var(--spacing-2);
  }
}
</style>
