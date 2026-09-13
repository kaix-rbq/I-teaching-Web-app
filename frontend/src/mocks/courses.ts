import type { ApiResponse, PageResult } from '@/types/api'
import type { ClassInfo, Course, CourseStatus } from '@/types/course'
import type { Resource, ResourceType } from '@/types/resource'
import { resolveResourceType } from '@/constants'
import { findUserById } from './auth'
import { route, type MockRoute } from './router'

export const DEPARTMENTS = [
  '软件工程教研室',
  '计算机科学与技术教研室',
  '网络工程教研室',
  '人工智能教研室'
]

export const SEMESTERS = ['2026-2027-1', '2025-2026-2', '2025-2026-1']

const CURRENT_SEMESTER = SEMESTERS[0]
const OTHERS = SEMESTERS.slice(1)

interface ClassSeed {
  className: string
  schedule: string
  location: string
  studentCount: number
}

interface CourseSeed {
  id: string
  code: string
  name: string
  credit: number
  hours: number
  teacherId: string
  department: string
  status: CourseStatus
  semester?: string
  description: string
  objective: string
  major: string
  classes: ClassSeed[]
  resources: { name: string; type: ResourceType; size: number; uploader: string; uploadedAt: string }[]
}

function cls(
  className: string,
  schedule: string,
  location: string,
  studentCount: number
): ClassSeed {
  return { className, schedule, location, studentCount }
}

