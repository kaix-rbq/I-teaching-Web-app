<script setup lang="ts">
/**
 * AI 徽标：一切智能体产出内容的「准入证」——四芒星 + 渐变紫 + 文字徽章。
 * 规范见 docs/前端设计-new.md §4.5：AI 内容必带徽标，与人类督导产出形成语法对比。
 */
withDefaults(defineProps<{ text?: string }>(), { text: 'AI 参考' })
</script>

<template>
  <span class="ai-badge">
    <svg class="ai-badge__star" viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M12 2l2.4 7.6L22 12l-7.6 2.4L12 22l-2.4-7.6L2 12l7.6-2.4L12 2z"
        fill="currentColor"
      />
    </svg>
    <span class="ai-badge__text">{{ text }}</span>
  </span>
</template>

<style scoped lang="scss">
.ai-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--gradient-ai);
  color: #fff;
  font-size: var(--font-size-xs);
  font-weight: 600;
  line-height: 1.4;
  white-space: nowrap;
}

.ai-badge__star {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
}

.ai-badge__text {
  letter-spacing: 0.5px;
}
</style>
