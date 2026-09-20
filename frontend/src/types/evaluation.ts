/**
 * 评分域类型（Sprint 2 阶段①）。
 * 与后端 internal/dto/evaluation.go 逐字对齐：字段名即后端 JSON tag，缺失侧一律 null。
 */

export type SessionStatus = 'scheduled' | 'recorded' | 'evaluated'
export type EvaluatorType = 'supervisor' | 'agent'

/** 5 个评分维度 key，一旦上线不得增删（开发计划 §2.1） */
export type DimensionKey =
  | 'objective'
  | 'content'
  | 'interaction'
  | 'organization'
  | 'frontier'

/** 课程历史授课记录行（GET /courses/:id/sessions） */
export interface SessionListItem {
  id: number
  /** YYYY-MM-DD */
  sessionDate: string
  /** 节次，如 "3-4 节" */
  period: string
  topic: string
  status: SessionStatus
  /** 单次总分（仅展示/排序；无评价为 null，不得显示 0） */
  supervisorScore: number | null
  agentScore: number | null
  evaluationCount: number
}

/** 单场次基本信息（GET /sessions/:id 与创建后的响应） */
export interface SessionDetail {
  id: number
  courseId: number
  courseCode: string
  courseName: string
  /** 0 = 未指定班级 */
  classId: number
  className: string
  teacherId: number
  teacherName: string
  semester: string
  sessionDate: string
  period: string
  topic: string
  status: SessionStatus
}

/** 一条评价的完整展示（督导表单回填 / 智能体参考共用） */
export interface EvaluationDTO {
  evaluatorId: number
  evaluatorName: string
  evaluatorType: EvaluatorType
  aiModelVersion: string
  aiConfidence: number | null
  formulaVersion: string
  /** 1-5；智能体侧 objective 恒为 null */
  objective: number | null
  content: number | null
  interaction: number | null
  organization: number | null
  frontier: number | null
  totalScore: number | null
  comment: string
  /** 亮点 */
  highlights: string
  /** 待改进 */
  improvements: string
  /** 建议 / 智能体提优建议 */
  suggestions: string
  createdAt: string
  updatedAt: string
}

/** GET /sessions/:id/evaluation 聚合响应（当堂课评估页） */
export interface SessionEvaluation {
  session: SessionDetail
  /** 一次课可多督导 */
  supervisorScores: EvaluationDTO[]
  /** 阶段一恒为 null */
  agentScore: EvaluationDTO | null
}

/** 督导评分提交体（PUT /sessions/:id/supervisor-evaluation，五维必填） */
export interface SupervisorEvaluationPayload {
  objective: number
  content: number
  interaction: number
  organization: number
  frontier: number
  comment?: string
  highlights?: string
  improvements?: string
  suggestions?: string
}

/** 创建授课记录提交体（POST /sessions，仅督导） */
export interface SessionCreatePayload {
  courseId: number
  /** 0 / 不传 = 未指定班级 */
  classId?: number
  /** YYYY-MM-DD，不得晚于今天 */
  sessionDate: string
  period: string
  topic?: string
  planId?: number
}

/** 督导评分表单模型（未选择维度为 null，提交前校验五维必填） */
export interface EvaluationFormModel {
  objective: number | null
  content: number | null
  interaction: number | null
  organization: number | null
  frontier: number | null
  comment: string
  highlights: string
  improvements: string
  suggestions: string
}
