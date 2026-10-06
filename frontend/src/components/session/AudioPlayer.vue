<script setup lang="ts">
/**
 * 课堂录音播放器。
 *
 * src 应为带短时票据的播放地址（RecordingDTO.playbackUrl）：<audio> 发出的 Range 请求
 * 无法携带 Authorization 头，因此服务端为流式路由额外接受 ?ticket= 查询串鉴权。
 *
 * 不再使用 fetch + Blob 的旧实现：那样必须整包下载完才能播放，等于废掉 Range，
 * 45 分钟音频会出现长时间空白且内存峰值很高。
 */
defineProps<{ src: string; name?: string }>()
</script>

<template>
  <div class="audio-player">
    <span class="audio-player__name">{{ name || '课堂录音' }}</span>
    <audio v-if="src" :src="src" controls preload="metadata" />
  </div>
</template>

<style scoped lang="scss">
.audio-player { display:flex; flex-direction:column; gap:var(--spacing-2); }
.audio-player__name { font-size:var(--font-size-sm); color:var(--color-text-secondary); }
audio { width:100%; }
</style>
