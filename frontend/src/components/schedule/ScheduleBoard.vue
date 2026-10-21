<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createScheduleEventApi,
  deleteScheduleEventApi,
  fetchScheduleEventsApi,
  updateScheduleEventRemarkApi
} from '@/api/schedule'
import { SCHEDULE_EVENT_TYPES } from '@/types/schedule'
import type { ScheduleEvent, ScheduleEventType } from '@/types/schedule'

/**
 * 主任「事务管理」周视图：纵轴 6:00-24:00（每小时一刻度），横轴周一至周日。
 * 事务块按起止时间绝对定位，类型分色；当天列加粗 + 背景高亮。
 */
const HOUR_HEIGHT = 44
const DAY_START = 6
const DAY_END = 24
const WEEK_LABELS = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

const todayKey = dayjs().format('YYYY-MM-DD')
const loading = ref(true)
const error = ref(false)
const events = ref<ScheduleEvent[]>([])

const weekDays = computed(() => {
  const monday = dayjs().startOf('day').subtract((dayjs().day() + 6) % 7, 'day')
  return Array.from({ length: 7 }, (_, index) => {
    const date = monday.add(index, 'day')
    return {
      key: date.format('YYYY-MM-DD'),
      label: WEEK_LABELS[index],
      day: date.format('D'),
      month: date.format('M月'),
      isToday: date.format('YYYY-MM-DD') === todayKey
    }
  })
})

const weekRange = computed(() => {
  const first = weekDays.value[0]
  const last = weekDays.value[6]
  return `${first.month}${first.day}日 - ${last.month}${last.day}日`
})

const hourTicks = computed(() =>
  Array.from({ length: DAY_END - DAY_START + 1 }, (_, index) => DAY_START + index)
)

function timeToMinutes(value: string): number {
  const [hour, minute] = value.split(':').map(Number)
  return hour * 60 + (minute || 0)
}

function minutesToTop(minutes: number): number {
  return ((minutes - DAY_START * 60) / 60) * HOUR_HEIGHT
}

function eventStyle(event: ScheduleEvent): Record<string, string> {
  const start = timeToMinutes(event.startTime)
  const end = timeToMinutes(event.endTime)
  const top = minutesToTop(start)
  const height = Math.max(HOUR_HEIGHT / 2, ((end - start) / 60) * HOUR_HEIGHT - 2)
  return { top: `${top}px`, height: `${height}px` }
}

function eventsOfDay(dateKey: string): ScheduleEvent[] {
  return events.value
    .filter((event) => event.eventDate === dateKey)
    .sort((left, right) => left.startTime.localeCompare(right.startTime))
}

function eventName(event: ScheduleEvent): string {
  if (event.typeText.trim()) return event.typeText.trim()
  return SCHEDULE_EVENT_TYPES.find((item) => item.value === event.eventType)?.label ?? '事务'
}

function eventTypeClass(type: ScheduleEventType): string {
  return `schedule-event--${type}`
}

const TYPE_COLORS: Record<ScheduleEventType, string> = {
  meeting: 'var(--color-primary)',
  open_class: 'var(--color-info)',
  lecture: 'var(--color-ai)',
  other: 'var(--color-warning)'
}

function typeColor(type: ScheduleEventType): string {
  return TYPE_COLORS[type]
}

async function load(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    events.value = (await fetchScheduleEventsApi({
      from: weekDays.value[0].key,
      to: weekDays.value[6].key
    })) ?? []
  } catch {
    error.value = true
    events.value = []
  } finally {
    loading.value = false
  }
}

/* ===== 新增事务 ===== */
const createVisible = ref(false)
const createSubmitting = ref(false)
const conflictMessage = ref('')
const createForm = reactive<{
  eventDate: string
  startTime: string
  endTime: string
  eventType: ScheduleEventType
  typeText: string
  remark: string
}>({
  eventDate: '',
  startTime: '09:00',
  endTime: '10:00',
  eventType: 'meeting',
  typeText: '',
  remark: ''
})

function buildTimeOptions(include24: boolean): string[] {
  const options: string[] = []
  for (let minutes = DAY_START * 60; minutes < DAY_END * 60; minutes += 30) {
    options.push(`${String(Math.floor(minutes / 60)).padStart(2, '0')}:${String(minutes % 60).padStart(2, '0')}`)
  }
  if (include24) options.push('24:00')
  return options
}

const startOptions = buildTimeOptions(false)
const endOptions = buildTimeOptions(true)

