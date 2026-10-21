<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { createReflectionApi, fetchReflectionsApi, updateReflectionApi } from '@/api/reflection'
import {
  completeImprovementApi,
  createImprovementApi,
  deleteImprovementApi,
  fetchImprovementsApi,
  updateImprovementApi
} from '@/api/improvement'
import { fetchTeacherEvaluationsApi } from '@/api/teacher'
import { useAuthStore } from '@/stores/auth'
import type { ImprovementItem, Reflection } from '@/types/improvement'
import type { TeacherTimelineItem } from '@/types/teacher'

/**
 * 教学反思 + 待改进项双栏：左侧沉淀反思，右侧把反思落成可勾选的改进项。
 * 改进项状态简化为「进行中 / 已完成」；自评完成即关闭，不再需要督导确认。
 */
const auth = useAuthStore()
const isTeacher = computed(() => auth.role === 'teacher')

const loading = ref(true)
const error = ref(false)
const items = ref<Reflection[]>([])
const improvements = ref<ImprovementItem[]>([])
const improvementsLoading = ref(true)
const improvementsError = ref(false)

const openImprovements = computed(() => improvements.value.filter((item) => item.status !== 'closed'))
const closedImprovements = computed(() => improvements.value.filter((item) => item.status === 'closed'))

async function loadReflections(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    items.value = await fetchReflectionsApi()
  } catch {
    error.value = true
    items.value = []
  } finally {
    loading.value = false
  }
}

async function loadImprovements(): Promise<void> {
  improvementsLoading.value = true
  improvementsError.value = false
  try {
    const page = await fetchImprovementsApi({ pageSize: 50 })
    improvements.value = page.list ?? []
  } catch {
    improvementsError.value = true
    improvements.value = []
  } finally {
    improvementsLoading.value = false
  }
}

/* ===== 反思 CRUD（保持原有能力） ===== */
const editorVisible = ref(false)
const editorSubmitting = ref(false)
const editor = reactive<{
  id: number | null
  sessionId: number | ''
  content: string
  plan: string
  nextAction: string
}>({ id: null, sessionId: '', content: '', plan: '', nextAction: '' })

function openCreate(): void {
  editor.id = null
  editor.sessionId = ''
  editor.content = ''
  editor.plan = ''
  editor.nextAction = ''
  editorVisible.value = true
}

function openEdit(item: Reflection): void {
  editor.id = item.id
  editor.sessionId = item.sessionId || ''
  editor.content = item.content
  editor.plan = item.plan
  editor.nextAction = item.nextAction
  editorVisible.value = true
}

async function submitReflection(): Promise<void> {
  if (!editor.content.trim() && !editor.plan.trim() && !editor.nextAction.trim()) {
    ElMessage.error('请至少填写一项内容')
    return
  }
  editorSubmitting.value = true
  try {
    const payload = {
      sessionId: editor.sessionId === '' ? undefined : editor.sessionId,
      content: editor.content,
      plan: editor.plan,
      nextAction: editor.nextAction
    }
    if (editor.id === null) {
      await createReflectionApi(payload)
      ElMessage.success('反思已保存')
    } else {
      await updateReflectionApi(editor.id, payload)
      ElMessage.success('反思已更新')
    }
    editorVisible.value = false
    await loadReflections()
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    editorSubmitting.value = false
  }
}

/* ===== 待改进项：新建 / 编辑 / 删除 / 自评完成 ===== */
const sessionOptions = ref<TeacherTimelineItem[]>([])
const improveEditorVisible = ref(false)
const improveSubmitting = ref(false)
const improveEditor = reactive<{ id: number | null; title: string; sessionId: number | '' }>({
  id: null,
  title: '',
  sessionId: ''
})

const SOURCE_LABELS: Record<string, string> = {
  ai: 'AI 观察',
  supervisor: '督导观察',
  self: '自建'
}

function sourceLabel(item: ImprovementItem): string {
  return SOURCE_LABELS[item.source] ?? '自建'
}

function statusLabel(item: ImprovementItem): string {
  return item.status === 'closed' ? '已完成' : '进行中'
}

async function loadSessionOptions(): Promise<void> {
  if (!isTeacher.value || !auth.user?.id) return
  try {
    const page = await fetchTeacherEvaluationsApi(auth.user.id, { pageSize: 50 })
    sessionOptions.value = page.list ?? []
  } catch {
    sessionOptions.value = []
  }
}

function sessionLabel(item: TeacherTimelineItem): string {
  return `${item.sessionDate} ${item.courseName}${item.topic ? `，${item.topic}` : ''}`
}

