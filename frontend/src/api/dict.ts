import { request } from './http'
import { SEMESTERS } from '@/constants'
import type { DepartmentOption, DictData, TeacherOption } from '@/types/dict'

/**
 * 字典数据源：后端提供 /departments 与 /teachers 两个接口，
 * 学期由前端常量维护（见 constants/index.ts），此处合并为统一的 DictData。
 */
export async function fetchDictApi(): Promise<DictData> {
  const [departments, teachers] = await Promise.all([
    request<DepartmentOption[]>({ url: '/departments', method: 'get' }),
    request<TeacherOption[]>({ url: '/teachers', method: 'get' })
  ])
  return {
    semesters: [...SEMESTERS],
    departments: departments ?? [],
    teachers: teachers ?? []
  }
}
