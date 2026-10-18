import { request } from './http'
import type { Reflection, ReflectionPayload } from '@/types/improvement'

/** 教学反思列表（teacher 本人；supervisor 需带 sessionId 读关联反思） */
export function fetchReflectionsApi(
  params: { sessionId?: number } = {}
): Promise<Reflection[]> {
  return request<Reflection[]>({ url: '/teaching-reflections', method: 'get', params })
}

export function createReflectionApi(payload: ReflectionPayload): Promise<Reflection> {
  return request<Reflection>({ url: '/teaching-reflections', method: 'post', data: payload })
}

export function updateReflectionApi(id: number, payload: ReflectionPayload): Promise<Reflection> {
  return request<Reflection>({ url: `/teaching-reflections/${id}`, method: 'put', data: payload })
}
