// Package router 是唯一的路由注册地：组装依赖、注册分组、挂载中间件。
package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/handler"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/jwtutil"
)

// New 组装 repository → service → handler 并返回唯一的 gin 引擎。
//
// 中间件生效顺序：Recovery → Logger → CORS → [Auth → RBAC]（按分组挂载）。
func New(db *gorm.DB, cfg *config.Config, jwt *jwtutil.Manager) *gin.Engine {
	dto.RegisterValidators()

	userRepo := repository.NewUserRepository(db)
	courseRepo := repository.NewCourseRepository(db)
	resourceRepo := repository.NewResourceRepository(db)
	supervisionRepo := repository.NewSupervisionRepository(db)

	authSvc := service.NewAuthService(userRepo, jwt)
	courseSvc := service.NewCourseService(courseRepo, userRepo)
	resourceSvc := service.NewResourceService(resourceRepo, courseRepo, cfg.Upload)
	supervisionSvc := service.NewSupervisionService(supervisionRepo)
	dashboardSvc := service.NewDashboardService(courseRepo, userRepo, resourceRepo, supervisionRepo)
	dictSvc := service.NewDictService(userRepo)

	authH := handler.NewAuthHandler(authSvc)
	courseH := handler.NewCourseHandler(courseSvc)
	resourceH := handler.NewResourceHandler(resourceSvc)
	supervisionH := handler.NewSupervisionHandler(supervisionSvc)
	dashboardH := handler.NewDashboardHandler(dashboardSvc)
	dictH := handler.NewDictHandler(dictSvc)

	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(middleware.Recovery(), middleware.Logger(), middleware.CORS(cfg.CORS.Origins))

	// 运维探活（同时挂在根路径与 /api/v1 下，兼容验收脚本）
	r.GET("/healthz", handler.Health)

	v1 := r.Group("/api/v1")
	v1.GET("/healthz", handler.Health)
	v1.POST("/auth/login", authH.Login)

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

	authed.Group("", middleware.RequireRoles(service.RoleDirector)).
		POST("/courses", courseH.Create).
		PUT("/courses/:id", courseH.Update)

	authed.Group("", middleware.RequireRoles(service.RoleTeacher)).
		POST("/courses/:id/resources", resourceH.Upload).
		DELETE("/resources/:id", resourceH.Delete)

	authed.Group("", middleware.RequireRoles(service.RoleSupervisor)).
		GET("/supervision/coverage", supervisionH.Coverage).
		GET("/supervision/plans", supervisionH.Plans)

	return r
}
