<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import FilterBar from '@/components/common/FilterBar.vue'
import {
  closeImprovementApi,
  completeImprovementApi,
  createImprovementApi,
  fetchImprovementsApi,
  updateImprovementApi
} from '@/api/improvement'
import { fetchReflectionsApi } from '@/api/reflection'
import { useAuthStore } from '@/stores/auth'
import type { ImprovementItem, ImprovementStatus, Reflection } from '@/types/improvement'

/**
 * 待改进清单（N4 简化版）：标题 + 来源 + 完成说明 + 三态。
 * 教师：本人清单、新建、提交自评完成；督导：全部清单、确认关闭（可读关联反思）。
 */
const auth = useAuthStore()
const router = useRouter()

const isTeacher = computed(() => auth.role === 'teacher')
const isSupervisor = computed(() => auth.role === 'supervisor')

const loading = ref(true)
const error = ref(false)
const items = ref<ImprovementItem[]>([])
const statusFilter = ref<ImprovementStatus | ''>('')
const actingId = ref<number | null>(null)

const STATUS_META: Record<ImprovementStatus, { label: string; tone: string }> = {
  open: { label: '进行中', tone: 'info' },
  done_self: { label: '待督导确认', tone: 'warning' },
  closed: { label: '已关闭', tone: 'success' }
}

const counts = computed(() => {
  const acc = { open: 0, done_self: 0, closed: 0 }
  for (const it of items.value) acc[it.status] += 1
  return acc
})

async function load(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    const page = await fetchImprovementsApi({ status: statusFilter.value || undefined, pageSize: 50 })
    items.value = page.list ?? []
  } catch {
    error.value = true
    items.value = []
  } finally {
    loading.value = false
  }
}

/* ===== 教师：新建 / 编辑 / 自评完成 ===== */
const editorVisible = ref(false)
const editorSubmitting = ref(false)
const editor = reactive<{ id: number | null; title: string; sessionId: number | '' }>({
  id: null,
  title: '',
  sessionId: ''
})

function openCreate(): void {
  editor.id = null
  editor.title = ''
  editor.sessionId = ''
  editorVisible.value = true
}

function openEdit(item: ImprovementItem): void {
  editor.id = item.id
  editor.title = item.title
  editor.sessionId = item.sessionId || ''
  editorVisible.value = true
}

async function submitEditor(): Promise<void> {
  const title = editor.title.trim()
  if (!title) {
    ElMessage.error('请填写改进项标题')
    return
  }
  editorSubmitting.value = true
  try {
    if (editor.id === null) {
      await createImprovementApi({ title, sessionId: editor.sessionId === '' ? undefined : editor.sessionId, source: 'self' })
      ElMessage.success('改进项已创建')
    } else {
      await updateImprovementApi(editor.id, title)
      ElMessage.success('改进项已更新')
    }
    editorVisible.value = false
    await load()
  } catch {
    // 统一错误提示由 http 拦截器处理
  } finally {
    editorSubmitting.value = false
  }
}

const completeVisible = ref(false)
const completeNote = ref('')
const completeTarget = ref<ImprovementItem | null>(null)

function openComplete(item: ImprovementItem): void {
  completeTarget.value = item
  completeNote.value = ''
  completeVisible.value = true
}

async function submitComplete(): Promise<void> {
  if (!completeTarget.value) return
  actingId.value = completeTarget.value.id
  try {
    await completeImprovementApi(completeTarget.value.id, completeNote.value.trim())
    ElMessage.success('已提交自评，等待督导确认关闭')
    completeVisible.value = false
    await load()
  } catch {
    // ignore
  } finally {
    actingId.value = null
  }
}

/* ===== 督导：确认关闭 + 关联反思 ===== */
const reflectionVisible = ref(false)
const reflectionLoading = ref(false)
const reflections = ref<Reflection[]>([])

async function openReflection(item: ImprovementItem): Promise<void> {
  reflectionVisible.value = true
  reflectionLoading.value = true
  reflections.value = []
  try {
    reflections.value = item.sessionId ? await fetchReflectionsApi({ sessionId: item.sessionId }) : []
  } catch {
    reflections.value = []
  } finally {
    reflectionLoading.value = false
  }
}