function openCreate(): void {
  createForm.eventDate = todayKey
  createForm.startTime = '09:00'
  createForm.endTime = '10:00'
  createForm.eventType = 'meeting'
  createForm.typeText = ''
  createForm.remark = ''
  conflictMessage.value = ''
  createVisible.value = true
}

function serverMessage(err: unknown): string {
  const candidate = err as { response?: { data?: { message?: string } } }
  return candidate.response?.data?.message ?? ''
}

async function submitCreate(): Promise<void> {
  conflictMessage.value = ''
  if (!createForm.eventDate) {
    ElMessage.error('请选择事务日期')
    return
  }
  if (timeToMinutes(createForm.endTime) <= timeToMinutes(createForm.startTime)) {
    ElMessage.error('结束时间必须晚于开始时间')
    return
  }
  if (createForm.eventType === 'other' && !createForm.typeText.trim()) {
    ElMessage.error('请填写「其他」事务名称')
    return
  }
  createSubmitting.value = true
  try {
    await createScheduleEventApi({
      eventDate: createForm.eventDate,
      startTime: createForm.startTime,
      endTime: createForm.endTime,
      eventType: createForm.eventType,
      typeText: createForm.eventType === 'other' ? createForm.typeText.trim() : '',
      remark: createForm.remark.trim()
    })
    ElMessage.success('事务已新增')
    createVisible.value = false
    await load()
  } catch (err) {
    const message = serverMessage(err)
    if (message.includes('冲突')) {
      conflictMessage.value = message
    } else {
      ElMessage.error(message || '新增事务失败')
    }
  } finally {
    createSubmitting.value = false
  }
}

/* ===== 查看 / 修改备注 / 删除 ===== */
const detailVisible = ref(false)
const detailSubmitting = ref(false)
const selected = ref<ScheduleEvent | null>(null)
const remarkDraft = ref('')

function openDetail(event: ScheduleEvent): void {
  selected.value = event
  remarkDraft.value = event.remark
  detailVisible.value = true
}

async function saveRemark(): Promise<void> {
  if (!selected.value) return
  detailSubmitting.value = true
  try {
    const updated = await updateScheduleEventRemarkApi(selected.value.id, remarkDraft.value.trim())
    ElMessage.success('备注已更新')
    selected.value = updated
    await load()
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    detailSubmitting.value = false
  }
}

