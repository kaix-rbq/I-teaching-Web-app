import type { Role } from '@/types/user'
import type { CourseStatus } from '@/types/course'
import type { ResourceType } from '@/types/resource'
import type { SupervisionStatus } from '@/types/supervision'
import type { DimensionKey, SessionStatus } from '@/types/evaluation'

export const TOKEN_KEY = 'aijiaoxue_token'
export const USER_KEY = 'aijiaoxue_user'

export interface RoleMeta {
  value: Role
  label: string
  shortLabel: string
  color: string
}

export const ROLES: RoleMeta[] = [
  { value: 'director', label: '教研室主任', shortLabel: '主任', color: '#2f54eb' },
  { value: 'teacher', label: '教师', shortLabel: '教师', color: '#13c2c2' },
  { value: 'supervisor', label: '教学督导', shortLabel: '督导', color: '#722ed1' }
]

export function getRoleMeta(role?: Role): RoleMeta | undefined {
  return ROLES.find((item) => item.value === role)
}

export interface OptionMeta<T extends string> {
  value: T
  label: string
}

export const COURSE_STATUS: OptionMeta<CourseStatus>[] = [
  { value: 'open', label: '开课中' },
  { value: 'draft', label: '草稿' },
  { value: 'closed', label: '已结课' }
]

export const SUPERVISION_STATUS: OptionMeta<SupervisionStatus>[] = [
  { value: 'planned', label: '计划中' },
  { value: 'completed', label: '已完成' }
]

export type TagType = 'success' | 'warning' | 'info' | 'primary' | 'danger'

export function getCourseStatusMeta(status: CourseStatus): {
  label: string
  tag: TagType
} {
  const map: Record<CourseStatus, { label: string; tag: TagType }> = {
    open: { label: '开课中', tag: 'success' },
    draft: { label: '草稿', tag: 'warning' },
    closed: { label: '已结课', tag: 'info' }
  }
  return map[status]
}

export function getSupervisionStatusMeta(status: SupervisionStatus): {
  label: string
  tag: TagType
} {
  const map: Record<SupervisionStatus, { label: string; tag: TagType }> = {
    planned: { label: '计划中', tag: 'warning' },
    completed: { label: '已完成', tag: 'success' }
  }
  return map[status]
}

export const RESOURCE_ACCEPT = '.pdf,.doc,.docx,.ppt,.pptx,.mp4,.zip'
export const MAX_RESOURCE_SIZE = 100 * 1024 * 1024

export const RESOURCE_TYPE_LABELS: Record<ResourceType, string> = {
  pdf: 'PDF',
  doc: '文档',
  ppt: '演示文稿',
  video: '视频',
  zip: '压缩包',
  other: '其他'
}

export const PAGE_SIZES = [10, 20, 50]

/** 学期字典：Sprint 1 由前端常量维护，与后端 internal/service/consts.go 的 Semesters 同源 */
export const SEMESTERS = ['2026-2027-1', '2025-2026-2', '2025-2026-1']

/** 当前学期（工作台/覆盖率口径），与后端 CurrentSemester 一致 */
export const CURRENT_SEMESTER = SEMESTERS[0]

export function resolveResourceType(fileName: string): ResourceType {
  const ext = fileName.slice(fileName.lastIndexOf('.') + 1).toLowerCase()
  if (ext === 'pdf') return 'pdf'
  if (ext === 'doc' || ext === 'docx') return 'doc'
  if (ext === 'ppt' || ext === 'pptx') return 'ppt'
  if (ext === 'mp4' || ext === 'avi' || ext === 'mov') return 'video'
  if (ext === 'zip' || ext === 'rar' || ext === '7z') return 'zip'
  return 'other'
}

/* ===== Sprint 2 阶段①：授课记录与评分 ===== */

export const SESSION_STATUS: OptionMeta<SessionStatus>[] = [
  { value: 'scheduled', label: '待上课' },
  { value: 'recorded', label: '已录音' },
  { value: 'evaluated', label: '已评价' }
]

export function getSessionStatusMeta(status: SessionStatus): {
  label: string
  tag: TagType
} {
  const map: Record<SessionStatus, { label: string; tag: TagType }> = {
    scheduled: { label: '待评价', tag: 'warning' },
    recorded: { label: '待评价', tag: 'warning' },
    evaluated: { label: '已评价', tag: 'success' }
  }
  return map[status]
}