const COURSE_SEEDS: CourseSeed[] = [
  {
    id: 'c-001',
    code: 'SE3101',
    name: '软件工程导论',
    credit: 3,
    hours: 48,
    teacherId: 'u-teacher-1',
    department: '软件工程教研室',
    status: 'open',
    description:
      '本课程系统介绍软件工程的基本概念、软件生存周期与主流开发方法，帮助学生建立工程化软件开发的整体认识。',
    objective: '掌握软件工程基本理论与方法，能够运用结构化与面向对象方法完成小型软件项目。',
    major: '软件工程 / 计算机科学与技术',
    classes: [
      cls('软件 2201', '周一 3-4 节', '逸夫楼 301', 42),
      cls('软件 2202', '周三 1-2 节', '逸夫楼 302', 40)
    ],
    resources: [
      { name: '软件工程导论-教学大纲.pdf', type: 'pdf', size: 2_412_544, uploader: '李文娟', uploadedAt: '2026-08-20T02:15:00.000Z' },
      { name: '第1章 软件工程概述.pptx', type: 'ppt', size: 8_912_896, uploader: '李文娟', uploadedAt: '2026-08-22T06:30:00.000Z' },
      { name: '课程设计任务书.docx', type: 'doc', size: 324_608, uploader: '李文娟', uploadedAt: '2026-08-25T09:05:00.000Z' },
      { name: '案例讲解视频.mp4', type: 'video', size: 157_286_400, uploader: '李文娟', uploadedAt: '2026-09-01T01:40:00.000Z' }
    ]
  },
  {
    id: 'c-002',
    code: 'SE3102',
    name: '软件需求工程',
    credit: 2,
    hours: 32,
    teacherId: 'u-teacher-2',
    department: '软件工程教研室',
    status: 'open',
    description: '围绕需求获取、分析、规格说明与验证，讲解需求工程全流程及常用建模工具。',
    objective: '能够独立完成需求调研与需求规格说明书的编写，掌握 UML 用例与领域建模。',
    major: '软件工程',
    classes: [cls('软件 2203', '周二 3-4 节', '逸夫楼 205', 45)],
    resources: [
      { name: '需求规格说明书模板.docx', type: 'doc', size: 198_656, uploader: '赵鹏', uploadedAt: '2026-08-21T03:10:00.000Z' },
      { name: '需求分析案例集.pdf', type: 'pdf', size: 5_242_880, uploader: '赵鹏', uploadedAt: '2026-08-28T07:20:00.000Z' }
    ]
  },
  {
    id: 'c-003',
    code: 'SE3103',
    name: '软件设计与体系结构',
    credit: 3,
    hours: 48,
    teacherId: 'u-teacher-3',
    department: '软件工程教研室',
    status: 'open',
    description: '讲解软件体系结构风格、设计模式与架构评估方法，结合实例开展架构设计实践。',
    objective: '掌握常见架构风格与设计模式，能够针对业务场景完成架构方案设计与评审。',
    major: '软件工程',
    classes: [
      cls('软件 2201', '周四 1-2 节', '逸夫楼 401', 42),
      cls('软件 2203', '周五 3-4 节', '逸夫楼 402', 45)
    ],
    resources: [
      { name: '架构风格综述.pdf', type: 'pdf', size: 3_355_443, uploader: '孙晓雨', uploadedAt: '2026-08-19T08:00:00.000Z' },
      { name: '设计模式源码示例.zip', type: 'zip', size: 12_582_912, uploader: '孙晓雨', uploadedAt: '2026-09-02T02:25:00.000Z' }
    ]
  },
  {
    id: 'c-004',
    code: 'SE3201',
    name: '软件测试技术',
    credit: 2,
    hours: 32,
    teacherId: 'u-teacher-1',
    department: '软件工程教研室',
    status: 'draft',
    semester: CURRENT_SEMESTER,
    description: '介绍黑盒与白盒测试方法、测试用例设计、自动化测试框架与持续集成实践。',
    objective: '能够设计高质量测试用例，并使用主流工具搭建自动化测试流程。',
    major: '软件工程',
    classes: [cls('软件 2101', '周三 5-6 节', '逸夫楼 106', 38)],
    resources: [
      { name: '测试用例设计方法.pdf', type: 'pdf', size: 1_887_436, uploader: '李文娟', uploadedAt: '2026-09-05T05:15:00.000Z' }
    ]
  },
  {
    id: 'c-005',
    code: 'SE3202',
    name: '软件项目管理',
    credit: 2,
    hours: 32,
    teacherId: 'u-teacher-2',
    department: '软件工程教研室',
    status: 'closed',
    semester: OTHERS[1],
    description: '讲授软件项目进度、成本、质量与风险管理方法，结合 Scrum 开展项目实训。',
    objective: '掌握敏捷项目管理实践，能够制定并跟踪项目计划。',
    major: '软件工程',
    classes: [cls('软件 2102', '周二 5-6 节', '逸夫楼 108', 36)],
    resources: [
      { name: 'Scrum 实践指南.pdf', type: 'pdf', size: 4_194_304, uploader: '赵鹏', uploadedAt: '2026-03-10T06:20:00.000Z' }
    ]
  },
  {
    id: 'c-006',
    code: 'CS2101',
    name: '数据结构与算法',
    credit: 4,
    hours: 64,
    teacherId: 'u-teacher-4',
    department: '计算机科学与技术教研室',
    status: 'open',
    description: '系统讲解线性表、树、图等数据结构及其经典算法，培养算法分析与实现能力。',
    objective: '能够选择合适的数据结构与算法解决实际问题，并具备复杂度分析能力。',
    major: '计算机科学与技术',
    classes: [
      cls('计科 2201', '周一 1-2 节', '计算中心 A101', 50),
      cls('计科 2202', '周三 3-4 节', '计算中心 A102', 48),
      cls('计科 2203', '周五 1-2 节', '计算中心 A103', 47)
    ],
    resources: [
      { name: '数据结构讲义.pdf', type: 'pdf', size: 9_961_472, uploader: '陈立', uploadedAt: '2026-08-18T01:00:00.000Z' },
      { name: '算法练习题库.docx', type: 'doc', size: 512_000, uploader: '陈立', uploadedAt: '2026-08-30T03:30:00.000Z' },
      { name: '实验代码框架.zip', type: 'zip', size: 6_291_456, uploader: '陈立', uploadedAt: '2026-09-04T07:45:00.000Z' }
    ]
  },
  {
    id: 'c-007',
    code: 'CS2102',
    name: '计算机组成原理',
    credit: 4,
    hours: 64,
    teacherId: 'u-teacher-4',
    department: '计算机科学与技术教研室',
    status: 'open',
    description: '从数字逻辑出发，讲解计算机各部件的组成原理与协同工作机制。',
    objective: '理解 CPU、存储与输入输出系统的工作原理，能够进行简单指令系统设计。',
    major: '计算机科学与技术',
    classes: [cls('计科 2201', '周二 1-2 节', '计算中心 B201', 50)],
    resources: [
      { name: 'CPU 设计实验指导.pdf', type: 'pdf', size: 3_145_728, uploader: '陈立', uploadedAt: '2026-08-27T02:10:00.000Z' }
    ]
  },
  {
    id: 'c-008',
    code: 'CS3101',
    name: '操作系统',
    credit: 4,
    hours: 64,
    teacherId: 'u-teacher-4',
    department: '计算机科学与技术教研室',
    status: 'draft',
    description: '讲解进程管理、内存管理、文件系统与设备管理等操作系统核心内容。',
    objective: '掌握操作系统基本原理，能够分析并解决并发与资源调度问题。',
    major: '计算机科学与技术',
    classes: [cls('计科 2101', '周四 3-4 节', '计算中心 B105', 44)],
    resources: []
  },
  {
    id: 'c-009',
    code: 'NE2201',
    name: '计算机网络',
    credit: 3,
    hours: 48,
    teacherId: 'u-teacher-5',
    department: '网络工程教研室',
    status: 'open',
    description: '基于 TCP/IP 协议栈，讲解网络体系结构、协议原理与组网技术。',
    objective: '理解各层协议工作原理，能够完成小型网络的规划与配置。',
    major: '网络工程',
    classes: [
      cls('网络 2201', '周一 5-6 节', '信息楼 301', 41),
      cls('网络 2202', '周三 5-6 节', '信息楼 302', 39)
    ],
    resources: [
      { name: 'TCP-IP 详解笔记.pdf', type: 'pdf', size: 7_340_032, uploader: '周敏', uploadedAt: '2026-08-23T04:00:00.000Z' },
      { name: '抓包实验.pcapng.zip', type: 'zip', size: 15_728_640, uploader: '周敏', uploadedAt: '2026-09-06T06:10:00.000Z' }
    ]
  },
  {
    id: 'c-010',
    code: 'NE2202',
    name: '网络安全技术',
    credit: 3,
    hours: 48,
    teacherId: 'u-teacher-5',
    department: '网络工程教研室',
    status: 'open',
    description: '介绍密码学基础、身份认证、访问控制与常见网络攻防技术。',
    objective: '掌握网络安全防护基本方法，能够开展漏洞分析与安全加固。',
    major: '网络工程 / 信息安全',
    classes: [cls('网络 2203', '周五 5-6 节', '信息楼 305', 43)],
    resources: [
      { name: '网络安全实验手册.pdf', type: 'pdf', size: 4_718_592, uploader: '周敏', uploadedAt: '2026-08-29T08:40:00.000Z' }
    ]
  },
  {
    id: 'c-011',
    code: 'NE3301',
    name: '网络协议分析',
    credit: 2,
    hours: 32,
    teacherId: 'u-teacher-5',
    department: '网络工程教研室',
    status: 'closed',
    semester: OTHERS[0],
    description: '深入分析典型网络协议报文结构，结合 Wireshark 开展协议逆向分析。',
    objective: '能够独立完成协议抓包、解析与异常定位。',
    major: '网络工程',
    classes: [cls('网络 2101', '周二 7-8 节', '信息楼 210', 35)],
    resources: []
  },
  {
    id: 'c-012',
    code: 'AI2401',
    name: '机器学习基础',
    credit: 3,
    hours: 48,
    teacherId: 'u-teacher-6',
    department: '人工智能教研室',
    status: 'open',
    description: '讲解监督学习、无监督学习与模型评估方法，包含特征工程与调参实践。',
    objective: '掌握主流机器学习算法原理，能够完成完整建模流程。',
    major: '人工智能 / 数据科学',
    classes: [
      cls('智能 2201', '周一 7-8 节', '智能楼 401', 46),
      cls('智能 2202', '周四 5-6 节', '智能楼 402', 44)
    ],
    resources: [
      { name: '机器学习实验指导.pdf', type: 'pdf', size: 6_815_744, uploader: '吴桐', uploadedAt: '2026-08-24T03:00:00.000Z' },
      { name: '数据集与 Notebook.zip', type: 'zip', size: 52_428_800, uploader: '吴桐', uploadedAt: '2026-09-03T05:30:00.000Z' }
    ]
  },
  {
    id: 'c-013',
    code: 'AI2402',
    name: '深度学习导论',
    credit: 3,
    hours: 48,
    teacherId: 'u-teacher-6',
    department: '人工智能教研室',
    status: 'open',
    description: '介绍神经网络、卷积网络与循环网络的基本结构与训练方法。',
    objective: '能够使用主流框架搭建并训练深度模型解决视觉/文本任务。',
    major: '人工智能',
    classes: [cls('智能 2203', '周三 7-8 节', '智能楼 405', 45)],
    resources: [
      { name: 'CNN 原理讲解.pptx', type: 'ppt', size: 18_874_368, uploader: '吴桐', uploadedAt: '2026-08-31T02:50:00.000Z' }
    ]
  },
  {
    id: 'c-014',
    code: 'AI3401',
    name: '自然语言处理',
    credit: 3,
    hours: 48,
    teacherId: 'u-teacher-6',
    department: '人工智能教研室',
    status: 'draft',
    description: '讲解词表示、序列建模、预训练语言模型与常见 NLP 任务。',
    objective: '掌握文本表示与序列建模方法，能够完成文本分类与信息抽取任务。',
    major: '人工智能',
    classes: [cls('智能 2101', '周五 7-8 节', '智能楼 301', 40)],
    resources: []
  }
]

