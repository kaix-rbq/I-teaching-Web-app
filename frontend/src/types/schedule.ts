export type ScheduleEventType = 'meeting' | 'open_class' | 'lecture' | 'other'

/** 主任事务（GET/POST /schedule-events）。 */
export interface ScheduleEvent {
  id: number
  ownerUserId: number
  /** YYYY-MM-DD */
  eventDate: string
  /** HH:MM */
  startTime: string
  /** HH:MM（24:00 合法） */
  endTime: string
  eventType: ScheduleEventType
  typeText: string
  remark: string
  createdAt: string
  updatedAt: string
}

export interface ScheduleEventCreatePayload {
  eventDate: string
  startTime: string
  endTime: string
  eventType: ScheduleEventType
  typeText?: string
  remark?: string
}

export const SCHEDULE_EVENT_TYPES: Array<{ value: ScheduleEventType; label: string }> = [
  { value: 'meeting', label: '会议' },
  { value: 'open_class', label: '公开课' },
  { value: 'lecture', label: '讲座' },
  { value: 'other', label: '其他' }
]
