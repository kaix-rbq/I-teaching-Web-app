import { request } from './http'
import type { ScheduleEvent, ScheduleEventCreatePayload } from '@/types/schedule'

/** 本人事务（仅主任）：from/to 为 YYYY-MM-DD。 */
export function fetchScheduleEventsApi(params: { from: string; to: string }): Promise<ScheduleEvent[]> {
  return request<ScheduleEvent[]>({ url: '/schedule-events', method: 'get', params })
}

export function createScheduleEventApi(payload: ScheduleEventCreatePayload): Promise<ScheduleEvent> {
  return request<ScheduleEvent>({ url: '/schedule-events', method: 'post', data: payload })
}

export function updateScheduleEventRemarkApi(id: number, remark: string): Promise<ScheduleEvent> {
  return request<ScheduleEvent>({ url: `/schedule-events/${id}`, method: 'patch', data: { remark } })
}

export function deleteScheduleEventApi(id: number): Promise<{ id: number }> {
  return request<{ id: number }>({ url: `/schedule-events/${id}`, method: 'delete' })
}