async function removeEvent(): Promise<void> {
  if (!selected.value) return
  try {
    await ElMessageBox.confirm(
      `确认删除「${eventName(selected.value)}」？删除后不可恢复。`,
      '删除事务',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  detailSubmitting.value = true
  try {
    await deleteScheduleEventApi(selected.value.id)
    ElMessage.success('事务已删除')
    detailVisible.value = false
    await load()
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    detailSubmitting.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <section class="schedule-board" aria-label="事务管理">
    <header class="schedule-board__head">
      <div>
        <h2 class="schedule-board__title">事务管理</h2>
        <p class="schedule-board__sub">本周 {{ weekRange }}，6:00-24:00</p>
      </div>
      <el-button type="primary" @click="openCreate">新增事务</el-button>
    </header>

    <el-skeleton v-if="loading" :rows="8" animated />

    <EmptyState v-else-if="error" description="事务加载失败">
      <template #action>
        <el-button @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <div v-else class="schedule-board__scroll">
      <div class="schedule-grid" :style="{ '--hour-height': `${HOUR_HEIGHT}px`, '--half-hour-height': `${HOUR_HEIGHT / 2}px`, '--grid-height': `${18 * HOUR_HEIGHT}px` }">
        <div class="schedule-grid__corner" aria-hidden="true" />
        <div
          v-for="day in weekDays"
          :key="`head-${day.key}`"
          class="schedule-grid__day-head"
          :class="{ 'schedule-grid__day-head--today': day.isToday }"
        >
          <span class="schedule-grid__day-label">{{ day.label }}</span>
          <span class="schedule-grid__day-date tabular-nums">{{ day.day }}</span>
        </div>

        <div class="schedule-grid__time-col" aria-hidden="true">
          <span
            v-for="hour in hourTicks"
            :key="hour"
            class="schedule-grid__time"
            :style="{ top: `${minutesToTop(hour * 60)}px` }"
          >
            {{ String(hour).padStart(2, '0') }}:00
          </span>
        </div>

        <div
          v-for="day in weekDays"
          :key="`col-${day.key}`"
          class="schedule-grid__day"
          :class="{ 'schedule-grid__day--today': day.isToday }"
        >
          <button
            v-for="event in eventsOfDay(day.key)"
            :key="event.id"
            type="button"
            class="schedule-event"
            :class="eventTypeClass(event.eventType)"
            :style="eventStyle(event)"
            :aria-label="`查看事务 ${eventName(event)} ${event.startTime} 至 ${event.endTime}`"
            @click="openDetail(event)"
          >
            <span class="schedule-event__time tabular-nums">{{ event.startTime }}-{{ event.endTime }}</span>
            <span class="schedule-event__name">{{ eventName(event) }}</span>
          </button>
        </div>
      </div>
    </div>

    <p class="schedule-board__legend" aria-hidden="true">
      <span v-for="type in SCHEDULE_EVENT_TYPES" :key="type.value" class="schedule-board__legend-item">
        <i class="schedule-board__swatch" :style="{ backgroundColor: typeColor(type.value) }" />
        {{ type.label }}
      </span>
    </p>

    <!-- 新增事务 -->
    <el-dialog v-model="createVisible" title="新增事务" width="500px" append-to-body>
      <el-form label-width="80px">
        <el-form-item label="日期" required>
          <el-date-picker
            v-model="createForm.eventDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="选择日期"
            class="schedule-board__field"
          />
        </el-form-item>
        <el-form-item label="时间" required>
          <div class="schedule-board__time-range">
            <el-select v-model="createForm.startTime" class="schedule-board__time-select" aria-label="开始时间">
              <el-option v-for="time in startOptions" :key="`start-${time}`" :label="time" :value="time" />
            </el-select>
            <span>至</span>
            <el-select v-model="createForm.endTime" class="schedule-board__time-select" aria-label="结束时间">
              <el-option v-for="time in endOptions" :key="`end-${time}`" :label="time" :value="time" />
            </el-select>
          </div>
        </el-form-item>
        <el-form-item label="类型" required>
          <el-radio-group v-model="createForm.eventType">
            <el-radio v-for="type in SCHEDULE_EVENT_TYPES" :key="type.value" :value="type.value">
              {{ type.label }}
            </el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="createForm.eventType === 'other'" label="名称" required>
          <el-input v-model="createForm.typeText" maxlength="32" placeholder="填写事务名称" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input
            v-model="createForm.remark"
            type="textarea"
            :rows="2"
            maxlength="255"
            show-word-limit
            placeholder="选填"
          />
        </el-form-item>
        <p v-if="conflictMessage" class="schedule-board__conflict" role="alert">
          {{ conflictMessage }}
        </p>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createSubmitting" @click="submitCreate">保存</el-button>
      </template>
    </el-dialog>

    <!-- 查看 / 修改备注 / 删除 -->
    <el-dialog v-model="detailVisible" title="事务详情" width="480px" append-to-body>
      <template v-if="selected">
        <dl class="schedule-board__detail">
          <div>
            <dt>时间</dt>
            <dd class="tabular-nums">{{ selected.eventDate }} {{ selected.startTime }}-{{ selected.endTime }}</dd>
          </div>
          <div>
            <dt>类型</dt>
            <dd>{{ eventName(selected) }}</dd>
          </div>
        </dl>
        <el-form label-width="56px">
          <el-form-item label="备注">
            <el-input
              v-model="remarkDraft"
              type="textarea"
              :rows="3"
              maxlength="255"
              show-word-limit
              placeholder="补充事务说明"
            />
          </el-form-item>
        </el-form>
      </template>
      <template #footer>
        <el-button class="schedule-board__delete" text type="danger" :disabled="detailSubmitting" @click="removeEvent">
          删除
        </el-button>
        <el-button @click="detailVisible = false">取消</el-button>
        <el-button type="primary" :loading="detailSubmitting" @click="saveRemark">保存备注</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped lang="scss">
.schedule-board {
  padding: var(--spacing-5) var(--spacing-6);
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);

  &__head {
    display: flex;
    gap: var(--spacing-4);
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__title {
    font-size: var(--font-size-lg);
  }

  &__sub {
    margin-top: 4px;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__scroll {
    overflow-x: auto;
  }

  &__legend {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-4);
    padding-top: var(--spacing-3);
    margin-top: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    border-top: 1px dashed var(--color-divider);
  }

  &__legend-item {
    display: inline-flex;
    gap: 6px;
    align-items: center;
  }

  &__swatch {
    width: 10px;
    height: 10px;
    border-radius: 3px;
    background-color: var(--event-color);
  }

  &__field {
    width: 100%;
  }

  &__time-range {
    display: flex;
    gap: var(--spacing-2);
    align-items: center;
    color: var(--color-text-tertiary);
  }

  &__time-select {
    width: 110px;
  }

  &__conflict {
    padding: var(--spacing-2) var(--spacing-3);
    margin-left: 80px;
    font-size: var(--font-size-sm);
    color: var(--color-error);
    background-color: color-mix(in srgb, var(--color-error) 8%, #ffffff);
    border-radius: var(--radius-sm);
  }

  &__detail {
    display: flex;
    gap: var(--spacing-8);
    margin: 0 0 var(--spacing-4);

    dt {
      font-size: var(--font-size-xs);
      color: var(--color-text-tertiary);
    }

    dd {
      margin: 2px 0 0;
      font-size: var(--font-size-sm);
      color: var(--color-text-primary);
    }
  }

  &__delete {
    margin-right: auto;
  }
}

.schedule-grid {
  display: grid;
  grid-template-columns: 58px repeat(7, minmax(112px, 1fr));
  min-width: 860px;
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-md);

  &__corner {
    grid-column: 1;
    grid-row: 1;
    background-color: var(--color-bg-page);
    border-right: 1px solid var(--color-divider);
    border-bottom: 1px solid var(--color-divider);
    border-radius: var(--radius-md) 0 0;
  }

  &__day-head {
    display: flex;
    grid-row: 1;
    gap: var(--spacing-2);
    align-items: baseline;
    justify-content: center;
    padding: var(--spacing-2) var(--spacing-1);
    background-color: var(--color-bg-page);
    border-bottom: 1px solid var(--color-divider);
    border-left: 1px solid var(--color-divider);

    &--today {
      background-color: var(--color-primary-bg);

      .schedule-grid__day-label,
      .schedule-grid__day-date {
        font-weight: 700;
        color: var(--color-primary);
      }
    }
  }

  &__day-label {
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__day-date {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__time-col {
    position: relative;
    grid-column: 1;
    grid-row: 2;
    height: var(--grid-height);
    background-color: var(--color-bg-page);
    border-right: 1px solid var(--color-divider);
    border-radius: 0 0 0 var(--radius-md);
  }

  &__time {
    position: absolute;
    right: var(--spacing-2);
    font-size: 11px;
    color: var(--color-text-tertiary);
    transform: translateY(-50%);
  }

  &__day {
    position: relative;
    grid-row: 2;
    height: var(--grid-height);
    border-left: 1px solid var(--color-divider);
    background-image:
      repeating-linear-gradient(
        to bottom,
        transparent 0,
        transparent calc(var(--half-hour-height) - 1px),
        rgba(31, 35, 41, 0.035) calc(var(--half-hour-height) - 1px),
        rgba(31, 35, 41, 0.035) var(--half-hour-height),
        transparent var(--half-hour-height),
        transparent calc(var(--hour-height) - 1px),
        var(--color-divider) calc(var(--hour-height) - 1px),
        var(--color-divider) var(--hour-height)
      );

    &--today {
      background-color: color-mix(in srgb, var(--color-primary) 4%, transparent);
    }
  }
}

.schedule-event {
  --event-color: var(--color-primary);
  --event-bg: var(--color-primary-bg);

  position: absolute;
  right: 4px;
  left: 4px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 4px 6px;
  overflow: hidden;
  font-size: var(--font-size-xs);
  line-height: 1.3;
  text-align: left;
  color: var(--color-text-primary);
  cursor: pointer;
  background-color: var(--event-bg);
  border: none;
  border-left: 3px solid var(--event-color);
  border-radius: var(--radius-sm);
  transition: box-shadow var(--duration-fast) var(--ease-out-soft);

  &:hover,
  &:focus-visible {
    box-shadow: 0 2px 8px color-mix(in srgb, var(--event-color) 24%, transparent);
    outline: 2px solid var(--event-color);
    outline-offset: 1px;
  }

  &--meeting {
    --event-color: var(--color-primary);
    --event-bg: color-mix(in srgb, var(--color-primary) 10%, #ffffff);
  }

  &--open_class {
    --event-color: var(--color-info);
    --event-bg: color-mix(in srgb, var(--color-info) 12%, #ffffff);
  }

  &--lecture {
    --event-color: var(--color-ai);
    --event-bg: var(--color-ai-bg);
  }

  &--other {
    --event-color: var(--color-warning);
    --event-bg: color-mix(in srgb, var(--color-warning) 14%, #ffffff);
  }

  &__time {
    font-size: 10px;
    color: var(--color-text-tertiary);
  }

  &__name {
    font-weight: 600;
    color: var(--color-text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

@media (max-width: 960px) {
  .schedule-board__head {
    flex-direction: column;
  }
}
</style>
