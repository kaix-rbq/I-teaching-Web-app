import type { EvaluationDTO } from './evaluation'

export interface ScoreDimension {
  key: string
  name: string
  weight: number
  isObservation: boolean
  score: number | null
  supervisorScore: number | null
  agentScore: number | null
}
export interface ScoreSummary {
  compositeScore: number | null
  supervisorScore: number | null
  agentScore: number | null
  dimensions: ScoreDimension[]
  sample: { sessionCount: number; evaluatedCount: number; supervisorCount: number; agentCount: number; alignedCount: number; sampleSufficient: boolean }
  flags: string[]
  weights: { supervisor: number; agent: number }
  formulaVersion: string
}
export interface TeacherScoreItem extends ScoreSummary { teacherId: number; teacherName: string; jobNo: string }
export interface TeacherSummary extends ScoreSummary { teacherId: number; teacherName: string; semester: string; courses: Array<ScoreSummary & { courseId: number; courseCode: string; courseName: string }> }
export interface TeacherTimelineItem { sessionId: number; sessionDate: string; period: string; topic: string; courseId: number; courseCode: string; courseName: string; status: string; compositeScore: number | null; supervisorScore: number | null; agentScore: number | null; supervisorEvaluations: EvaluationDTO[]; agentEvaluation: EvaluationDTO | null }

/** 趋势序列中的单个数据点；value 为 null 表示该课次无评价（折线断开，不补 0） */
export interface ScoreTrendPoint {
  /** 横轴标签，如 "第 1 次课" */
  label: string
  /** YYYY-MM-DD */
  date: string
  value: number | null
}

/** 单条维度趋势序列（GET /teachers/:id/score-trend 的响应项，阶段③未实现） */
export interface ScoreTrendSeries {
  key: string
  name: string
  isObservation: boolean
  points: ScoreTrendPoint[]
}