function openImproveCreate(): void {
  improveEditor.id = null
  improveEditor.title = ''
  improveEditor.sessionId = ''
  improveEditorVisible.value = true
}

function openImproveEdit(item: ImprovementItem): void {
  improveEditor.id = item.id
  improveEditor.title = item.title
  improveEditor.sessionId = item.sessionId || ''
  improveEditorVisible.value = true
}

async function submitImprovement(): Promise<void> {
  const title = improveEditor.title.trim()
  if (!title) {
    ElMessage.error('请填写改进项标题')
    return
  }
  improveSubmitting.value = true
  try {
    if (improveEditor.id === null) {
      await createImprovementApi({
        title,
        sessionId: improveEditor.sessionId === '' ? undefined : improveEditor.sessionId,
        source: 'self'
      })
      ElMessage.success('改进项已创建')
    } else {
      await updateImprovementApi(improveEditor.id, title)
      ElMessage.success('改进项已更新')
    }
    improveEditorVisible.value = false
    await loadImprovements()
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    improveSubmitting.value = false
  }
}

const completeVisible = ref(false)
const completeSubmitting = ref(false)
const completeNote = ref('')
const completeTarget = ref<ImprovementItem | null>(null)

function openComplete(item: ImprovementItem): void {
  completeTarget.value = item
  completeNote.value = item.doneNote || ''
  completeVisible.value = true
}

async function submitComplete(): Promise<void> {
  if (!completeTarget.value) return
  completeSubmitting.value = true
  try {
    await completeImprovementApi(completeTarget.value.id, completeNote.value.trim())
    ElMessage.success('已完成，改进项归档')
    completeVisible.value = false
    await loadImprovements()
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    completeSubmitting.value = false
  }
}

