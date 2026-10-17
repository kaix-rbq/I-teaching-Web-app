<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { createReflectionApi, fetchReflectionsApi, updateReflectionApi } from '@/api/reflection'
import { useAuthStore } from '@/stores/auth'
import type { Reflection } from '@/types/improvement'

/**
 * 教学反思（N5，教师私有）：收藏某条评价 + 反思 / 计划 / 下节行动。
 * 督导仅在关闭改进项时可读关联反思；本页仅教师可用。
 */
const auth = useAuthStore()
const isTeacher = computed(() => auth.role === 'teacher')

const loading = ref(true)
const error = ref(false)
const items = ref<Reflection[]>([])

async function load(): Promise<void> {
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

async function submit(): Promise<void> {
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
    await load()
  } catch {
    // ignore
  } finally {
    editorSubmitting.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="reflection">
    <PageHeader title="教学反思" subtitle="把每次课的思考沉淀为可执行的改进计划 · 仅本人可见">
      <template #actions>
        <el-button v-if="isTeacher" type="primary" @click="openCreate">写一条反思</el-button>
      </template>
    </PageHeader>

    <el-skeleton v-if="loading" :rows="6" animated />

    <EmptyState v-else-if="error" description="反思加载失败">
      <template #action>
        <el-button type="primary" @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <EmptyState v-else-if="!items.length" description="还没有教学反思">
      <template #action>
        <el-button v-if="isTeacher" type="primary" @click="openCreate">写第一条</el-button>
      </template>
    </EmptyState>

    <div v-else class="reflection__grid">
      <article
        v-for="(item, index) in items"
        :key="item.id"
        class="reflection__card anim-rise"
        :style="{ '--stagger': `${Math.min(index, 8) * 50}ms` }"
      >
        <header class="reflection__head">
          <span class="reflection__date tabular-nums">{{ item.createdAt.slice(0, 10) }}</span>
          <el-tag v-if="item.sessionId" size="small" effect="plain" round>课次 #{{ item.sessionId }}</el-tag>
          <el-button v-if="isTeacher" text type="primary" class="reflection__edit" @click="openEdit(item)">编辑</el-button>
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

    <el-dialog v-model="editorVisible" :title="editor.id === null ? '写一条反思' : '编辑反思'" width="560px">
      <el-form label-width="80px">
        <el-form-item label="关联课次">
          <el-input v-model.number="editor.sessionId" placeholder="授课记录 id（选填）" />
        </el-form-item>
        <el-form-item label="反思">
          <el-input v-model="editor.content" type="textarea" :rows="3" maxlength="4000" show-word-limit placeholder="这次课哪些地方做得好、哪些需要改进" />
        </el-form-item>
        <el-form-item label="改进计划">
          <el-input v-model="editor.plan" type="textarea" :rows="2" maxlength="2000" show-word-limit placeholder="打算怎么改进" />
        </el-form-item>
        <el-form-item label="下节行动">
          <el-input v-model="editor.nextAction" type="textarea" :rows="2" maxlength="2000" show-word-limit placeholder="下节课具体做的一件事" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="editorSubmitting" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.reflection {
  &__grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--spacing-4);

    @media (max-width: 1024px) { grid-template-columns: 1fr; }
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

    &:hover { box-shadow: var(--shadow-card-hover); transform: translateY(-1px); }
  }

  &__head {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    padding-bottom: var(--spacing-2);
    border-bottom: 1px dashed var(--color-divider);
  }

  &__date { font-size: var(--font-size-sm); font-weight: 600; color: var(--color-text-primary); }
  &__edit { margin-left: auto; }

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
  }
}
</style>
