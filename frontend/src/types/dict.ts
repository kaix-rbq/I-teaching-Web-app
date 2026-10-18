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

/** 最小课表下拉行（B11）：GET /timetable/mine，建课时课程/班级下拉数据源。 */
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
}
