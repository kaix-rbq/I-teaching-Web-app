import { request } from './http'
import type { Course } from '@/types/course'
import type { Resource } from '@/types/resource'
import type { SupervisionPlan } from '@/types/supervision'
import type { DraftDTO } from '@/types/draft'
import type { PendingSessionItem } from '@/types/evaluation'
import type { DashboardData, DashboardStat } from '@/types/supervision'

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
  recentPlans: SupervisionPlan[]
  recentDrafts: DraftDTO[]
  pendingSessions: PendingSessionItem[]
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

  // 督导工作台聚焦「记录课程并评估」：不再返回统计卡与覆盖率。
  return {
    stats: [],
    plans: raw.recentPlans ?? [],
    recentDrafts: raw.recentDrafts ?? [],
    pendingSessions: raw.pendingSessions ?? []
  }
}
