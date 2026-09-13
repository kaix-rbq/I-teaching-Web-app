import type { User } from '@/types/user'
import type { ApiResponse } from '@/types/api'
import { route, type MockRoute } from './router'

export const DEMO_PASSWORD = '123456'

interface Account {
  username: string
  password: string
  user: User
}

const DIRECTOR: User = {
  id: 'u-director',
  name: '王建国',
  role: 'director',
  department: '软件工程教研室',
  jobNo: 'D001'
}

const SUPERVISOR: User = {
  id: 'u-supervisor',
  name: '张华',
  role: 'supervisor',
  department: '教务处',
  jobNo: 'S001'
}

export const TEACHERS: User[] = [
  {
    id: 'u-teacher-1',
    name: '李文娟',
    role: 'teacher',
    department: '软件工程教研室',
    jobNo: 'T101'
  },
  {
    id: 'u-teacher-2',
    name: '赵鹏',
    role: 'teacher',
    department: '软件工程教研室',
    jobNo: 'T102'
  },
  {
    id: 'u-teacher-3',
    name: '孙晓雨',
    role: 'teacher',
    department: '软件工程教研室',
    jobNo: 'T103'
  },
  {
    id: 'u-teacher-4',
    name: '陈立',
    role: 'teacher',
    department: '计算机科学与技术教研室',
    jobNo: 'T201'
  },
  {
    id: 'u-teacher-5',
    name: '周敏',
    role: 'teacher',
    department: '网络工程教研室',
    jobNo: 'T301'
  },
  {
    id: 'u-teacher-6',
    name: '吴桐',
    role: 'teacher',
    department: '人工智能教研室',
    jobNo: 'T401'
  }
]

const ACCOUNTS: Account[] = [
  { username: 'director', password: DEMO_PASSWORD, user: DIRECTOR },
  { username: 'teacher', password: DEMO_PASSWORD, user: TEACHERS[0] },
  { username: 'supervisor', password: DEMO_PASSWORD, user: SUPERVISOR }
]

export function findUserById(id: string): User | undefined {
  if (DIRECTOR.id === id) return DIRECTOR
  if (SUPERVISOR.id === id) return SUPERVISOR
  return TEACHERS.find((teacher) => teacher.id === id)
}

export function listTeachersByDepartment(department: string): User[] {
  return TEACHERS.filter((teacher) => teacher.department === department)
}

export const authRoutes: MockRoute[] = [
  route('post', '/auth/login', (ctx): ApiResponse<unknown> => {
    const username = String(ctx.body.username ?? '')
    const password = String(ctx.body.password ?? '')
    const account = ACCOUNTS.find((item) => item.username === username)
    if (!account || account.password !== password) {
      return { code: 1, message: '账号或密码错误', data: null }
    }
    return {
      code: 0,
      message: 'ok',
      data: { token: `mock-token-${account.user.id}`, user: account.user }
    }
  }),
  route('get', '/auth/me', (ctx): ApiResponse<unknown> => {
    if (!ctx.user) return { code: 401, message: '登录状态已失效', data: null }
    return { code: 0, message: 'ok', data: ctx.user }
  }),
  route('post', '/auth/logout', (): ApiResponse<unknown> => {
    return { code: 0, message: 'ok', data: null }
  })
]
