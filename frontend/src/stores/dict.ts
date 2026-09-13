import { ref } from 'vue'
import { defineStore } from 'pinia'
import { fetchDictApi } from '@/api/dict'
import type { DictData } from '@/types/dict'
import type { User } from '@/types/user'

export const useDictStore = defineStore('dict', () => {
  const semesters = ref<string[]>([])
  const departments = ref<string[]>([])
  const teachers = ref<User[]>([])
  const loaded = ref(false)

  async function load(force = false): Promise<void> {
    if (loaded.value && !force) return
    const data: DictData = await fetchDictApi()
    semesters.value = data.semesters
    departments.value = data.departments
    teachers.value = data.teachers
    loaded.value = true
  }

  function teachersOfDepartment(department?: string): User[] {
    if (!department) return teachers.value
    return teachers.value.filter((teacher) => teacher.department === department)
  }

  return {
    semesters,
    departments,
    teachers,
    loaded,
    load,
    teachersOfDepartment
  }
})
