export type SupervisionStatus = 'planned' | 'completed'

export interface SupervisionPlan {
  id: number
  courseId: number
  courseName: string
  teacherName: string
  supervisorName: string
  plannedDate: string
  status: SupervisionStatus
  /** 关联授课记录（null = 尚未创建，需先创建再评估） */
  sessionId: number | null
  /** 关联授课记录是否已完成督导评价（决定是否暴露评估入口） */
  evaluated: boolean
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
  /** 督导工作台「草稿箱」区块：最近创建的草稿 */
  recentDrafts?: import('./draft').DraftDTO[]
  /** 督导工作台「待评估授课记录」：手动新增或尚未评价的授课记录 */
  pendingSessions?: import('./evaluation').PendingSessionItem[]
}
