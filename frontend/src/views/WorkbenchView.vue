<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ClassroomIssueBoard from '@/components/classroom/ClassroomIssueBoard.vue'
import { fetchMyTimetableApi, updateTimetableRemarkApi } from '@/api/dict'
import { createSessionApi } from '@/api/session'
import { useSemester } from '@/composables/useSemester'
import type { TimetableItem } from '@/types/dict'

/**
 * 教师工作台：周课表 + 今日课程 + 课堂问题登记。
 * 分数与档案相关能力留在「我的质量档案」，本页只回答「今天上什么、怎么记录」。
 */
const { semester } = useSemester()

const WEEK_LABELS = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
const PERIODS = Array.from({ length: 11 }, (_, index) => index + 1)
const SECTIONS = [
  { label: '上午', start: 1, end: 4 },
  { label: '下午', start: 5, end: 8 },
  { label: '晚上', start: 9, end: 11 }
]
const BLOCK_TONES = ['primary', 'info', 'ai', 'warning']

const todayKey = dayjs().format('YYYY-MM-DD')
const todayWeekday = dayjs().day() === 0 ? 7 : dayjs().day()
const todayLabel = `${dayjs().format('M月D日')} ${WEEK_LABELS[todayWeekday - 1]}`

const loading = ref(true)
const error = ref(false)
const timetable = ref<TimetableItem[]>([])

const weekDays = computed(() => {
  const monday = dayjs().startOf('day').subtract((dayjs().day() + 6) % 7, 'day')
  return Array.from({ length: 7 }, (_, index) => {
    const date = monday.add(index, 'day')
    return {
      key: date.format('YYYY-MM-DD'),
      weekday: index + 1,
      label: WEEK_LABELS[index],
      day: date.format('D'),
      isToday: index + 1 === todayWeekday
    }
  })
})

const scheduledRows = computed(() =>
  timetable.value.filter((row) => row.weekday >= 1 && row.weekday <= 7 && row.periodStart >= 1)
)

const todayRows = computed(() =>
  scheduledRows.value
    .filter((row) => row.weekday === todayWeekday)
    .slice()
    .sort((left, right) => left.periodStart - right.periodStart)
)

const weekCourseCount = computed(() => scheduledRows.value.length)

function toneOf(row: TimetableItem): string {
  return BLOCK_TONES[row.courseId % BLOCK_TONES.length]
}

function blockStyle(row: TimetableItem): Record<string, string> {
  const end = Math.max(row.periodStart, row.periodEnd || row.periodStart)
  return {
    gridColumn: String(row.weekday + 2),
    gridRow: `${row.periodStart + 1} / ${end + 2}`
  }
}

function periodLabel(row: TimetableItem): string {
  const end = row.periodEnd || row.periodStart
  return end > row.periodStart ? `${row.periodStart}-${end} 节` : `第 ${row.periodStart} 节`
}

async function load(): Promise<void> {
  loading.value = true
  error.value = false
  try {
    timetable.value = (await fetchMyTimetableApi({ semester: semester.value || undefined })) ?? []
  } catch {
    error.value = true
    timetable.value = []
  } finally {
    loading.value = false
  }
}

/* ===== 课程块：查看教室 / 班级 + 编辑备注 ===== */
const detailVisible = ref(false)
const detailSubmitting = ref(false)
const selected = ref<TimetableItem | null>(null)
const remarkDraft = ref('')

function openDetail(row: TimetableItem): void {
  selected.value = row
  remarkDraft.value = row.remark
  detailVisible.value = true
}

async function saveRemark(): Promise<void> {
  if (!selected.value) return
  detailSubmitting.value = true
  try {
    const updated = await updateTimetableRemarkApi(selected.value.id, remarkDraft.value.trim())
    const index = timetable.value.findIndex((row) => row.id === updated.id)
    if (index >= 0) timetable.value[index] = updated
    selected.value = updated
    ElMessage.success('课表备注已更新')
    detailVisible.value = false
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    detailSubmitting.value = false
  }
}

