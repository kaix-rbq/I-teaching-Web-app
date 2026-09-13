export type SupervisionStatus = 'planned' | 'completed'

export interface SupervisionPlan {
  id: number
  courseId: number
  courseName: string
  teacherName: string
  supervisorName: string
  plannedDate: string
  status: SupervisionStatus
}

export interface DepartmentCoverage {
  department: string
  rate: number
}

export interface CoverageStat {
  totalCourses: number
  supervisedCourses: number
  rate: number
  byDepartment: DepartmentCoverage[]
}

export interface SupervisionQuery {
  status?: SupervisionStatus | ''
  /** 计划日期起（在 api 层映射为后端 dateFrom） */
  date?: string
  page?: number
  pageSize?: number
}

export interface DashboardStat {
  label: string
  value: number | string
  unit?: string
  icon?: string
  tone?: 'primary' | 'success' | 'warning' | 'info' | 'supervisor'
}

/**
 * 工作台视图模型。
 * 后端 GET /dashboard 按角色返回三种 DTO，由 api/dashboard.ts 收敛成同一形状，
 * 页面组件因此只需一套模板 + 角色分支。
 */
export interface DashboardData {
  stats: DashboardStat[]
  courses?: import('./course').Course[]
  plans?: SupervisionPlan[]
  coverage?: CoverageStat | null
  recentResources?: import('./resource').Resource[]
}
