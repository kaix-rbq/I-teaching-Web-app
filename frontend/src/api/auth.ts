import { request } from './http'
import type { LoginPayload, LoginResult, User } from '@/types/user'

export function loginApi(payload: LoginPayload): Promise<LoginResult> {
  return request<LoginResult>({
    url: '/auth/login',
    method: 'post',
    data: payload
  })
}

export function getMeApi(): Promise<User> {
  return request<User>({ url: '/auth/me', method: 'get' })
}

export function logoutApi(): Promise<null> {
  return request<null>({ url: '/auth/logout', method: 'post' })
}