/* ===== 今日课程队列 + 录音演示 ===== */
const recordingVisible = ref(false)
const recordingState = ref<'idle' | 'recording'>('idle')
const recordingRow = ref<TimetableItem | null>(null)
const creating = ref(false)
const waveBars = ref<Array<{ height: number; delay: number }>>([])

function makeWaveBars(): void {
  waveBars.value = Array.from({ length: 9 }, (_, index) => ({
    height: 26 + Math.round(Math.random() * 54),
    delay: Number((index * 0.08).toFixed(2))
  }))
}

function startClass(row: TimetableItem): void {
  recordingRow.value = row
  recordingState.value = 'idle'
  makeWaveBars()
  recordingVisible.value = true
}

function stopRecording(): void {
  recordingState.value = 'idle'
  recordingVisible.value = false
}

async function toggleRecording(): Promise<void> {
  if (recordingState.value === 'idle') {
    recordingState.value = 'recording'
    return
  }
  const row = recordingRow.value
  recordingState.value = 'idle'
  recordingVisible.value = false
  if (!row) return
  creating.value = true
  try {
    await createSessionApi({
      courseId: row.courseId,
      classId: row.classId,
      sessionDate: todayKey,
      period: periodLabel(row),
      topic: ''
    })
    ElMessage.success('已创建授课记录，补传录音后将触发 AI 评分')
  } catch {
    // 错误提示由 http 拦截器统一处理
  } finally {
    creating.value = false
  }
}

const recordingMeta = computed(() => {
  const row = recordingRow.value
  if (!row) return null
  return {
    courseName: row.courseName,
    className: row.className,
    classroom: row.classroom || '教室待补',
    period: periodLabel(row)
  }
})

onMounted(() => {
  void load()
})

watch(semester, () => {
  void load()
})
</script>

