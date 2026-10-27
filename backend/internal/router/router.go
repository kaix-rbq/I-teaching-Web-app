// Package router 是唯一的路由注册地：组装依赖、注册分组、挂载中间件。
package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"aijiaoxue-api/internal/agent"
	"aijiaoxue-api/internal/asr"
	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/handler"
	"aijiaoxue-api/internal/kb"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/jwtutil"
)

// New 组装 repository → service → handler 并返回唯一的 gin 引擎。
//
// 中间件生效顺序：Recovery → Logger → CORS → [Auth → RBAC]（按分组挂载）。
//
// 同时返回 recordingSvc：进程重启时需要由 main 驱动一次 RequeueStuck
// （重新入队中断的转写任务）。路由层不承担生命周期副作用，故不在此处自行启动。
func New(db *gorm.DB, cfg *config.Config, jwt *jwtutil.Manager) (*gin.Engine, service.RecordingService, service.AiTaskService) {
	dto.RegisterValidators()

	userRepo := repository.NewUserRepository(db)
	courseRepo := repository.NewCourseRepository(db)
	resourceRepo := repository.NewResourceRepository(db)
	supervisionRepo := repository.NewSupervisionRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	evaluationRepo := repository.NewEvaluationRepository(db)
	recordingRepo := repository.NewRecordingRepository(db)
	draftRepo := repository.NewDraftRepository(db)
	aiTaskRepo := repository.NewAiTaskRepository(db)
	improvementRepo := repository.NewImprovementRepository(db)
	reflectionRepo := repository.NewReflectionRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	demoRepo := repository.NewDemoRepository(db)
	timetableRepo := repository.NewTimetableRepository(db)
	scheduleEventRepo := repository.NewScheduleEventRepository(db)
	classroomIssueRepo := repository.NewClassroomIssueRepository(db)

	authSvc := service.NewAuthService(userRepo, jwt)
	courseSvc := service.NewCourseService(courseRepo, userRepo)
	resourceSvc := service.NewResourceService(resourceRepo, courseRepo, cfg.Upload)
	supervisionSvc := service.NewSupervisionService(supervisionRepo)
	dashboardSvc := service.NewDashboardService(courseRepo, userRepo, resourceRepo, supervisionRepo, draftRepo, sessionRepo)
	dictSvc := service.NewDictService(userRepo, courseRepo, timetableRepo)
	notificationSvc := service.NewNotificationService(notificationRepo, sessionRepo)
	sessionSvc := service.NewSessionServiceWithDelete(sessionRepo, evaluationRepo, courseRepo, userRepo, recordingRepo, improvementRepo, cfg.Evaluation, notificationSvc)
	improvementSvc := service.NewImprovementService(improvementRepo)
	reflectionSvc := service.NewReflectionService(reflectionRepo)
	supervisionTaskSvc := service.NewSupervisionTaskService(evaluationRepo, cfg.Evaluation)
	demoSvc := service.NewDemoService(demoRepo, cfg.Demo.Enabled)
	scheduleEventSvc := service.NewScheduleEventService(scheduleEventRepo)
	classroomIssueSvc := service.NewClassroomIssueService(classroomIssueRepo)
	teacherScoreSvc := service.NewTeacherScoreService(userRepo, courseRepo, evaluationRepo, cfg.Evaluation, improvementRepo)
	// ⚠️ 不能写 Transcriber(asr.NewClient(...))：把 nil 的 *asr.Client 装进接口会得到
	// 非 nil 接口（holds-nil-pointer），service 里的 `s.transcriber == nil` 判断会失效，
	// 调用时直接空指针 panic。必须显式判空后再赋值。
	var transcriber service.Transcriber
	if client := asr.NewClient(cfg.Transcription); client != nil {
		transcriber = client
	}
	// 同理：agent.NewClient 未配置时返回 nil *Client，直接装进接口同样会 holds-nil-pointer。
	var sessionAnalyzer agent.SessionAnalyzer
	if client := agent.NewClient(cfg.Agent); client != nil {
		sessionAnalyzer = client
	}
	// 知识库检索器与 embedding 客户端同生共死：未启用 embedding 时保持 nil，
	// S1 会无 KB 降级运行，且不会因检索失败把 ai_tasks 标 failed。
	var retriever agent.Retriever
	if emb := agent.NewEmbeddingClient(cfg.Agent, cfg.Embedding); emb != nil {
		retriever = agent.NewRetriever(emb, kb.NewStore(db), cfg.Embedding)
	}
	aiTaskSvc := service.NewAiTaskService(
		aiTaskRepo, sessionRepo, evaluationRepo, recordingRepo,
		sessionAnalyzer, retriever, cfg.Agent, cfg.Evaluation,
	)
	recordingSvc := service.NewRecordingService(
		recordingRepo, sessionRepo,
		cfg.Upload, cfg.Transcription,
		jwt, transcriber,
		aiTaskSvc.EnqueueSessionEval,
	)

	authH := handler.NewAuthHandler(authSvc)
	courseH := handler.NewCourseHandler(courseSvc)
	resourceH := handler.NewResourceHandler(resourceSvc)
	supervisionH := handler.NewSupervisionHandler(supervisionSvc)
	dashboardH := handler.NewDashboardHandler(dashboardSvc)
	dictH := handler.NewDictHandler(dictSvc)
	sessionH := handler.NewSessionHandler(sessionSvc, recordingSvc)
	teacherScoreH := handler.NewTeacherScoreHandler(teacherScoreSvc)
	recordingH := handler.NewRecordingHandler(recordingSvc)
	aiTaskH := handler.NewAiTaskHandler(aiTaskSvc)
	improvementH := handler.NewImprovementHandler(improvementSvc)
	reflectionH := handler.NewReflectionHandler(reflectionSvc)
	notificationH := handler.NewNotificationHandler(notificationSvc)
	supervisionTaskH := handler.NewSupervisionTaskHandler(supervisionTaskSvc)
	demoH := handler.NewDemoHandler(demoSvc)
	scheduleEventH := handler.NewScheduleEventHandler(scheduleEventSvc)
	classroomIssueH := handler.NewClassroomIssueHandler(classroomIssueSvc)

	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(middleware.Recovery(), middleware.Logger(), middleware.CORS(cfg.CORS.Origins, cfg.CORS.AllowAnyOrigin))

	// 运维探活（同时挂在根路径与 /api/v1 下，兼容验收脚本）
	r.GET("/healthz", handler.Health)

	v1 := r.Group("/api/v1")
	v1.GET("/healthz", handler.Health)
	v1.POST("/auth/login", authH.Login)

	// 🔴 音频流必须挂在 v1 上，不能放进下面的 authed 组：
	// <audio> 无法携带 Authorization 头，只能靠 ?ticket= 查询串鉴权。
	// 若放进 authed 组，组上的 Auth 会先执行并直接 401，票据中间件根本没机会运行。
	// 角色与数据范围校验仍由 RecordingService.Stream 在 service 层强制（教师 403）。
	v1.GET("/recordings/:id/stream", middleware.AuthOrPlaybackTicket(jwt), recordingH.Stream)

	authed := v1.Group("", middleware.Auth(jwt))
	authed.GET("/auth/me", authH.Me)
	authed.POST("/auth/logout", authH.Logout)
	authed.GET("/dashboard", dashboardH.Board)
	authed.GET("/courses", courseH.List)
	authed.GET("/courses/:id", courseH.Detail)
	authed.GET("/courses/:id/resources", resourceH.List)
	authed.GET("/resources/:id/download", resourceH.Download)
	authed.GET("/departments", dictH.Departments)
	authed.GET("/teachers", dictH.Teachers)
	// B11 最小课表下拉：建课时课程/班级下拉数据源（维护见下方 director/supervisor 组）。
	authed.GET("/timetable/mine", dictH.MyTimetable)

	// Sprint 2.1：评价闭环（数据裁剪在 service 层，铁律见后端 AGENTS.md §3）。
	authed.GET("/courses/:id/sessions", sessionH.ListByCourse)
	authed.GET("/courses/:id/evaluation-summary", teacherScoreH.CourseSummary)
	authed.GET("/sessions/:id", sessionH.Detail)
	authed.GET("/sessions/:id/evaluation", sessionH.Evaluation)
	authed.GET("/sessions/:id/transcript", recordingH.Transcript)
	authed.POST("/sessions/:id/transcript/retry", recordingH.Retry)
	authed.GET("/teachers/:id/evaluation-summary", teacherScoreH.TeacherSummary)
	authed.GET("/teachers/:id/evaluations", teacherScoreH.TeacherEvaluations)
	authed.GET("/teachers/:id/report", teacherScoreH.Report)

	// —— AI 观察（D3 / R0.2）：详情与入队；数据裁剪在 service 层（D8 教师侧风险脱敏）——
	authed.GET("/sessions/:id/ai-observation", aiTaskH.Observation)
	authed.POST("/sessions/:id/ai-observation/analyze", aiTaskH.Analyze)
	authed.GET("/ai-tasks/:id", aiTaskH.TaskStatus)

	// —— 提优闭环（2026-10-16 重构 N4/N5/B12）：清单 / 反思 / 站内信，角色与归属在 service 层判定 ——
	authed.GET("/improvement-items", improvementH.List)
	authed.POST("/improvement-items", improvementH.Create)
	authed.PUT("/improvement-items/:id", improvementH.Update)
	authed.PUT("/improvement-items/:id/complete", improvementH.Complete)
	authed.DELETE("/improvement-items/:id", improvementH.Delete)
	authed.GET("/teaching-reflections", reflectionH.List)
	authed.POST("/teaching-reflections", reflectionH.Create)
	authed.PUT("/teaching-reflections/:id", reflectionH.Update)
	authed.GET("/notifications", notificationH.List)
	// 课堂问题：教师看/确认本人，督导与主任看全部并处理（service 层裁剪）。
	authed.GET("/classroom-issues", classroomIssueH.List)
	// 🔴 注册顺序：静态段 /read-all 必须先于参数段 /:id/read。
	authed.PUT("/notifications/read-all", notificationH.MarkAllRead)
	authed.PUT("/notifications/:id/read", notificationH.MarkRead)

	authed.Group("", middleware.RequireRoles(service.RoleDirector)).
		POST("/courses", courseH.Create).
		PUT("/courses/:id", courseH.Update).
		// 主任事务管理（周视图）：仅本人，service 层再次校验 owner。
		GET("/schedule-events", scheduleEventH.List).
		POST("/schedule-events", scheduleEventH.Create).
		PATCH("/schedule-events/:id", scheduleEventH.Update).
		DELETE("/schedule-events/:id", scheduleEventH.Delete)

	authed.Group("", middleware.RequireRoles(service.RoleTeacher)).
		POST("/courses/:id/resources", resourceH.Upload).
		DELETE("/resources/:id", resourceH.Delete).
		PATCH("/timetable/:id/remark", dictH.UpdateTimetableRemark).
		POST("/classroom-issues", classroomIssueH.Create).
		POST("/classroom-issues/:id/confirm", classroomIssueH.Confirm)

	// —— 授课与录音（B1/D6）：teacher 本人课程 / supervisor 代录；建课即触发后续 AI 评析 ——
	authed.Group("", middleware.RequireRoles(service.RoleTeacher, service.RoleSupervisor)).
		POST("/sessions", sessionH.Create).
		DELETE("/sessions/:id", sessionH.Delete).
		POST("/sessions/:id/recording", recordingH.Upload).
		DELETE("/sessions/:id/recording", recordingH.Delete)

	authed.Group("", middleware.RequireRoles(service.RoleSupervisor)).
		GET("/supervision/coverage", supervisionH.Coverage).
		GET("/supervision/plans", supervisionH.Plans).
		GET("/supervision/comment-tags", supervisionH.CommentTags).
		GET("/supervision/ai-tasks", supervisionTaskH.Queue).
		PUT("/sessions/:id/supervisor-observation", sessionH.SubmitObservation).
		// 2026-10-16 重构 §4.5：五维评分与草稿箱整套停用，路由暂留并返回 50002。
		PUT("/sessions/:id/supervisor-evaluation", handler.Deprecated("督导五维评分已废弃，请改用督导观察接口")).
		POST("/ai-tasks/:id/retry", aiTaskH.Retry).
		GET("/sessions/:id/draft", handler.Deprecated("评估草稿已停用")).
		PUT("/sessions/:id/draft", handler.Deprecated("评估草稿已停用")).
		GET("/drafts", handler.Deprecated("评估草稿已停用")).
		DELETE("/drafts/:id", handler.Deprecated("评估草稿已停用")).
		POST("/drafts/:id/submit", handler.Deprecated("评估草稿已停用"))

	authed.Group("", middleware.RequireRoles(service.RoleDirector, service.RoleSupervisor)).
		GET("/teacher-scores", teacherScoreH.List).
		POST("/timetable", dictH.CreateTimetable).
		DELETE("/timetable/:id", dictH.DeleteTimetable).
		POST("/classroom-issues/:id/handle", classroomIssueH.Handle).
		POST("/demo/seed", demoH.Seed)

	return r, recordingSvc, aiTaskSvc
}
