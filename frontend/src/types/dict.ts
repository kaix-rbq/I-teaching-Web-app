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