function buildClasses(seed: CourseSeed): ClassInfo[] {
  return seed.classes.map((item, index) => ({
    id: `${seed.id}-class-${index + 1}`,
    className: item.className,
    schedule: item.schedule,
    location: item.location,
    studentCount: item.studentCount
  }))
}

function buildResources(seed: CourseSeed): Resource[] {
  return seed.resources.map((item, index) => ({
    id: `${seed.id}-res-${index + 1}`,
    courseId: seed.id,
    name: item.name,
    type: item.type,
    size: item.size,
    uploader: item.uploader,
    uploadedAt: item.uploadedAt
  }))
}

export const courses: Course[] = COURSE_SEEDS.map((seed) => {
  const classes = buildClasses(seed)
  const resources = buildResources(seed)
  const teacher = findUserById(seed.teacherId)
  return {
    id: seed.id,
    code: seed.code,
    name: seed.name,
    credit: seed.credit,
    hours: seed.hours,
    semester: seed.semester ?? CURRENT_SEMESTER,
    department: seed.department,
    teacherId: seed.teacherId,
    teacherName: teacher?.name ?? '未知教师',
    description: seed.description,
    objective: seed.objective,
    major: seed.major,
    classes,
    studentCount: classes.reduce((sum, item) => sum + item.studentCount, 0),
    resourceCount: resources.length,
    status: seed.status
  }
})

