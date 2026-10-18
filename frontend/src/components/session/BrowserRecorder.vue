<script setup lang="ts">
/**
 * BrowserRecorder 浏览器录音原型（R2.4 / B5）。
 *
 * 本轮只做**前端原型占位**：无业务功能、不调用任何接口、不产生真实课堂录音。
 * 真实录音走「补传文件上传」入口；演示环境不产生真实课堂录音。
 */
</script>

<template>
  <div class="browser-recorder ai-panel">
    <div class="browser-recorder__head">
      <span class="browser-recorder__dot" aria-hidden="true" />
      <span class="browser-recorder__title">浏览器录音</span>
      <el-tag size="small" type="info" effect="plain">演示占位</el-tag>
    </div>
    <p class="browser-recorder__hint">
      浏览器录音仅作交互原型，本轮不接入业务功能；真实课堂录音请使用「补传文件上传」。
    </p>
    <div class="browser-recorder__actions">
      <el-button disabled size="small">开始录音</el-button>
      <el-button disabled size="small">停止并转写</el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.browser-recorder { padding: var(--spacing-4); }
.browser-recorder__head { display: flex; align-items: center; gap: var(--spacing-2); margin-bottom: var(--spacing-2); }
.browser-recorder__dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-text-disabled); }
.browser-recorder__title { font-size: var(--font-size-base); color: var(--color-text-primary); }
.browser-recorder__hint { margin: 0 0 var(--spacing-3); font-size: var(--font-size-sm); color: var(--color-text-tertiary); }
.browser-recorder__actions { display: flex; gap: var(--spacing-2); }
</style>
