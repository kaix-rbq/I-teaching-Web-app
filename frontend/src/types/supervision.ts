export type SupervisionStatus = 'planned' | 'completed'

export interface SupervisionPlan {
  id: string
  courseId: string
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

export interface DashboardData {
  stats: DashboardStat[]
  courses?: import('./course').Course[]
  plans?: SupervisionPlan[]
  coverage?: CoverageStat
  recentResources?: import('./resource').Resource[]
  departments?: DepartmentCoverage[]
}