async function handleClose(item: ImprovementItem): Promise<void> {
  try {
    await ElMessageBox.confirm(`确认关闭改进项「${item.title}」？关闭后教师端将标记为已完成。`, '确认关闭', {
      type: 'warning',
      confirmButtonText: '确认关闭'
    })
  } catch {
    return
  }
  actingId.value = item.id
  try {
    await closeImprovementApi(item.id)
    ElMessage.success('已确认关闭')
    await load()
  } catch {
    // ignore
  } finally {
    actingId.value = null
  }
}

function goSession(sessionId: number): void {
  if (!sessionId) return
  void router.push({ name: 'session-evaluation', params: { id: sessionId } })
}

onMounted(load)
watch(statusFilter, load)
</script>

<template>
  <div class="improve">
    <PageHeader title="待改进清单" :subtitle="isSupervisor ? '教师自评完成后由督导确认关闭' : '记录改进项并提交自评，督导确认后关闭'">
      <template #actions>
        <el-button v-if="isTeacher" type="primary" @click="openCreate">新建改进项</el-button>
      </template>
    </PageHeader>

    <FilterBar @search="load" @reset="statusFilter = ''">
      <el-select v-model="statusFilter" placeholder="全部状态" clearable class="improve__filter">
        <el-option label="进行中" value="open" />
        <el-option label="待督导确认" value="done_self" />
        <el-option label="已关闭" value="closed" />
      </el-select>
      <template #actions>
        <span class="improve__counts tabular-nums">
          进行中 {{ counts.open }} · 待确认 {{ counts.done_self }} · 已关闭 {{ counts.closed }}
        </span>
      </template>
    </FilterBar>

    <el-skeleton v-if="loading" :rows="6" animated />

    <EmptyState v-else-if="error" description="清单加载失败">
      <template #action>
        <el-button type="primary" @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <EmptyState v-else-if="!items.length" description="暂无待改进项">
      <template #action>
        <el-button v-if="isTeacher" type="primary" @click="openCreate">新建第一条</el-button>
      </template>
    </EmptyState>

    <ul v-else class="improve__list">
      <li
        v-for="(item, index) in items"
        :key="item.id"
        class="improve__item anim-rise"
        :data-status="item.status"
        :style="{ '--stagger': `${Math.min(index, 8) * 45}ms` }"
      >
        <div class="improve__main">
          <div class="improve__head">
            <span class="improve__title">{{ item.title }}</span>
            <el-tag size="small" :type="STATUS_META[item.status].tone as any" effect="light" round>
              {{ STATUS_META[item.status].label }}
            </el-tag>
            <el-tag size="small" effect="plain" round>{{ item.source === 'ai' ? 'AI' : item.source === 'supervisor' ? '督导' : '自建' }}</el-tag>
          </div>
          <p class="improve__meta">
            <span v-if="isSupervisor">{{ item.teacherName || '—' }} · </span>
            {{ item.createdAt.slice(0, 10) }}
            <template v-if="item.sessionId">
              · <el-button link type="primary" @click="goSession(item.sessionId)">查看当堂课</el-button>
            </template>
          </p>
          <p v-if="item.doneNote" class="improve__note">
            <span class="improve__note-tag">完成说明</span>{{ item.doneNote }}
          </p>
          <p v-if="item.closedByName" class="improve__closed">已由 {{ item.closedByName }} 于 {{ item.closedAt.slice(0, 10) }} 确认关闭</p>
        </div>

        <div class="improve__actions">
          <template v-if="isTeacher">
            <el-button v-if="item.status === 'open'" text type="primary" @click="openEdit(item)">编辑</el-button>
            <el-button
              v-if="item.status === 'open'"
              type="primary"
              plain
              size="small"
              :loading="actingId === item.id"
              @click="openComplete(item)"
            >
              提交自评完成
            </el-button>
          </template>
          <template v-else-if="isSupervisor">
            <el-button v-if="item.sessionId" text @click="openReflection(item)">关联反思</el-button>
            <el-button
              v-if="item.status === 'done_self'"
              type="primary"
              size="small"
              :loading="actingId === item.id"
              @click="handleClose(item)"
            >
              确认关闭
            </el-button>
          </template>
        </div>
      </li>
    </ul>

    <!-- 新建 / 编辑（教师） -->
    <el-dialog v-model="editorVisible" :title="editor.id === null ? '新建改进项' : '编辑改进项'" width="480px">
      <el-form label-width="72px">
        <el-form-item label="标题" required>
          <el-input v-model="editor.title" maxlength="255" show-word-limit placeholder="如：提问后延长等待时间至 3-5 秒" />
        </el-form-item>
        <el-form-item label="关联课次">
          <el-input v-model.number="editor.sessionId" placeholder="授课记录 id（选填）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="editorSubmitting" @click="submitEditor">保存</el-button>
      </template>
    </el-dialog>

    <!-- 提交自评完成（教师） -->
    <el-dialog v-model="completeVisible" title="提交自评完成" width="480px">
      <p class="improve__dialog-hint">说明你如何完成了这项改进（督导将据此确认关闭）。</p>
      <el-input v-model="completeNote" type="textarea" :rows="3" maxlength="1000" show-word-limit placeholder="完成说明" />
      <template #footer>
        <el-button @click="completeVisible = false">取消</el-button>
        <el-button type="primary" :loading="actingId !== null" @click="submitComplete">提交</el-button>
      </template>
    </el-dialog>

    <!-- 关联反思（督导只读） -->
    <el-dialog v-model="reflectionVisible" title="关联教学反思（只读）" width="560px">
      <el-skeleton v-if="reflectionLoading" :rows="3" animated />
      <EmptyState v-else-if="!reflections.length" description="该课次暂无教学反思" />
      <ul v-else class="improve__reflections">
        <li v-for="rf in reflections" :key="rf.id" class="improve__reflection">
          <p v-if="rf.content" class="improve__reflection-line"><b>反思：</b>{{ rf.content }}</p>
          <p v-if="rf.plan" class="improve__reflection-line"><b>计划：</b>{{ rf.plan }}</p>
          <p v-if="rf.nextAction" class="improve__reflection-line"><b>下节行动：</b>{{ rf.nextAction }}</p>
        </li>
      </ul>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.improve {
  &__filter { width: 160px; }
  &__counts { font-size: var(--font-size-xs); color: var(--color-text-tertiary); }

  &__list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
  }

  &__item {
    position: relative;
    display: flex;
    gap: var(--spacing-4);
    align-items: flex-start;
    justify-content: space-between;
    padding: var(--spacing-4) var(--spacing-4) var(--spacing-4) var(--spacing-5);
    overflow: hidden;
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
    transition: box-shadow var(--duration-fast) var(--ease-out-soft), transform var(--duration-fast) var(--ease-out-soft);

    &::before {
      position: absolute;
      top: 0;
      bottom: 0;
      left: 0;
      width: 4px;
      content: '';
      background-color: var(--color-text-disabled);
    }

    &[data-status='open']::before { background-color: var(--color-info); }
    &[data-status='done_self']::before { background-color: var(--color-warning); }
    &[data-status='closed']::before { background-color: var(--color-success); }

    &:hover {
      box-shadow: var(--shadow-card-hover);
      transform: translateY(-1px);
    }
  }

  &__main { min-width: 0; flex: 1; }

  &__head {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);
    align-items: center;
  }

  &__title {
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__meta {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__note {
    margin-top: var(--spacing-2);
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__note-tag {
    padding: 1px 8px;
    margin-right: var(--spacing-2);
    font-size: var(--font-size-xs);
    color: var(--color-primary);
    background-color: var(--color-primary-bg);
    border-radius: 999px;
  }

  &__closed {
    margin-top: var(--spacing-1);
    font-size: var(--font-size-xs);
    color: var(--color-success);
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    gap: var(--spacing-2);
    align-items: center;
  }

  &__dialog-hint {
    margin-bottom: var(--spacing-3);
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__reflections {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
  }

  &__reflection {
    padding: var(--spacing-3);
    background-color: var(--color-bg-page);
    border-radius: var(--radius-md);
  }

  &__reflection-line {
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);

    & + & { margin-top: var(--spacing-1); }
  }
}
</style>