export const resources: Resource[] = COURSE_SEEDS.flatMap(buildResources)

function scopedCourses(user: MockContextUser | null): Course[] {
  if (!user) return []
  if (user.role === 'director') {
    return courses.filter((course) => course.department === user.department)
  }
  if (user.role === 'teacher') {
    return courses.filter((course) => course.teacherId === user.id)
  }
  return [...courses]
}

type MockContextUser = { id: string; role: string; department?: string }

function recomputeResourceCount(courseId: string): void {
  const course = courses.find((item) => item.id === courseId)
  if (course) course.resourceCount = resources.filter((item) => item.courseId === courseId).length
}

function paginate<T>(list: T[], page: number, pageSize: number): PageResult<T> {
  const start = (page - 1) * pageSize
  return {
    list: list.slice(start, start + pageSize),
    total: list.length,
    page,
    pageSize
  }
}

function toCourseForm(body: Record<string, unknown>): Partial<Course> {
  const teacherId = String(body.teacherId ?? '')
  const teacher = findUserById(teacherId)
  return {
    code: String(body.code ?? ''),
    name: String(body.name ?? ''),
    credit: Number(body.credit ?? 0),
    hours: Number(body.hours ?? 0),
    semester: String(body.semester ?? CURRENT_SEMESTER),
    department: String(body.department ?? ''),
    teacherId,
    teacherName: teacher?.name ?? '',
    description: String(body.description ?? ''),
    status: (body.status as CourseStatus) ?? 'draft'
  }
}

