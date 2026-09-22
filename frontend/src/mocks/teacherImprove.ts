/**
 * 「教学提优」页面临时演示数据（前端 mock）。
 *
 * ⚠️ 说明：智能体接入（阶段②/③）与趋势接口尚未实现，本文件为**前端临时演示数据**，
 * 仅用于在真实接口就绪前驱动页面。真实接口列于开发计划 §4.2 / §4.4：
 *   - `GET  /teachers/:id/score-trend?semester=&dimension=`
 *   - `POST /agent/chat`（SSE 流式）
 * 后端就绪后应删除本文件，改由 `api/agent.ts` 发起真实请求（接口签名保持不变）。
 *
 * 演示数值取自开发计划 §3.4 种子数据，保证「逐次上升」的趋势可验收。
 */
import type { AgentSuggestion } from '@/types/agent'
import type { ScoreTrendSeries } from '@/types/teacher'

/** 演示用课次（与种子数据一致，按时间正序） */
const DEMO_SESSIONS = [
  { sessionId: 1, sessionDate: '2026-09-12', period: '3-4 节', topic: '项目立项与章程' },
  { sessionId: 2, sessionDate: '2026-09-19', period: '3-4 节', topic: '需求调研与用户故事' },
  { sessionId: 3, sessionDate: '2026-09-26', period: '3-4 节', topic: '迭代计划与估点' }
] as const

/** 各维度逐次观测分（0-100 制，与聚合接口一致；frontier 为观测项） */
const DIMENSION_TREND: Record<string, { name: string; isObservation: boolean; values: number[] }> = {
  objective: { name: '教学目标与内容准确性', isObservation: false, values: [80, 80, 100] },
  content: { name: '内容质量与深度', isObservation: false, values: [60, 80, 80] },
  interaction: { name: '学生互动与参与', isObservation: false, values: [40, 60, 80] },
  organization: { name: '课堂组织与节奏', isObservation: false, values: [60, 80, 80] },
  frontier: { name: '前沿与交叉学科', isObservation: true, values: [40, 60, 60] }
}

/**
 * 维度历史趋势（课程级）。
 * 返回首项为「综合分」，其余为五个维度，供前端按维度切换折线。
 */
export function mockScoreTrend(): ScoreTrendSeries[] {
  const dates = DEMO_SESSIONS.map((item) => item.sessionDate)
  const labels = DEMO_SESSIONS.map((_, index) => `第 ${index + 1} 次课`)

  const composite: ScoreTrendSeries = {
    key: 'composite',
    name: '综合分',
    isObservation: false,
    points: [58.75, 70.0, 82.5].map((value, index) => ({
      label: labels[index],
      date: dates[index],
      value
    }))
  }

  const dimensions: ScoreTrendSeries[] = Object.entries(DIMENSION_TREND).map(
    ([key, meta]) => ({
      key,
      name: meta.name,
      isObservation: meta.isObservation,
      points: meta.values.map((value, index) => ({
        label: labels[index],
        date: dates[index],
        value
      }))
    })
  )

  return [composite, ...dimensions]
}

/** 智能体提优建议（按时间倒序，标注对应课次） */
export function mockAgentSuggestions(courseId: number): AgentSuggestion[] {
  const items: AgentSuggestion[] = [
    {
      id: 3,
      courseId,
      sessionId: 3,
      sessionDate: '2026-09-26',
      period: '3-4 节',
      topic: '迭代计划与估点',
      summary:
        '本次课估点练习设计贴近实战，建议把练习拆成「先独立估点、再小组比对」两步，让每位学生都先形成自己的判断，再通过讨论校准。',
      highlights: '练习设计贴近实战，学生参与度高',
      improvements: '课堂末尾节奏偏紧，总结略显仓促',
      evidence:
        '转写片段：「大家先自己估一下……好，现在和旁边同学对一下，看差多少。」（学生 A、B 的讨论占本环节约 6 分钟）',
      modelVersion: 'qwen-audio-v1',
      confidence: 0.75,
      createdAt: '2026-09-26T18:20:00'
    },
    {
      id: 2,
      courseId,
      sessionId: 2,
      sessionDate: '2026-09-19',
      period: '3-4 节',
      topic: '需求调研与用户故事',
      summary:
        '小组讨论组织得当，但部分小组偏离主题。建议为每组设定明确的产出物（如一张用户故事卡），并在讨论前用 1 分钟说明验收标准。',
      highlights: '用户故事讲解透彻，小组讨论有效',
      improvements: '个别小组讨论偏离主题，缺少明确产出物',
      evidence:
        '转写片段：「你们组先把用户角色定下来，再写故事。」（教师在第 3、5 组间反复提醒）',
      modelVersion: 'qwen-audio-v1',
      confidence: 0.68,
      createdAt: '2026-09-19T18:05:00'
    },
    {
      id: 1,
      courseId,
      sessionId: 1,
      sessionDate: '2026-09-12',
      period: '3-4 节',
      topic: '项目立项与章程',
      summary:
        '开篇结构完整，但提问后等待时间不足（平均不足 2 秒），建议将等待时间延长到 3-5 秒，给学生组织语言的空间，互动维度可明显提升。',
      highlights: '课程框架清晰，项目章程要素讲解完整',
      improvements: '提问后等待时间不足，学生回应多为简单附和',
      evidence:
        '转写片段：「这个章程里最关键的是哪一项？……对，是范围。」（自问自答，等待约 1.5 秒）',
      modelVersion: 'qwen-audio-v1',
      confidence: 0.72,
      createdAt: '2026-09-12T18:10:00'
    }
  ]

  return items.sort((a, b) => (a.sessionDate < b.sessionDate ? 1 : -1))
}

/** 智能体对话演示回复：按关键词命中，未命中时给通用改进建议 */
export function mockAgentChatReply(question: string): string {
  const text = question.trim()

  if (/互动|提问|参与/.test(text)) {
    return '针对「学生互动」：你本次课的有效提问比例约 35%，高于上一轮。建议在抛出问题后保持 3-5 秒沉默，并把「自问自答」改成「点名学生 + 追问一层」。可在下一节课挑 2-3 个关键节点刻意练习。'
  }
  if (/节奏|时间|组织|拖堂/.test(text)) {
    return '针对「课堂组织与节奏」：练习环节实际用时超出计划约 6 分钟，导致总结被压缩。建议把练习拆成「限时独立完成 + 小组比对」两段，各设 5 分钟并显式报时。'
  }
  if (/内容|深度|案例/.test(text)) {
    return '针对「内容质量与深度」：本次课理论讲解完整，但真实项目案例偏少。建议每个知识点配 1 个近两年的行业案例，并让学生判断「如果是你会怎么做」，把内容深度与互动一起带起来。'
  }
  if (/前沿|交叉/.test(text)) {
    return '针对「前沿与交叉学科」：本次课未涉及学科前沿，但该维度为观测项、不单独扣分。若想用作亮点，可在引入环节用 2 分钟关联一个最新研究或行业动态即可，不必强求。'
  }
  if (/趋势|进步|提升/.test(text)) {
    return '从近 3 次课看，你的综合分由 58.75 → 70.00 → 82.50 稳步上升，互动维度提升最明显（40 → 60 → 80）。建议保持当前的提问—讨论设计，并重点补齐「内容深度」的案例支撑。'
  }

  return '收到。结合本次课的转写与评分，我建议优先关注「学生互动」与「内容深度」两个维度：互动上延长提问后的等待时间，内容上每个知识点补充一个真实案例。需要我针对某个维度给出更具体的课堂动作清单吗？'
}
