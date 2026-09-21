<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { TOKEN_KEY } from '@/constants'

const props = defineProps<{ src: string; name?: string }>()
const playableSrc = ref('')
let objectURL = ''
async function load(): Promise<void> {
  if (!props.src) return
  const response = await fetch(props.src, { headers: { Authorization: `Bearer ${localStorage.getItem(TOKEN_KEY) || ''}` } })
  if (!response.ok) return
  if (objectURL) URL.revokeObjectURL(objectURL)
  objectURL = URL.createObjectURL(await response.blob())
  playableSrc.value = objectURL
}
watch(() => props.src, () => { void load() }, { immediate: true })
onBeforeUnmount(() => { if (objectURL) URL.revokeObjectURL(objectURL) })
</script>

<template>
  <div class="audio-player">
    <span class="audio-player__name">{{ name || '课堂录音' }}</span>
    <audio :src="playableSrc" controls preload="metadata" />
  </div>
</template>

<style scoped lang="scss">
.audio-player { display:flex; flex-direction:column; gap:var(--spacing-2); }
.audio-player__name { font-size:var(--font-size-sm); color:var(--color-text-secondary); }
audio { width:100%; }
</style>