export const courseRoutes: MockRoute[] = [
  route('get', '/courses', (ctx): ApiResponse<unknown> => {
    const query = ctx.query
    let list = scopedCourses(ctx.user)
    if (query.semester) list = list.filter((item) => item.semester === query.semester)
    if (query.department) list = list.filter((item) => item.department === query.department)
    if (query.teacherId) list = list.filter((item) => item.teacherId === query.teacherId)
    if (query.status) list = list.filter((item) => item.status === query.status)
    if (query.keyword) {
      const keyword = query.keyword.toLowerCase()
      list = list.filter(
        (item) =>
          item.name.toLowerCase().includes(keyword) || item.code.toLowerCase().includes(keyword)
      )
    }
    const page = Number(query.page ?? 1) || 1
    const pageSize = Number(query.pageSize ?? 10) || 10
    return { code: 0, message: 'ok', data: paginate(list, page, pageSize) }
  }),

  route('get', '/courses/:id', (ctx): ApiResponse<unknown> => {
    const course = courses.find((item) => item.id === ctx.params.id)
    if (!course) return { code: 1, message: '课程不存在', data: null }
    return { code: 0, message: 'ok', data: course }
  }),

  route('post', '/courses', (ctx): ApiResponse<unknown> => {
    if (!ctx.user || ctx.user.role !== 'director') {
      return { code: 403, message: '无权新增课程', data: null }
    }
    const form = toCourseForm(ctx.body)
    const id = `c-${Date.now()}`
    const course: Course = {
      id,
      code: form.code ?? '',
      name: form.name ?? '',
      credit: form.credit ?? 0,
      hours: form.hours ?? 0,
      semester: form.semester ?? CURRENT_SEMESTER,
      department: form.department ?? '',
      teacherId: form.teacherId ?? '',
      teacherName: form.teacherName ?? '',
      description: form.description ?? '',
      classes: [],
      studentCount: 0,
      resourceCount: 0,
      status: (form.status as CourseStatus) ?? 'draft'
    }
    courses.unshift(course)
    return { code: 0, message: 'ok', data: course }
  }),

  route('put', '/courses/:id', (ctx): ApiResponse<unknown> => {
    if (!ctx.user || ctx.user.role !== 'director') {
      return { code: 403, message: '无权修改课程', data: null }
    }
    const course = courses.find((item) => item.id === ctx.params.id)
    if (!course) return { code: 1, message: '课程不存在', data: null }
    Object.assign(course, toCourseForm(ctx.body))
    return { code: 0, message: 'ok', data: course }
  }),

  route('get', '/courses/:id/resources', (ctx): ApiResponse<unknown> => {
    const list = resources.filter((item) => item.courseId === ctx.params.id)
    return { code: 0, message: 'ok', data: list }
  }),

  route('post', '/courses/:id/resources', async (ctx): Promise<ApiResponse<unknown>> => {
    if (!ctx.user || ctx.user.role !== 'teacher') {
      return { code: 403, message: '无权上传资源', data: null }
    }
    const course = courses.find((item) => item.id === ctx.params.id)
    if (!course) return { code: 1, message: '课程不存在', data: null }
    const file = ctx.formData?.get('file')
    if (!(file instanceof File)) {
      return { code: 1, message: '未接收到文件', data: null }
    }
    const resource: Resource = {
      id: `res-${Date.now()}`,
      courseId: course.id,
      name: file.name,
      type: resolveResourceType(file.name),
      size: file.size,
      uploader: ctx.user.name,
      uploadedAt: new Date().toISOString()
    }
    resources.unshift(resource)
    recomputeResourceCount(course.id)
    return { code: 0, message: 'ok', data: resource }
  }),

  route('delete', '/resources/:id', (ctx): ApiResponse<unknown> => {
    if (!ctx.user || ctx.user.role !== 'teacher') {
      return { code: 403, message: '无权删除资源', data: null }
    }
    const index = resources.findIndex((item) => item.id === ctx.params.id)
    if (index < 0) return { code: 1, message: '资源不存在', data: null }
    const [removed] = resources.splice(index, 1)
    recomputeResourceCount(removed.courseId)
    return { code: 0, message: 'ok', data: null }
  })
]

export { CURRENT_SEMESTER, paginate, scopedCourses }