<template>
  <div class="workbench">
    <PageHeader title="教师工作台" :subtitle="`${todayLabel}，先看今天，再安排本周`">
      <template #actions>
        <span class="workbench__week-meta tabular-nums">本周 {{ weekCourseCount }} 节课</span>
      </template>
    </PageHeader>

    <el-skeleton v-if="loading" :rows="10" animated />

    <EmptyState v-else-if="error" description="课表加载失败">
      <template #action>
        <el-button @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <template v-else>
      <section class="workbench__block" aria-label="周课表">
        <header class="workbench__block-head">
          <div>
            <h2 class="workbench__block-title">本周课表</h2>
            <p class="workbench__block-sub">横轴周一至周日，纵轴 1-11 节；点击课程块查看教室并编辑备注。</p>
          </div>
          <span class="workbench__today-hint">今天（{{ WEEK_LABELS[todayWeekday - 1] }}）</span>
        </header>

        <div class="week-scroll">
          <div class="week-grid">
            <div class="week-grid__corner" aria-hidden="true">节次</div>
            <div
              v-for="day in weekDays"
              :key="`head-${day.key}`"
              class="week-grid__day-head"
              :class="{ 'week-grid__day-head--today': day.isToday }"
            >
              <span class="week-grid__day-label">{{ day.label }}</span>
              <span class="week-grid__day-date tabular-nums">{{ day.day }}</span>
            </div>

            <div
              v-for="section in SECTIONS"
              :key="section.label"
              class="week-grid__section"
              :style="{ gridRow: `${section.start + 1} / ${section.end + 2}` }"
            >
              {{ section.label }}
            </div>

            <template v-for="period in PERIODS" :key="`p-${period}`">
              <div class="week-grid__period tabular-nums" :style="{ gridRow: period + 1 }">
                {{ period }}
              </div>
              <div
                v-for="day in weekDays"
                :key="`cell-${day.weekday}-${period}`"
                class="week-grid__cell"
                :class="{ 'week-grid__cell--today': day.isToday }"
                :style="{ gridColumn: day.weekday + 2, gridRow: period + 1 }"
                aria-hidden="true"
              />
            </template>

            <button
              v-for="row in scheduledRows"
              :key="row.id"
              type="button"
              class="course-block"
              :class="`course-block--${toneOf(row)}`"
              :style="blockStyle(row)"
              :aria-label="`${row.courseName} ${periodLabel(row)} ${row.className}，查看详情`"
              @click="openDetail(row)"
            >
              <span class="course-block__name">{{ row.courseName }}</span>
              <span class="course-block__meta">{{ row.className }}</span>
              <span v-if="row.periodEnd - row.periodStart >= 1" class="course-block__room">
                {{ row.classroom || '教室待补' }}
              </span>
            </button>
          </div>
        </div>

        <p v-if="!scheduledRows.length" class="workbench__empty-note">
          课表尚未维护。请联系教务或主任补充课程时间。
        </p>
      </section>

      <div class="workbench__columns">
        <section class="workbench__block" aria-label="今日课程">
          <header class="workbench__block-head">
            <div>
              <h2 class="workbench__block-title">今日课程</h2>
              <p class="workbench__block-sub">点击「开始上课」进入录音演示，下课后自动创建授课记录。</p>
            </div>
            <span class="workbench__count tabular-nums">{{ todayRows.length }} 节</span>
          </header>

          <ol v-if="todayRows.length" class="today-list">
            <li v-for="(row, index) in todayRows" :key="row.id" class="today-row">
              <div class="today-row__rail">
                <span class="today-row__period tabular-nums">{{ periodLabel(row) }}</span>
                <span class="today-row__time">{{ row.classroom || '教室待补' }}</span>
              </div>
              <div class="today-row__main">
                <p class="today-row__course">{{ row.courseName }}</p>
                <p class="today-row__meta">
                  {{ row.className }}
                  <template v-if="row.remark">，{{ row.remark }}</template>
                </p>
              </div>
              <el-button
                :type="index === 0 ? 'primary' : 'default'"
                :plain="index > 0"
                :loading="creating && recordingRow?.id === row.id"
                @click="startClass(row)"
              >
                开始上课
              </el-button>
            </li>
          </ol>
          <p v-else class="workbench__empty-note">今天没有排课，可以从本周课表查看后续安排。</p>
        </section>

        <ClassroomIssueBoard />
      </div>
    </template>

    <!-- 课程块详情：教室 / 班级 / 备注 -->
    <el-dialog v-model="detailVisible" title="课程详情" width="480px" append-to-body>
      <template v-if="selected">
        <dl class="workbench__detail">
          <div>
            <dt>课程</dt>
            <dd>{{ selected.courseName }}（{{ selected.courseCode }}）</dd>
          </div>
          <div>
            <dt>班级</dt>
            <dd>{{ selected.className }}</dd>
          </div>
          <div>
            <dt>时间</dt>
            <dd>{{ WEEK_LABELS[selected.weekday - 1] }} {{ periodLabel(selected) }}</dd>
          </div>
          <div>
            <dt>教室</dt>
            <dd>{{ selected.classroom || '待补' }}</dd>
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
              placeholder="例如：本周安排随堂练习"
            />
          </el-form-item>
        </el-form>
      </template>
      <template #footer>
        <el-button @click="detailVisible = false">取消</el-button>
        <el-button type="primary" :loading="detailSubmitting" @click="saveRemark">保存备注</el-button>
      </template>
    </el-dialog>

    <!-- 录音演示：假波形 + 上课/下课 -->
    <el-dialog
      v-model="recordingVisible"
      title="课堂录音演示"
      width="520px"
      append-to-body
      aria-label="课堂录音演示"
      @closed="stopRecording"
    >
      <div class="recorder">
        <p v-if="recordingMeta" class="recorder__meta">
          <span>{{ recordingMeta.courseName }}</span>
          <span>{{ recordingMeta.className }}</span>
          <span>{{ recordingMeta.period }}</span>
          <span>{{ recordingMeta.classroom }}</span>
        </p>

        <div class="recorder__wave" :class="{ 'recorder__wave--active': recordingState === 'recording' }" aria-hidden="true">
          <span
            v-for="(bar, index) in waveBars"
            :key="index"
            class="recorder__bar"
            :style="{ '--bar-height': `${bar.height}px`, '--bar-delay': `${bar.delay}s` }"
          />
        </div>

        <p class="recorder__hint" role="status">
          {{ recordingState === 'recording' ? '正在记录课堂声音（演示）' : '点击「上课」开始记录' }}
        </p>

        <button
          type="button"
          class="recorder__button"
          :class="{ 'recorder__button--stop': recordingState === 'recording' }"
          :aria-label="recordingState === 'recording' ? '下课并创建授课记录' : '上课，开始录音演示'"
          @click="toggleRecording"
        >
          {{ recordingState === 'recording' ? '下课' : '上课' }}
        </button>
        <p class="recorder__note">演示模式不会调用真实麦克风，下课后仅创建授课记录。</p>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.workbench {
  &__week-meta {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__block {
    padding: var(--spacing-5) var(--spacing-6);
    background-color: var(--color-bg-card);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);

    & + & {
      margin-top: var(--spacing-4);
    }
  }

  &__block-head {
    display: flex;
    gap: var(--spacing-4);
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: var(--spacing-4);
  }

  &__block-title {
    font-size: var(--font-size-lg);
  }

  &__block-sub {
    margin-top: 4px;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__today-hint,
  &__count {
    flex-shrink: 0;
    font-size: var(--font-size-sm);
    color: var(--color-primary);
  }

  &__columns {
    display: grid;
    grid-template-columns: minmax(0, 1.25fr) minmax(0, 1fr);
    gap: var(--spacing-4);
    margin-top: var(--spacing-4);

    .workbench__block + .workbench__block {
      margin-top: 0;
    }
  }

  &__empty-note {
    padding: var(--spacing-4) 0;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
  }

  &__detail {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--spacing-4);
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
}

.week-scroll {
  overflow-x: auto;
}

.week-grid {
  display: grid;
  grid-template-columns: 34px 44px repeat(7, minmax(104px, 1fr));
  grid-template-rows: 48px repeat(11, 52px);
  min-width: 860px;
  overflow: hidden;
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-md);

  &__corner {
    display: flex;
    grid-column: 1 / 3;
    grid-row: 1;
    align-items: center;
    justify-content: center;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    background-color: var(--color-bg-page);
    border-bottom: 1px solid var(--color-divider);
  }

  &__day-head {
    display: flex;
    grid-row: 1;
    gap: var(--spacing-2);
    align-items: baseline;
    justify-content: center;
    background-color: var(--color-bg-page);
    border-bottom: 1px solid var(--color-divider);
    border-left: 1px solid var(--color-divider);

    &--today {
      background-color: var(--color-primary-bg);

      .week-grid__day-label,
      .week-grid__day-date {
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

  &__section {
    display: flex;
    grid-column: 1;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.12em;
    color: var(--color-text-tertiary);
    background-color: var(--color-bg-page);
    border-right: 1px solid var(--color-divider);
    writing-mode: vertical-rl;
  }

  &__period {
    display: flex;
    grid-column: 2;
    align-items: center;
    justify-content: center;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    background-color: var(--color-bg-page);
    border-top: 1px solid var(--color-divider);
    border-right: 1px solid var(--color-divider);
  }

  &__cell {
    background-color: var(--color-bg-card);
    border-top: 1px solid var(--color-divider);
    border-left: 1px solid var(--color-divider);

    &--today {
      background-color: color-mix(in srgb, var(--color-primary) 4%, var(--color-bg-card));
    }
  }
}

.course-block {
  --block-color: var(--color-primary);
  --block-bg: var(--color-primary-bg);

  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 1px;
  align-items: flex-start;
  padding: 5px 7px;
  margin: 3px;
  overflow: hidden;
  font-size: var(--font-size-xs);
  line-height: 1.35;
  text-align: left;
  color: var(--color-text-primary);
  cursor: pointer;
  background-color: var(--block-bg);
  border: none;
  border-left: 3px solid var(--block-color);
  border-radius: var(--radius-sm);
  transition: box-shadow var(--duration-fast) var(--ease-out-soft);

  &:hover,
  &:focus-visible {
    box-shadow: 0 3px 10px color-mix(in srgb, var(--block-color) 22%, transparent);
    outline: 2px solid var(--block-color);
    outline-offset: 1px;
  }

  &--primary {
    --block-color: var(--color-primary);
    --block-bg: var(--color-primary-bg);
  }

  &--info {
    --block-color: var(--color-info);
    --block-bg: var(--color-teacher-bg);
  }

  &--ai {
    --block-color: var(--color-ai);
    --block-bg: var(--color-ai-bg);
  }

  &--warning {
    --block-color: var(--color-warning);
    --block-bg: color-mix(in srgb, var(--color-warning) 13%, #ffffff);
  }

  &__name {
    max-width: 100%;
    overflow: hidden;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__meta,
  &__room {
    max-width: 100%;
    overflow: hidden;
    font-size: 10px;
    color: var(--color-text-tertiary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.today-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.today-row {
  display: flex;
  gap: var(--spacing-4);
  align-items: center;
  padding: var(--spacing-3) var(--spacing-4);
  background-color: var(--color-bg-page);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-md);

  &__rail {
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    gap: 2px;
    width: 76px;
  }

  &__period {
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--color-primary);
  }

  &__time {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__main {
    flex: 1;
    min-width: 0;
  }

  &__course {
    font-size: var(--font-size-base);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__meta {
    margin-top: 2px;
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }
}

.recorder {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-4);
  align-items: center;
  padding: var(--spacing-2) 0 var(--spacing-4);

  &__meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-1) var(--spacing-3);
    justify-content: center;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
    text-align: center;
  }

  &__wave {
    display: flex;
    gap: 7px;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 120px;
    padding: var(--spacing-4);
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-lg);
  }

  &__bar {
    width: 8px;
    height: var(--bar-height);
    background: var(--gradient-ai);
    border-radius: 999px;
    opacity: 0.45;
    transform: scaleY(0.55);
    transition: transform 0.25s ease, opacity 0.25s ease;
  }

  &__wave--active &__bar {
    opacity: 1;
    animation: wave-jump 1.1s ease-in-out infinite;
    animation-delay: var(--bar-delay);
  }

  &__hint {
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
  }

  &__button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 88px;
    height: 88px;
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: #ffffff;
    cursor: pointer;
    background: var(--gradient-ai);
    border: none;
    border-radius: 50%;
    box-shadow: 0 8px 24px color-mix(in srgb, var(--color-ai) 32%, transparent);
    transition: transform var(--duration-fast) var(--ease-out-soft), box-shadow var(--duration-fast) var(--ease-out-soft);

    &:hover,
    &:focus-visible {
      box-shadow: 0 10px 28px color-mix(in srgb, var(--color-ai) 42%, transparent);
      outline: 3px solid var(--color-ai-deep);
      outline-offset: 3px;
    }

    &:active {
      transform: scale(0.97);
    }

    &--stop {
      background: var(--color-error);
      box-shadow: 0 8px 24px color-mix(in srgb, var(--color-error) 32%, transparent);
    }
  }

  &__note {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }
}

@keyframes wave-jump {
  0%,
  100% {
    transform: scaleY(0.5);
  }
  50% {
    transform: scaleY(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .recorder__wave--active .recorder__bar {
    animation: none;
    transform: scaleY(0.85);
  }
}

@media (max-width: 1100px) {
  .workbench__columns {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 960px) {
  .workbench__block-head {
    flex-direction: column;
  }

  .today-row {
    flex-wrap: wrap;
  }
}
</style>
