import type { AxiosProgressEvent, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import type { ApiResponse } from '@/types/api'
import type { User } from '@/types/user'
import type { DashboardData, DashboardStat } from '@/types/supervision'
import { authRoutes, findUserById, TEACHERS } from './auth'
import { courseRoutes, courses, resources, scopedCourses } from './courses'
import { dictRoutes } from './dict'
import { getCoverage, plans, supervisionRoutes } from './supervision'
import { matchRoute, route, type MockRoute } from './router'

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function extractToken(config: InternalAxiosRequestConfig): string | null {
  const headers = config.headers as unknown as Record<string, unknown> | undefined
  const raw = (headers?.Authorization ?? headers?.authorization) as string | undefined
  if (!raw) return null
  return raw.replace(/^Bearer\s+/i, '')
}

function parseToken(token: string | null): User | null {
  if (!token) return null
  const match = /^mock-token-(.+)$/.exec(token)
  if (!match) return null
  return findUserById(match[1]) ?? null
}

async function simulateUpload(config: InternalAxiosRequestConfig): Promise<void> {
  if (!config.onUploadProgress) return
  const steps = [0.2, 0.55, 0.85, 1]
  for (const ratio of steps) {
    await delay(120)
    config.onUploadProgress({ loaded: ratio, total: 1, bytes: ratio, lengthComputable: true } as AxiosProgressEvent)
  }
}

function buildDashboard(user: User | null): DashboardData {
  if (!user) {
    return { stats: [] }
  }

  if (user.role === 'director') {
    const deptCourses = scopedCourses(user)
    const deptTeachers = TEACHERS.filter((teacher) => teacher.department === user.department)
    const classCount = deptCourses.reduce((sum, course) => sum + course.classes.length, 0)
    const resourceTotal = deptCourses.reduce((sum, course) => sum + course.resourceCount, 0)
    const stats: DashboardStat[] = [
      { label: '本室课程数', value: deptCourses.length, unit: '门', icon: 'course', tone: 'primary' },
      { label: '本室教师数', value: deptTeachers.length, unit: '人', icon: 'teacher', tone: 'info' },
      { label: '本学期开课班次', value: classCount, unit: '个', icon: 'class', tone: 'success' },
      { label: '课程资源总数', value: resourceTotal, unit: '份', icon: 'resource', tone: 'warning' }
    ]
    return { stats, courses: deptCourses.slice(0, 10) }
  }

  if (user.role === 'teacher') {
    const myCourses = scopedCourses(user)
    const classCount = myCourses.reduce((sum, course) => sum + course.classes.length, 0)
    const studentTotal = myCourses.reduce((sum, course) => sum + course.studentCount, 0)
    const resourceTotal = myCourses.reduce((sum, course) => sum + course.resourceCount, 0)
    const myCourseIds = new Set(myCourses.map((course) => course.id))
    const recentResources = resources
      .filter((item) => myCourseIds.has(item.courseId))
      .sort((a, b) => b.uploadedAt.localeCompare(a.uploadedAt))
      .slice(0, 5)
    const stats: DashboardStat[] = [
      { label: '我的课程数', value: myCourses.length, unit: '门', icon: 'course', tone: 'primary' },
      { label: '授课班级数', value: classCount, unit: '个', icon: 'class', tone: 'info' },
      { label: '学生总人次', value: studentTotal, unit: '人次', icon: 'student', tone: 'success' },
      { label: '资源总数', value: resourceTotal, unit: '份', icon: 'resource', tone: 'warning' }
    ]
    return { stats, courses: myCourses, recentResources }
  }

  const coverage = getCoverage()
  const completed = plans.filter((plan) => plan.status === 'completed').length
  const stats: DashboardStat[] = [
    { label: '全校课程数', value: courses.length, unit: '门', icon: 'course', tone: 'primary' },
    { label: '本学期听评课计划', value: plans.length, unit: '项', icon: 'plan', tone: 'info' },
    { label: '已完成听评课', value: completed, unit: '项', icon: 'done', tone: 'success' },
    { label: '督导覆盖率', value: Math.round(coverage.rate * 100), unit: '%', icon: 'coverage', tone: 'supervisor' }
  ]
  return { stats, plans: plans.slice(0, 8), coverage, departments: coverage.byDepartment }
}

const dashboardRoute: MockRoute = route('get', '/dashboard', (ctx): ApiResponse<unknown> => {
  if (!ctx.user) return { code: 401, message: '登录状态已失效', data: null }
  return { code: 0, message: 'ok', data: buildDashboard(ctx.user) }
})

const routes: MockRoute[] = [
  ...authRoutes,
  ...courseRoutes,
  ...supervisionRoutes,
  ...dictRoutes,
  dashboardRoute
]

export async function mockAdapter(config: InternalAxiosRequestConfig): Promise<AxiosResponse> {
  await delay(260)

  const method = (config.method ?? 'get').toLowerCase()
  let url = config.url ?? ''
  if (config.baseURL && url.startsWith(config.baseURL)) {
    url = url.slice(config.baseURL.length)
  }
  const [rawPath, queryString = ''] = url.split('?')
  const path = rawPath.startsWith('/') ? rawPath : `/${rawPath}`
  const query = Object.fromEntries(new URLSearchParams(queryString).entries())

  const token = extractToken(config)
  const user = parseToken(token)

  let body: Record<string, unknown> = {}
  let formData: FormData | null = null
  if (typeof FormData !== 'undefined' && config.data instanceof FormData) {
    formData = config.data
    await simulateUpload(config)
  } else if (typeof config.data === 'string') {
    try {
      body = JSON.parse(config.data) as Record<string, unknown>
    } catch {
      body = {}
    }
  } else if (config.data && typeof config.data === 'object') {
    body = config.data as Record<string, unknown>
  }

  const matched = matchRoute(routes, method, path)
  let payload: ApiResponse<unknown>
  if (!matched) {
    payload = { code: 404, message: `接口不存在：${method.toUpperCase()} ${path}`, data: null }
  } else {
    payload = await matched.route.handler({
      method,
      path,
      query,
      params: matched.params,
      body,
      formData,
      token,
      user
    })
  }

  return {
    data: payload,
    status: 200,
    statusText: 'OK',
    headers: {},
    config
  }
}
