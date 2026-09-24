import { ref } from 'vue'
import { CURRENT_SEMESTER, SEMESTERS } from '@/constants'

/**
 * 全局学期口径（评价数据的切片维度）。
 * 顶栏学期选择器写入，驾驶舱 / 画像 / 提优页消费；localStorage 持久化。
 * 以 composable 而非 store 提供（遵守 stores 仅 auth/dict 的既有约定）。
 */
const STORAGE_KEY = 'aijiaoxue_semester'

function readInitial(): string {
  const saved = typeof localStorage === 'undefined' ? null : localStorage.getItem(STORAGE_KEY)
  return saved && SEMESTERS.includes(saved) ? saved : CURRENT_SEMESTER
}

const semester = ref<string>(readInitial())

export function useSemester() {
  function setSemester(value: string): void {
    semester.value = value
    try {
      localStorage.setItem(STORAGE_KEY, value)
    } catch {
      /* 隐私模式等场景写入失败不阻塞 */
    }
  }

  return { semester, setSemester }
}