async function removeImprovement(item: ImprovementItem): Promise<void> {
  try {
    await ElMessageBox.confirm(`确认删除改进项「${item.title}」？删除后不可恢复。`, '删除改进项', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    await deleteImprovementApi(item.id)
    ElMessage.success('改进项已删除')
    await loadImprovements()
  } catch {
    // 错误提示由 http 拦截器统一处理
  }
}

onMounted(() => {
  void loadReflections()
  void loadImprovements()
  void loadSessionOptions()
})
</script>

<template>
  <div class="reflection">
    <PageHeader title="教学反思" subtitle="左侧沉淀反思，右侧把反思落成可执行的改进项，仅本人可见">
      <template #actions>
        <el-button v-if="isTeacher" type="primary" @click="openCreate">写一条反思</el-button>
      </template>
    </PageHeader>

    <div class="reflection__layout">
      <!-- 左 3/5：反思列表 -->
      <section class="reflection__main" aria-label="教学反思列表">
        <header class="reflection__section-head">
          <h2 class="reflection__section-title">反思记录</h2>
          <span class="reflection__section-count tabular-nums">{{ items.length }} 条</span>
        </header>

        <el-skeleton v-if="loading" :rows="6" animated />

        <EmptyState v-else-if="error" description="反思加载失败">
          <template #action>
            <el-button @click="loadReflections">重新加载</el-button>
          </template>
        </EmptyState>

        <EmptyState v-else-if="!items.length" description="还没有教学反思">
          <template #action>
            <el-button v-if="isTeacher" @click="openCreate">写第一条</el-button>
          </template>
        </EmptyState>

        <div v-else class="reflection__list">
          <article
            v-for="(item, index) in items"
            :key="item.id"
            class="reflection__card anim-rise"
            :style="{ '--stagger': `${Math.min(index, 8) * 50}ms` }"
          >
            <header class="reflection__head">
              <span class="reflection__date tabular-nums">{{ item.createdAt.slice(0, 10) }}</span>
              <el-tag v-if="item.sessionId" size="small" effect="plain" round>课次 #{{ item.sessionId }}</el-tag>
              <el-button v-if="isTeacher" text type="primary" class="reflection__edit" @click="openEdit(item)">
                编辑
              </el-button>
            </header>
            <section v-if="item.content" class="reflection__block">
              <h3 class="reflection__block-title">反思</h3>
              <p class="reflection__text">{{ item.content }}</p>
            </section>
            <section v-if="item.plan" class="reflection__block">
              <h3 class="reflection__block-title">改进计划</h3>
              <p class="reflection__text">{{ item.plan }}</p>
            </section>
            <section v-if="item.nextAction" class="reflection__block">
              <h3 class="reflection__block-title">下节行动</h3>
              <p class="reflection__text">{{ item.nextAction }}</p>
            </section>
          </article>
        </div>
      </section>

      <!-- 右 2/5：待改进项 -->
      <aside class="improve-panel" aria-label="待改进项">
        <header class="improve-panel__head">
          <div>
            <h2 class="improve-panel__title">待改进项</h2>
            <p class="improve-panel__sub">进行中 {{ openImprovements.length }} 项，已完成 {{ closedImprovements.length }} 项</p>
          </div>
          <el-button v-if="isTeacher" plain size="small" @click="openImproveCreate">新建</el-button>
        </header>

        <el-skeleton v-if="improvementsLoading" :rows="4" animated />

        <EmptyState v-else-if="improvementsError" description="改进项加载失败">
          <template #action>
            <el-button @click="loadImprovements">重新加载</el-button>
          </template>
        </EmptyState>

        <p v-else-if="!improvements.length" class="improve-panel__empty">暂无待改进项。</p>

        <template v-else>
          <ul class="improve-list">
            <li
              v-for="item in openImprovements"
              :key="item.id"
              class="improve-item"
              :class="{ 'improve-item--done': item.status === 'closed' }"
            >
              <span class="improve-item__dot" aria-hidden="true" />
              <div class="improve-item__main">
                <p class="improve-item__title">{{ item.title }}</p>
                <p class="improve-item__meta">
                  <span>{{ sourceLabel(item) }}</span>
                  <span v-if="item.sessionId">课次 #{{ item.sessionId }}</span>
                  <span>{{ statusLabel(item) }}</span>
                </p>
                <p v-if="item.doneNote" class="improve-item__note">{{ item.doneNote }}</p>
                <div class="improve-item__actions">
                  <el-button v-if="isTeacher" text type="primary" size="small" @click="openImproveEdit(item)">
                    编辑
                  </el-button>
                  <el-button v-if="isTeacher" text type="success" size="small" @click="openComplete(item)">
                    自评完成
                  </el-button>
                  <el-button v-if="isTeacher" text type="danger" size="small" @click="removeImprovement(item)">
                    删除
                  </el-button>
                </div>
              </div>
            </li>
          </ul>

          <details v-if="closedImprovements.length" class="improve-closed">
            <summary class="improve-closed__summary">已完成 {{ closedImprovements.length }} 项</summary>
            <ul class="improve-list">
              <li v-for="item in closedImprovements" :key="item.id" class="improve-item improve-item--done">
                <span class="improve-item__dot" aria-hidden="true" />
                <div class="improve-item__main">
                  <p class="improve-item__title">{{ item.title }}</p>
                  <p class="improve-item__meta">{{ sourceLabel(item) }}，已完成</p>
                  <div class="improve-item__actions">
                    <el-button v-if="isTeacher" text type="danger" size="small" @click="removeImprovement(item)">
                      删除
                    </el-button>
                  </div>
                </div>
              </li>
            </ul>
          </details>
        </template>
      </aside>
    </div>

    <!-- 反思编辑器 -->
    <el-dialog v-model="editorVisible" :title="editor.id === null ? '写一条反思' : '编辑反思'" width="560px">
      <el-form label-width="80px">
        <el-form-item label="关联课次">
          <el-input v-model.number="editor.sessionId" placeholder="授课记录 id（选填）" />
        </el-form-item>
        <el-form-item label="反思">
          <el-input
            v-model="editor.content"
            type="textarea"
            :rows="3"
            maxlength="4000"
            show-word-limit
            placeholder="这次课哪些地方做得好、哪些需要改进"
          />
        </el-form-item>
        <el-form-item label="改进计划">
          <el-input
            v-model="editor.plan"
            type="textarea"
            :rows="2"
            maxlength="2000"
            show-word-limit
            placeholder="打算怎么改进"
          />
        </el-form-item>
        <el-form-item label="下节行动">
          <el-input
            v-model="editor.nextAction"
            type="textarea"
            :rows="2"
            maxlength="2000"
            show-word-limit
            placeholder="下节课具体做的一件事"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="editorSubmitting" @click="submitReflection">保存</el-button>
      </template>
    </el-dialog>

    <!-- 改进项编辑器 -->
    <el-dialog
      v-model="improveEditorVisible"
      :title="improveEditor.id === null ? '新建改进项' : '编辑改进项'"
      width="480px"
    >
      <el-form label-width="72px">
        <el-form-item label="标题" required>
          <el-input v-model="improveEditor.title" maxlength="255" placeholder="一句话描述要改进的事" />
        </el-form-item>
        <el-form-item label="关联课次">
          <el-select v-model="improveEditor.sessionId" clearable filterable placeholder="选填" class="reflection__field">
            <el-option
              v-for="session in sessionOptions"
              :key="session.sessionId"
              :label="sessionLabel(session)"
              :value="session.sessionId"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="improveEditorVisible = false">取消</el-button>
        <el-button type="primary" :loading="improveSubmitting" @click="submitImprovement">保存</el-button>
      </template>
    </el-dialog>

    <!-- 自评完成 -->
    <el-dialog v-model="completeVisible" title="自评完成" width="460px">
      <p class="reflection__complete-target">{{ completeTarget?.title }}</p>
      <el-input
        v-model="completeNote"
        type="textarea"
        :rows="3"
        maxlength="1000"
        show-word-limit
        placeholder="记录完成情况（选填）"
      />
      <template #footer>
        <el-button @click="completeVisible = false">取消</el-button>
        <el-button type="primary" :loading="completeSubmitting" @click="submitComplete">标记完成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.reflection {
  &__layout {
    display: grid;
    grid-template-columns: 3fr 2fr;
    gap: var(--spacing-4);
    align-items: start;

    @media (max-width: 1100px) {
      grid-template-columns: 1fr;
    }
  }

  &__main {
    min-width: 0;
  }

  &__section-head {
    display: flex;
    gap: var(--spacing-3);
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: var(--spacing-3);
  }

  &__section-title {
    font-size: var(--font-size-lg);
  }

  &__section-count {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
  }

  &__card {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
    padding: var(--spacing-5);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
    transition: box-shadow var(--duration-fast) var(--ease-out-soft), transform var(--duration-fast) var(--ease-out-soft);

    &:hover {
      box-shadow: var(--shadow-card-hover);
      transform: translateY(-1px);
    }
  }

  &__head {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    padding-bottom: var(--spacing-2);
    border-bottom: 1px dashed var(--color-divider);
  }

  &__date {
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__edit {
    margin-left: auto;
  }

  &__block-title {
    margin-bottom: var(--spacing-1);
    font-size: var(--font-size-xs);
    font-weight: 600;
    letter-spacing: 0.04em;
    color: var(--color-ai);
  }

  &__text {
    font-size: var(--font-size-sm);
    line-height: 1.7;
    color: var(--color-text-secondary);
    white-space: pre-wrap;
  }

  &__field {
    width: 100%;
  }

  &__complete-target {
    padding: var(--spacing-2) var(--spacing-3);
    margin-bottom: var(--spacing-3);
    font-size: var(--font-size-sm);
    color: var(--color-text-primary);
    background-color: var(--color-bg-page);
    border-radius: var(--radius-md);
  }
}

.improve-panel {
  position: sticky;
  top: calc(var(--header-height) + var(--spacing-4));
  padding: var(--spacing-5);
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);

  @media (max-width: 1100px) {
    position: static;
  }

  &__head {
    display: flex;
    gap: var(--spacing-3);
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__title {
    font-size: var(--font-size-lg);
  }

  &__sub {
    margin-top: 2px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__empty {
    padding: var(--spacing-4) 0;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }
}

.improve-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.improve-item {
  display: flex;
  gap: var(--spacing-3);
  padding: var(--spacing-3) var(--spacing-3) var(--spacing-2);
  background-color: var(--color-bg-page);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-md);

  &__dot {
    flex-shrink: 0;
    width: 8px;
    height: 8px;
    margin-top: 7px;
    background-color: var(--color-warning);
    border-radius: 50%;
  }

  &--done .improve-item__dot {
    background-color: var(--color-success);
  }

  &__main {
    flex: 1;
    min-width: 0;
  }

  &__title {
    font-size: var(--font-size-sm);
    font-weight: 600;
    line-height: 1.5;
    color: var(--color-text-primary);
  }

  &__meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0 var(--spacing-2);
    margin-top: 2px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__note {
    padding: var(--spacing-2);
    margin-top: var(--spacing-2);
    font-size: var(--font-size-xs);
    line-height: 1.6;
    color: var(--color-text-secondary);
    background-color: var(--color-bg-card);
    border-radius: var(--radius-sm);
  }

  &__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-1);
    margin-top: var(--spacing-1);
  }
}

.improve-closed {
  margin-top: var(--spacing-4);
  border-top: 1px dashed var(--color-divider);

  &__summary {
    padding-top: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    cursor: pointer;
  }
}
</style>
