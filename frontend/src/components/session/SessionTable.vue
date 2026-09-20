<script setup lang="ts">
import { getSessionStatusMeta } from '@/constants'
import { formatDate } from '@/utils/format'
import type { SessionListItem } from '@/types/evaluation'

withDefaults(
  defineProps<{
    rows: SessionListItem[]
    loading?: boolean
  }>(),
  {
    loading: false
  }
)

const emit = defineEmits<{ (e: 'view', session: SessionListItem): void }>()

/** 缺失分一律渲染 —，禁止显示 0（前端 AGENTS §9.6） */
function formatScore(score: number | null): string {
  return score === null || score === undefined ? '—' : score.toFixed(2)
}

function handleRowClick(row: SessionListItem): void {
  emit('view', row)
}

function asSession(row: unknown): SessionListItem {
  return row as SessionListItem
}
</script>

<template>
  <el-skeleton v-if="loading" :rows="5" animated />
  <el-table
    v-else
    class="session-table"
    :data="rows"
    row-key="id"
    stripe
    @row-click="handleRowClick"
  >
    <el-table-column label="日期" width="130">
      <template #default="{ row }">{{ formatDate(row.sessionDate) }}</template>
    </el-table-column>
    <el-table-column prop="period" label="节次" width="100" />
    <el-table-column label="主题" min-width="200">
      <template #default="{ row }">
        <span class="session-table__topic">{{ row.topic || '—' }}</span>
      </template>
    </el-table-column>
    <el-table-column label="状态" width="100" align="center">
      <template #default="{ row }">
        <el-tag :type="getSessionStatusMeta(row.status).tag" effect="light" round>
          {{ getSessionStatusMeta(row.status).label }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column label="督导分" width="100" align="right">
      <template #default="{ row }">
        <span class="tabular-nums">{{ formatScore(row.supervisorScore) }}</span>
      </template>
    </el-table-column>
    <el-table-column label="智能体分" width="110" align="right">
      <template #default="{ row }">
        <span class="tabular-nums session-table__agent">
          {{ formatScore(row.agentScore) }}
        </span>
      </template>
    </el-table-column>
    <el-table-column label="已评次数" width="100" align="center">
      <template #default="{ row }">
        <span class="tabular-nums">{{ row.evaluationCount }}</span>
      </template>
    </el-table-column>
    <el-table-column label="操作" width="120" align="center">
      <template #default="{ row }">
        <el-button link type="primary" @click.stop="handleRowClick(asSession(row))">查看评估</el-button>
      </template>
    </el-table-column>
    <template #empty>
      <span class="session-table__empty">暂无授课记录</span>
    </template>
  </el-table>
</template>

<style scoped lang="scss">
.session-table {
  width: 100%;

  &__topic {
    font-weight: 500;
    color: var(--color-text-primary);
  }

  &__agent {
    color: var(--color-text-tertiary);
  }

  &__empty {
    color: var(--color-text-tertiary);
  }

  :deep(.el-table__row) {
    cursor: pointer;
  }
}
</style>
