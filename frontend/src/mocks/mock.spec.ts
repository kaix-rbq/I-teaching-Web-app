import { describe, expect, it } from 'vitest'
import type { InternalAxiosRequestConfig } from 'axios'
import { mockAdapter } from '@/mocks'
import type { ApiResponse, PageResult } from '@/types/api'
import type { Course } from '@/types/course'
import type { LoginResult } from '@/types/user'

interface CallOptions {
  data?: Record<string, unknown>
  token?: string
}

async function call<T>(
  method: string,
  url: string,
  options: CallOptions = {}
): Promise<ApiResponse<T>> {
  const headers: Record<string, string> = {}
  if (options.token) headers.Authorization = `Bearer ${options.token}`
  const config = {
    url,
    method,
    baseURL: '/api/v1',
    headers,
    data: options.data
  } as unknown as InternalAxiosRequestConfig

  const response = await mockAdapter(config)
  return response.data as ApiResponse<T>
}

async function loginAs(username: string): Promise<string> {
  const res = await call<LoginResult>('post', '/auth/login', {
    data: { username, password: '123456' }
  })
  expect(res.code).toBe(0)
  return res.data.token
}

describe('mock api', () => {
  it('rejects wrong credentials', async () => {
    const res = await call('post', '/auth/login', {
      data: { username: 'director', password: 'wrong' }
    })
    expect(res.code).toBe(1)
  })

  it('logs in three demo roles', async () => {
    for (const username of ['director', 'teacher', 'supervisor']) {
      const token = await loginAs(username)
      expect(token).toContain('mock-token-')
    }
  })

  it('returns the current user for a valid token', async () => {
    const token = await loginAs('teacher')
    const res = await call('get', '/auth/me', { token })
    expect(res.code).toBe(0)
    expect(res.data).toMatchObject({ role: 'teacher' })
  })

  it('scopes course list by role', async () => {
    const teacherToken = await loginAs('teacher')
    const teacherRes = await call<PageResult<Course>>('get', '/courses?page=1&pageSize=50', {
      token: teacherToken
    })
    expect(teacherRes.data.list.length).toBeGreaterThan(0)
    expect(teacherRes.data.list.every((course) => course.teacherId === 'u-teacher-1')).toBe(true)

    const supervisorToken = await loginAs('supervisor')
    const supervisorRes = await call<PageResult<Course>>('get', '/courses?page=1&pageSize=50', {
      token: supervisorToken
    })
    expect(supervisorRes.data.total).toBeGreaterThan(teacherRes.data.total)

    const directorToken = await loginAs('director')
    const directorRes = await call<PageResult<Course>>(
      'get',
      '/courses?page=1&pageSize=50&department=' + encodeURIComponent('软件工程教研室'),
      { token: directorToken }
    )
    expect(directorRes.data.list.every((course) => course.department === '软件工程教研室')).toBe(
      true
    )
  })

  it('forbids non-director from creating a course', async () => {
    const teacherToken = await loginAs('teacher')
    const res = await call('post', '/courses', {
      token: teacherToken,
      data: { code: 'SE9999', name: '测试课程' }
    })
    expect(res.code).toBe(403)
  })

  it('allows director to create and update a course', async () => {
    const directorToken = await loginAs('director')
    const created = await call<Course>('post', '/courses', {
      token: directorToken,
      data: {
        code: 'SE9999',
        name: '测试课程',
        credit: 2,
        hours: 32,
        semester: '2026-2027-1',
        department: '软件工程教研室',
        teacherId: 'u-teacher-1',
        description: '测试用',
        status: 'draft'
      }
    })
    expect(created.code).toBe(0)
    expect(created.data.teacherName).toBe('李文娟')

    const updated = await call<Course>('put', `/courses/${created.data.id}`, {
      token: directorToken,
      data: { code: 'SE9999', name: '测试课程（已改）', status: 'open' }
    })
    expect(updated.code).toBe(0)
    expect(updated.data.name).toBe('测试课程（已改）')
    expect(updated.data.status).toBe('open')
  })

  it('returns role-specific dashboard data', async () => {
    const teacherToken = await loginAs('teacher')
    const teacherRes = await call<{ stats: unknown[] }>('get', '/dashboard', {
      token: teacherToken
    })
    expect(teacherRes.code).toBe(0)
    expect(teacherRes.data.stats).toHaveLength(4)

    const supervisorToken = await loginAs('supervisor')
    const supervisorRes = await call<{ coverage: { rate: number } }>('get', '/dashboard', {
      token: supervisorToken
    })
    expect(supervisorRes.data.coverage.rate).toBeGreaterThan(0)
  })

  it('forbids non-supervisor from coverage', async () => {
    const directorToken = await loginAs('director')
    const res = await call('get', '/supervision/coverage', { token: directorToken })
    expect(res.code).toBe(403)
  })
})
