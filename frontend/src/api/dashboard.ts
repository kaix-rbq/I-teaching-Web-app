import { request } from './http'
import type { Course } from '@/types/course'
import type { Resource } from '@/types/resource'
import type { SupervisionPlan } from '@/types/supervision'
import type { CoverageStat, DashboardData, DashboardStat } from '@/types/supervision'

/** 后端 GET /dashboard 按角色返回的三种 DTO（见后端 AGENTS.md §8.2） */
interface DirectorDashboard {
  courseCount: number
  teacherCount: number
  classCount: number
  resourceCount: number
  recentCourses: Course[]
}

interface TeacherDashboard {
  courseCount: number
  classCount: number
  studentCount: number
  resourceCount: number
  myCourses: Course[]
  recentResources: Resource[]
}

interface SupervisorDashboard {
  courseCount: number
  planCount: number
  completedCount: number
  coverageRate: number
  recentPlans: SupervisionPlan[]
  byDepartment: { department: string; rate: number }[]
}

function stat(
  label: string,
  value: number,
  unit: string,
  icon: string,
  tone: DashboardStat['tone']
): DashboardStat {
  return { label, value, unit, icon, tone }
}

/**
 * 把后端三种角色 DTO 收敛为前端统一的 DashboardData，
 * 使 DashboardView 只需一套模板 + 角色分支。
 */
export async function fetchDashboardApi(): Promise<DashboardData> {
  const raw = await request<DirectorDashboard | TeacherDashboard | SupervisorDashboard>({
    url: '/dashboard',
    method: 'get'
  })

  if ('recentCourses' in raw) {
    return {
      stats: [
        stat('本室课程数', raw.courseCount, '门', 'course', 'primary'),
        stat('本室教师数', raw.teacherCount, '人', 'teacher', 'info'),
        stat('本学期开课班次', raw.classCount, '个', 'class', 'success'),
        stat('课程资源总数', raw.resourceCount, '份', 'resource', 'warning')
      ],
      courses: raw.recentCourses ?? []
    }
  }

  if ('myCourses' in raw) {
    return {
      stats: [
        stat('我的课程数', raw.courseCount, '门', 'course', 'primary'),
        stat('授课班级数', raw.classCount, '个', 'class', 'info'),
        stat('学生总人次', raw.studentCount, '人次', 'student', 'success'),
        stat('资源总数', raw.resourceCount, '份', 'resource', 'warning')
      ],
      courses: raw.myCourses ?? [],
      recentResources: raw.recentResources ?? []
    }
  }

  const coverage: CoverageStat = {
    totalCourses: raw.courseCount,
    supervisedCourses: raw.completedCount,
    rate: raw.coverageRate,
    byDepartment: raw.byDepartment ?? []
  }

  return {
    stats: [
      stat('全校课程数', raw.courseCount, '门', 'course', 'primary'),
      stat('本学期听评课计划', raw.planCount, '项', 'plan', 'info'),
      stat('已完成听评课', raw.completedCount, '项', 'done', 'success'),
      stat('督导覆盖率', Math.round(raw.coverageRate * 100), '%', 'coverage', 'supervisor')
    ],
    plans: raw.recentPlans ?? [],
    coverage
  }
}
