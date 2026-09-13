<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

function goBack(): void {
  if (auth.isLoggedIn) {
    void router.replace({ name: 'dashboard' })
  } else {
    void router.replace({ name: 'login' })
  }
}
</script>

<template>
  <div class="error-page">
    <div class="error-page__code">403</div>
    <h1 class="error-page__title">无权访问</h1>
    <p class="error-page__desc">当前账号角色没有访问该页面的权限。</p>
    <el-button type="primary" @click="goBack">返回工作台</el-button>
  </div>
</template>

<style scoped lang="scss">
.error-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  text-align: center;

  &__code {
    font-size: 88px;
    font-weight: 600;
    line-height: 1;
    color: var(--color-warning);
    font-variant-numeric: tabular-nums;
  }

  &__title {
    margin-top: var(--spacing-4);
    font-size: var(--font-size-3xl);
    color: var(--color-text-primary);
  }

  &__desc {
    margin: var(--spacing-2) 0 var(--spacing-6);
    color: var(--color-text-tertiary);
  }
}
</style>
