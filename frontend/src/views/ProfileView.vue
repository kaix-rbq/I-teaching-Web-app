<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import RoleTag from '@/components/common/RoleTag.vue'
import { useAuthStore } from '@/stores/auth'
import { getRoleMeta } from '@/constants'

const router = useRouter()
const auth = useAuthStore()

const avatarText = computed(() => auth.user?.name?.slice(0, 1) ?? '爱')
const roleLabel = computed(() => getRoleMeta(auth.user?.role)?.label ?? '-')

const infoItems = computed(() => [
  { label: '姓名', value: auth.user?.name ?? '-' },
  { label: '角色', value: roleLabel.value },
  { label: '所属部门', value: auth.user?.department ?? '-' },
  { label: '工号', value: auth.user?.jobNo ?? '-' },
  { label: '账号', value: auth.user?.jobNo ?? '-' }
])

async function handleLogout(): Promise<void> {
  await auth.logout()
  void router.replace({ name: 'login' })
}
</script>

<template>
  <div class="profile">
    <PageHeader title="个人中心" subtitle="查看个人账号信息" />

    <div class="profile__card">
      <div class="profile__header">
        <span class="profile__avatar">{{ avatarText }}</span>
        <div class="profile__identity">
          <div class="profile__name-line">
            <span class="profile__name">{{ auth.user?.name }}</span>
            <RoleTag :role="auth.user?.role" />
          </div>
          <span class="profile__department">{{ auth.user?.department ?? '-' }}</span>
        </div>
      </div>

      <el-descriptions :column="2" border class="profile__descriptions">
        <el-descriptions-item v-for="item in infoItems" :key="item.label" :label="item.label">
          {{ item.value }}
        </el-descriptions-item>
      </el-descriptions>

      <div class="profile__actions">
        <el-button type="danger" plain @click="handleLogout">退出登录</el-button>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.profile {
  &__card {
    max-width: 720px;
    padding: var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  &__header {
    display: flex;
    align-items: center;
    gap: var(--spacing-4);
    padding-bottom: var(--spacing-6);
    margin-bottom: var(--spacing-6);
    border-bottom: 1px solid var(--color-divider);
  }

  &__avatar {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 64px;
    height: 64px;
    font-size: var(--font-size-3xl);
    font-weight: 600;
    color: var(--color-bg-card);
    background-color: var(--color-primary);
    border-radius: 50%;
  }

  &__identity {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-1);
  }

  &__name-line {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
  }

  &__name {
    font-size: var(--font-size-2xl);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__department {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__actions {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--spacing-6);
  }
}
</style>
