export type CourseStatus = 'open' | 'draft' | 'closed'

export interface ClassInfo {
  id: number
  className: string
  schedule: string
  location: string
  studentCount: number
}

/**
 * 课程视图模型。
 * 列表接口（GET /courses）返回除 classes 外的全部字段，班级数量用 classCount；
 * 详情接口（GET /courses/:id）额外返回 classes 明细。
 */
export interface Course {
  id: number
  code: string
  name: string
  credit: number
  hours: number
  semester: string
  departmentId: number
  department: string
  teacherId: number
  teacherName: string
  description: string
  /** 培养目标 / 适用专业：Sprint 2 课程表扩展列，当前后端不返回，渲染时按空值降级 */
  objective?: string
  major?: string
  classes?: ClassInfo[]
  classCount: number
  studentCount: number
  resourceCount: number
  status: CourseStatus
}

export interface CourseQuery {
  semester?: string
  departmentId?: number
  teacherId?: number
  status?: CourseStatus | ''
  keyword?: string
  page?: number
  pageSize?: number
}

/** 与后端 CourseUpsertReq 逐字对齐的提交结构 */
export interface CourseUpsertPayload {
  code: string
  name: string
  departmentId: number
  teacherId: number
  semester: string
  credit: number
  hours: number
  description: string
  status: CourseStatus
}

/** 表单模型：下拉未选择时保持空串，提交前收敛为 CourseUpsertPayload */
export interface CourseFormModel {
  id?: number
  code: string
  name: string
  credit: number
  hours: number
  semester: string
  departmentId: number | ''
  teacherId: number | ''
  description: string
  status: CourseStatus
}
