export type CourseStatus = 'open' | 'draft' | 'closed'

export interface ClassInfo {
  id: string
  className: string
  schedule: string
  location: string
  studentCount: number
}

export interface Course {
  id: string
  code: string
  name: string
  credit: number
  hours: number
  semester: string
  department: string
  teacherId: string
  teacherName: string
  description: string
  objective?: string
  major?: string
  classes: ClassInfo[]
  studentCount: number
  resourceCount: number
  status: CourseStatus
}

export interface CourseQuery {
  semester?: string
  department?: string
  teacherId?: string
  status?: CourseStatus | ''
  keyword?: string
  page?: number
  pageSize?: number
}

export interface CourseFormModel {
  id?: string
  code: string
  name: string
  credit: number
  hours: number
  semester: string
  department: string
  teacherId: string
  description: string
  status: CourseStatus
}
