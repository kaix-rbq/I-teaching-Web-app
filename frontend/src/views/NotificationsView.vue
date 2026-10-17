<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowRight } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { fetchNotificationsApi, markAllNotificationsReadApi, markNotificationReadApi } from '@/api/notification'
import type { NotificationItem } from '@/types/improvement'

/**
 * 站内信（B12）：只推督导评语 / AI 发现问题；进入详情自动已读。
 * 红点收敛到工作台清单与督导任务队列两处，本页承载完整列表。
 */
const router = useRouter()

const loading = ref(true)
const error = ref(false)
const items = ref<NotificationItem[]>([])
const unread = ref(0)
const acting = ref(false)

const hasUnread = computed(() => unread.value > 0)

async function load(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    const page = await fetchNotificationsApi({ pageSize: 50 })
    items.value = page.list ?? []
    unread.value = page.unread ?? 0
  } catch {
    error.value = true
    items.value = []
  } finally {
    loading.value = false
  }
}

async function openItem(item: NotificationItem): Promise<void> {
  if (!item.isRead) {
    try {
      await markNotificationReadApi(item.id)
      item.isRead = true
      unread.value = Math.max(0, unread.value - 1)
    } catch {
      // ignore
    }
  }
  if (item.refId) {
    void router.push({ name: 'session-evaluation', params: { id: item.refId } })
  }
}

async function readAll(): Promise<void> {
  acting.value = true
  try {
    await markAllNotificationsReadApi()
    items.value = items.value.map((it) => ({ ...it, isRead: true }))
    unread.value = 0
    ElMessage.success('已全部标为已读')
  } catch {
    // ignore
  } finally {
    acting.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="notice">
    <PageHeader title="站内信" subtitle="督导评语与 AI 发现的问题会推送到这里">
      <template #actions>
        <el-button :disabled="!hasUnread" :loading="acting" @click="readAll">全部标为已读</el-button>
      </template>
    </PageHeader>

    <el-skeleton v-if="loading" :rows="6" animated />

    <EmptyState v-else-if="error" description="站内信加载失败">
      <template #action>
        <el-button type="primary" @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <EmptyState v-else-if="!items.length" description="暂无站内信" />

    <ul v-else class="notice__list">
      <li
        v-for="(item, index) in items"
        :key="item.id"
        class="notice__item anim-rise"
        :class="{ 'notice__item--unread': !item.isRead }"
        :style="{ '--stagger': `${Math.min(index, 10) * 40}ms` }"
        @click="openItem(item)"
      >
        <span class="notice__dot" :class="{ 'notice__dot--on': !item.isRead }" aria-hidden="true" />
        <div class="notice__body">
          <div class="notice__head">
            <el-tag size="small" :type="item.type === 'ai_issue' ? 'warning' : 'primary'" effect="plain" round>
              {{ item.type === 'ai_issue' ? 'AI' : '督导' }}
            </el-tag>
            <span class="notice__title">{{ item.title }}</span>
          </div>
          <p class="notice__meta tabular-nums">{{ item.createdAt.slice(0, 16).replace('T', ' ') }}</p>
        </div>
        <el-icon class="notice__arrow"><ArrowRight /></el-icon>
      </li>
    </ul>
  </div>
</template>

<style scoped lang="scss">
.notice {
  &__list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-2);
  }

  &__item {
    display: flex;
    gap: var(--spacing-3);
    align-items: center;
    padding: var(--spacing-3) var(--spacing-4);
    cursor: pointer;
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-md);
    transition: box-shadow var(--duration-fast) var(--ease-out-soft), border-color var(--duration-fast) var(--ease-out-soft);

    &:hover {
      border-color: var(--color-primary-hover);
      box-shadow: var(--shadow-card-hover);
    }

    &--unread {
      background-color: var(--color-primary-bg);
    }
  }

  &__dot {
    width: 8px;
    height: 8px;
    background-color: var(--color-border);
    border-radius: 50%;

    &--on {
      background-color: var(--color-error);
      animation: dot-pulse 2.2s var(--ease-out-soft) infinite;
    }
  }

  &__body { flex: 1; min-width: 0; }

  &__head {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
  }

  &__title {
    overflow: hidden;
    font-size: var(--font-size-base);
    color: var(--color-text-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__meta {
    margin-top: 2px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__arrow { color: var(--color-text-disabled); }
}
</style>
