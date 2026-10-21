export interface DepartmentOption {
  id: number
  name: string
}

export interface TeacherOption {
  id: number
  name: string
  departmentId?: number
}

export interface DictData {
  semesters: string[]
  departments: DepartmentOption[]
  teachers: TeacherOption[]
}

/** 课表行（GET /timetable/mine）：B11 建课下拉 + 教师工作台周课表。 */
export interface TimetableItem {
  id: number
  courseId: number
  courseCode: string
  courseName: string
  classId: number
  className: string
  teacherId: number
  teacherName: string
  semester: string
  /** 1=周一 … 7=周日；0=历史数据未排课 */
  weekday: number
  /** 1-11 节；0=未指定 */
  periodStart: number
  periodEnd: number
  classroom: string
  remark: string
}