/** 评分维度元数据（权重与后端 config 同源，仅用于展示，前端不参与计分） */
export interface DimensionMeta {
  key: DimensionKey
  name: string
  /** 雷达图等紧凑场景的短标签 */
  shortName: string
  weight: number
  /** 观测项：权重为 0，仅作亮点展示，不参与加权总分 */
  isObservation: boolean
  description: string
}

export const EVALUATION_DIMENSIONS: DimensionMeta[] = [
  {
    key: 'objective',
    name: '教学目标与内容准确性',
    shortName: '目标内容',
    weight: 0.3,
    isObservation: false,
    description: '讲对了没有是底线维度'
  },
  {
    key: 'content',
    name: '内容质量与深度',
    shortName: '内容深度',
    weight: 0.3,
    isObservation: false,
    description: '有干货不水课'
  },
  {
    key: 'interaction',
    name: '学生互动与参与',
    shortName: '互动参与',
    weight: 0.2,
    isObservation: false,
    description: '有效提问与讨论、学生参与度'
  },
  {
    key: 'organization',
    name: '课堂组织与节奏',
    shortName: '组织节奏',
    weight: 0.2,
    isObservation: false,
    description: '环节清晰、时间分配合理（不以音量低为正向证据）'
  },
  {
    key: 'frontier',
    name: '前沿与交叉学科',
    shortName: '前沿交叉',
    weight: 0,
    isObservation: true,
    description: '观测项：不计入加权总分，仅作亮点展示（基础课不因无前沿扣分）'
  }
]

/** 通用评分锚点（每个维度每个分值都必须有可操作描述，见开发计划 §2.3） */
export interface ScoreLevel {
  value: number
  level: string
  generic: string
}

export const SCORE_LEVELS: ScoreLevel[] = [
  { value: 5, level: '优秀', generic: '可作为示范课' },
  { value: 4, level: '良好', generic: '达到骨干教师水平' },
  { value: 3, level: '合格', generic: '达到基本教学要求' },
  { value: 2, level: '待改进', generic: '存在明显短板' },
  { value: 1, level: '不合格', generic: '需要立即干预' }
]

/** 维度专属锚点：每个维度 1-5 分的行为描述，供 tooltip 呈现 */
export const DIMENSION_ANCHORS: Record<DimensionKey, Record<number, string>> = {
  objective: {
    5: '目标明确，内容准确无错误，重难点突出',
    4: '目标清晰，内容准确，重难点较为突出',
    3: '目标基本清晰，无实质性知识错误',
    2: '目标不够清晰，或存在个别知识性瑕疵',
    1: '目标缺失或存在知识性错误'
  },
  content: {
    5: '内容充实有深度，理论联系实际，无水分',
    4: '内容较充实，理论与实际结合较好',
    3: '内容完整但以照本宣科为主',
    2: '内容偏薄，案例或深度不足',
    1: '内容空洞或明显偏离课程大纲'
  },
  interaction: {
    5: '有效提问与讨论充分，学生参与度高',
    4: '提问与讨论较多，多数学生参与',
    3: '偶有提问但以自问自答为主',
    2: '互动较少，学生参与度不足',
    1: '全程单向讲授，无任何互动'
  },
  organization: {
    5: '环节清晰，时间分配合理，节奏张弛有度',
    4: '环节清晰，时间分配基本合理',
    3: '环节完整但时间分配略显失衡',
    2: '环节不够完整或时间分配明显失衡',
    1: '结构混乱或严重拖堂/提前下课'
  },
  frontier: {
    5: '自然融入学科前沿或交叉应用，与主线结合紧密',
    4: '较好结合学科前沿或交叉应用',
    3: '提及前沿但较生硬',
    2: '前沿内容较少且结合生硬（不单独作为扣分依据）',
    1: '无前沿内容（不单独作为扣分依据）'
  }
}

/** 取某维度某分值的锚点描述，找不到时回退到通用锚点 */
export function getDimensionAnchor(dimension: DimensionKey, score: number): string {
  return DIMENSION_ANCHORS[dimension]?.[score] ?? getScoreLevel(score)?.generic ?? ''
}

export function getScoreLevel(score: number): ScoreLevel | undefined {
  return SCORE_LEVELS.find((item) => item.value === score)
}
