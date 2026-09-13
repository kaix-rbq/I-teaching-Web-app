import type { ApiResponse, PageResult } from '@/types/api'
import type { CoverageStat, SupervisionPlan, SupervisionStatus } from '@/types/supervision'
import { DEPARTMENTS, courses } from './courses'
import { route, type MockRoute } from './router'

interface PlanSeed {
  id: string
  courseId: string
  supervisorName: string
  plannedDate: string
  status: SupervisionStatus
}

const PLAN_SEEDS: PlanSeed[] = [
  { id: 'p-001', courseId: 'c-001', supervisorName: '张华', plannedDate: '2026-09-15', status: 'completed' },
  { id: 'p-002', courseId: 'c-002', supervisorName: '张华', plannedDate: '2026-09-18', status: 'completed' },
  { id: 'p-003', courseId: 'c-006', supervisorName: '刘颖', plannedDate: '2026-09-22', status: 'planned' },
  { id: 'p-004', courseId: 'c-009', supervisorName: '张华', plannedDate: '2026-09-25', status: 'planned' },
  { id: 'p-005', courseId: 'c-012', supervisorName: '刘颖', plannedDate: '2026-09-29', status: 'completed' },
  { id: 'p-006', courseId: 'c-003', supervisorName: '张华', plannedDate: '2026-10-09', status: 'planned' },
  { id: 'p-007', courseId: 'c-007', supervisorName: '刘颖', plannedDate: '2026-10-13', status: 'planned' },
  { id: 'p-008', courseId: 'c-010', supervisorName: '张华', plannedDate: '2026-10-16', status: 'completed' },
  { id: 'p-009', courseId: 'c-013', supervisorName: '刘颖', plannedDate: '2026-10-20', status: 'planned' },
  { id: 'p-010', courseId: 'c-001', supervisorName: '刘颖', plannedDate: '2026-10-23', status: 'planned' },
  { id: 'p-011', courseId: 'c-006', supervisorName: '张华', plannedDate: '2026-10-27', status: 'completed' },
  { id: 'p-012', courseId: 'c-009', supervisorName: '刘颖', plannedDate: '2026-11-03', status: 'planned' }
]

export const plans: SupervisionPlan[] = PLAN_SEEDS.map((seed) => {
  const course = courses.find((item) => item.id === seed.courseId)
  return {
    id: seed.id,
    courseId: seed.courseId,
    courseName: course?.name ?? '未知课程',
    teacherName: course?.teacherName ?? '未知教师',
    supervisorName: seed.supervisorName,
    plannedDate: seed.plannedDate,
    status: seed.status
  }
})

function computeCoverage(): CoverageStat {
  const supervisedCourseIds = new Set(
    plans.filter((plan) => plan.status === 'completed').map((plan) => plan.courseId)
  )

  const byDepartment = DEPARTMENTS.map((department) => {
    const deptCourses = courses.filter((course) => course.department === department)
    if (deptCourses.length === 0) return { department, rate: 0 }
    const done = deptCourses.filter((course) => supervisedCourseIds.has(course.id)).length
    return { department, rate: done / deptCourses.length }
  })

  return {
    totalCourses: courses.length,
    supervisedCourses: supervisedCourseIds.size,
    rate: courses.length ? supervisedCourseIds.size / courses.length : 0,
    byDepartment
  }
}

export const getCoverage = computeCoverage

function paginate<T>(list: T[], page: number, pageSize: number): PageResult<T> {
  const start = (page - 1) * pageSize
  return { list: list.slice(start, start + pageSize), total: list.length, page, pageSize }
}

export const supervisionRoutes: MockRoute[] = [
  route('get', '/supervision/coverage', (ctx): ApiResponse<unknown> => {
    if (!ctx.user || ctx.user.role !== 'supervisor') {
      return { code: 403, message: '无权访问督导数据', data: null }
    }
    return { code: 0, message: 'ok', data: computeCoverage() }
  }),

  route('get', '/supervision/plans', (ctx): ApiResponse<unknown> => {
    if (!ctx.user || ctx.user.role !== 'supervisor') {
      return { code: 403, message: '无权访问督导数据', data: null }
    }
    let list = [...plans]
    if (ctx.query.status) list = list.filter((plan) => plan.status === ctx.query.status)
    if (ctx.query.date) list = list.filter((plan) => plan.plannedDate >= ctx.query.date)
    const page = Number(ctx.query.page ?? 1) || 1
    const pageSize = Number(ctx.query.pageSize ?? 10) || 10
    return { code: 0, message: 'ok', data: paginate(list, page, pageSize) }
  })
]
