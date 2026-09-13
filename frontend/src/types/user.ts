export type Role = 'director' | 'teacher' | 'supervisor'

export interface User {
  id: number
  name: string
  role: Role
  departmentId?: number
  department?: string
  jobNo?: string
  avatar?: string
}

export interface LoginPayload {
  username: string
  password: string
}

export interface LoginResult {
  token: string
  user: User
}
