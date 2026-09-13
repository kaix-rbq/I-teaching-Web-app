import type { ApiResponse } from '@/types/api'
import { TEACHERS } from './auth'
import { DEPARTMENTS, SEMESTERS } from './courses'
import { route, type MockRoute } from './router'

export const dictRoutes: MockRoute[] = [
  route('get', '/dict', (): ApiResponse<unknown> => {
    return {
      code: 0,
      message: 'ok',
      data: {
        semesters: SEMESTERS,
        departments: DEPARTMENTS,
        teachers: TEACHERS
      }
    }
  })
]
