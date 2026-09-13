<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="error-page">
    <div class="error-page__code">404</div>
    <h1 class="error-page__title">页面不存在</h1>
    <p class="error-page__desc">您访问的页面可能已被移动或删除。</p>
    <el-button type="primary" @click="router.replace({ name: 'dashboard' })">
      返回工作台
    </el-button>
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
    color: var(--color-primary);
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
