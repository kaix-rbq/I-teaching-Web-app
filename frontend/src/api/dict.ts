import { request } from './http'
import { SEMESTERS } from '@/constants'
import type { DepartmentOption, DictData, TeacherOption, TimetableItem } from '@/types/dict'

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

/** 课表（B11 建课下拉 / V13 周课表）。teacher 查本人，supervisor 可带 teacherId。 */
export function fetchMyTimetableApi(params: { teacherId?: number; semester?: string } = {}): Promise<TimetableItem[]> {
  return request<TimetableItem[]>({ url: '/timetable/mine', method: 'get', params })
}

/** 仅本人教师可改课表备注（PATCH /timetable/:id/remark）。 */
export function updateTimetableRemarkApi(id: number, remark: string): Promise<TimetableItem> {
  return request<TimetableItem>({ url: `/timetable/${id}/remark`, method: 'patch', data: { remark } })
}
