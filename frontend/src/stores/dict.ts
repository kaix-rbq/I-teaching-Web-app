import { ref } from 'vue'
import { defineStore } from 'pinia'
import { fetchDictApi } from '@/api/dict'
import type { DepartmentOption, TeacherOption } from '@/types/dict'

/**
 * 字典 store：启动/进入需要筛选的页面时拉取一次并缓存。
 * departments / teachers 直接使用后端的数字 id，供筛选与表单绑定。
 */
export const useDictStore = defineStore('dict', () => {
  const semesters = ref<string[]>([])
  const departments = ref<DepartmentOption[]>([])
  const teachers = ref<TeacherOption[]>([])
  const loaded = ref(false)

  async function load(force = false): Promise<void> {
    if (loaded.value && !force) return
    const data = await fetchDictApi()
    semesters.value = data.semesters
    departments.value = data.departments
    teachers.value = data.teachers
    loaded.value = true
  }

  function teachersOfDepartment(departmentId?: number | ''): TeacherOption[] {
    if (!departmentId) return teachers.value
    return teachers.value.filter((teacher) => teacher.departmentId === departmentId)
  }

  function departmentName(id?: number | ''): string {
    if (!id) return ''
    return departments.value.find((item) => item.id === id)?.name ?? ''
  }

  return {
    semesters,
    departments,
    teachers,
    loaded,
    load,
    teachersOfDepartment,
    departmentName
  }
})
