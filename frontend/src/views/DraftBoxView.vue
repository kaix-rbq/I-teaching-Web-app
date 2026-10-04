<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import FilterBar from '@/components/common/FilterBar.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { deleteDraftApi, fetchDraftsApi, submitDraftApi } from '@/api/draft'
import { PAGE_SIZES } from '@/constants'
import type { DraftDTO } from '@/types/draft'

/**
 * 草稿箱（督导）：所有未提交的评估草稿。
 * 支持按课程名查询、编辑（回到评估页继续写）、提交（五维齐全后转正式评价）与删除。
 */
const router = useRouter()

const drafts = ref<DraftDTO[]>([])
const loading = ref(true)
const error = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')
const submittingId = ref<number | null>(null)

async function loadDrafts(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    const result = await fetchDraftsApi({
      keyword: keyword.value.trim() || undefined,
      page: page.value,
      pageSize: pageSize.value
    })
    drafts.value = result.list
    total.value = result.total
  } catch {
    error.value = true
    drafts.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function search(): void {
  page.value = 1
  void loadDrafts()
}

function reset(): void {
  keyword.value = ''
  page.value = 1
  void loadDrafts()
}

function handlePageChange(value: number): void {
  page.value = value
  void loadDrafts()
}

function handlePageSizeChange(value: number): void {
  pageSize.value = value
  page.value = 1
  void loadDrafts()
}

function editDraft(draft: DraftDTO): void {
  void router.push({
    name: 'session-evaluation',
    params: { id: draft.sessionId },
    query: { draft: '1' }
  })
}

async function submitDraft(draft: DraftDTO): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `确认提交「${draft.courseName}」的草稿为正式评价吗？提交后草稿将从草稿箱移除。`,
      '提交确认',
      { type: 'info', confirmButtonText: '提交', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  submittingId.value = draft.id
  try {
    await submitDraftApi(draft.id)
    ElMessage.success('草稿已提交为正式评价')
    await loadDrafts()
  } catch {
    // 维度不全等业务错误由拦截器提示；草稿保留以便继续编辑
  } finally {
    submittingId.value = null
  }
}

async function removeDraft(draft: DraftDTO): Promise<void> {
  try {
    await ElMessageBox.confirm('是否确认删除所选草稿？', '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    await deleteDraftApi(draft.id)
    ElMessage.success('草稿已删除')
    await loadDrafts()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  }
}

onMounted(() => {
  void loadDrafts()
})
</script>

<template>
  <div class="draft-box">
    <PageHeader title="草稿箱" subtitle="未提交的评估草稿 · 可编辑、提交或删除" />

    <section class="draft-box__panel">
      <FilterBar @search="search" @reset="reset">
        <el-input
          v-model="keyword"
          placeholder="按课程名称查询"
          clearable
          class="draft-box__search"
          @keyup.enter="search"
        />
      </FilterBar>

      <div v-if="error" class="draft-box__state">
        <EmptyState description="草稿加载失败">
          <template #action>
            <el-button type="primary" @click="loadDrafts">重新加载</el-button>
          </template>
        </EmptyState>
      </div>

      <template v-else>
        <el-table v-loading="loading" :data="drafts" row-key="id" stripe>
          <el-table-column prop="courseName" label="课程名称" min-width="200">
            <template #default="{ row }">
              <span class="draft-box__course">{{ row.courseName }}</span>
              <el-tag v-if="row.courseCode" size="small" effect="plain" class="draft-box__code">
                {{ row.courseCode }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="teacherName" label="授课教师" min-width="120" />
          <el-table-column label="授课日期" min-width="120">
            <template #default="{ row }">
              <span class="tabular-nums">{{ row.sessionDate }}</span>
            </template>
          </el-table-column>
          <el-table-column label="节次" min-width="100">
            <template #default="{ row }">{{ row.period || '—' }}</template>
          </el-table-column>
          <el-table-column label="主题" min-width="160">
            <template #default="{ row }">{{ row.topic || '—' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="editDraft(row as DraftDTO)">编辑</el-button>
              <el-button
                link
                type="success"
                :loading="submittingId === (row as DraftDTO).id"
                @click="submitDraft(row as DraftDTO)"
              >
                提交
              </el-button>
              <el-button link type="danger" @click="removeDraft(row as DraftDTO)">删除</el-button>
            </template>
          </el-table-column>
          <template #empty>
            <EmptyState description="暂无草稿" />
          </template>
        </el-table>

        <div class="draft-box__pagination">
          <el-pagination
            :current-page="page"
            :page-size="pageSize"
            :page-sizes="PAGE_SIZES"
            :total="total"
            layout="total, sizes, prev, pager, next"
            background
            @current-change="handlePageChange"
            @size-change="handlePageSizeChange"
          />
        </div>
      </template>
    </section>
  </div>
</template>

<style scoped lang="scss">
.draft-box {
  &__panel {
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__search {
    width: 260px;
  }

  &__course {
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__code {
    margin-left: var(--spacing-2);
  }

  &__state {
    padding: var(--spacing-6);
  }

  &__pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--spacing-4);
  }
}
</style>
