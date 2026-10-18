import { request } from './http'
import type { NotificationList } from '@/types/improvement'

/** 站内信分页（含未读数） */
export function fetchNotificationsApi(
  params: { page?: number; pageSize?: number } = {}
): Promise<NotificationList> {
  return request<NotificationList>({ url: '/notifications', method: 'get', params })
}

/** 单条已读（进入详情自动已读） */
export function markNotificationReadApi(id: number): Promise<Record<string, never>> {
  return request({ url: `/notifications/${id}/read`, method: 'put' })
}

/** 全部已读 */
export function markAllNotificationsReadApi(): Promise<Record<string, never>> {
  return request({ url: '/notifications/read-all', method: 'put' })
}
