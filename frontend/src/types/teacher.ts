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
