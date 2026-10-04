package service

import (
	"context"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// DashboardService 按角色返回工作台聚合数据（一套接口三种 DTO）。
type DashboardService interface {
	Board(ctx context.Context, role string, deptID, userID uint64) (any, error)
}

type dashboardService struct {
	courses     repository.CourseRepository
	users       repository.UserRepository
	resources   repository.ResourceRepository
	supervision repository.SupervisionRepository
	drafts      repository.DraftRepository
	sessions    repository.SessionRepository
}

// NewDashboardService 构造工作台服务。
func NewDashboardService(
	courses repository.CourseRepository,
	users repository.UserRepository,
	resources repository.ResourceRepository,
	supervision repository.SupervisionRepository,
	drafts repository.DraftRepository,
	sessions repository.SessionRepository,
) DashboardService {
	return &dashboardService{
		courses: courses, users: users, resources: resources,
		supervision: supervision, drafts: drafts, sessions: sessions,
	}
}

// Board 依据角色分派到主任 / 教师 / 督导三种聚合。
func (s *dashboardService) Board(ctx context.Context, role string, deptID, userID uint64) (any, error) {
	switch role {
	case model.RoleDirector:
		return s.directorBoard(ctx, deptID)
	case model.RoleTeacher:
		return s.teacherBoard(ctx, userID)
	case model.RoleSupervisor:
		return s.supervisorBoard(ctx, userID)
	default:
		return nil, errcode.New(errcode.ForbiddenRole, "未知角色，无法加载工作台")
	}
}

func (s *dashboardService) directorBoard(ctx context.Context, deptID uint64) (*dto.DirectorDashboard, error) {
	filter := courseFilterFor(Scope{DepartmentID: deptID})

	courseCount, err := s.courses.Count(ctx, filter)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计本室课程数失败", err)
	}
	teacherCount, err := s.users.CountByDepartment(ctx, deptID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计本室教师数失败", err)
	}
	classCount, err := s.courses.CountClasses(ctx, repository.CourseFilter{
		Semester:          CurrentSemester,
		ScopeDepartmentID: deptID,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计开课班次失败", err)
	}
	resourceCount, err := s.courses.CountResources(ctx, filter)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计课程资源失败", err)
	}
	rows, _, err := s.courses.List(ctx, repository.CourseListParams{
		CourseFilter: filter,
		Page:         1,
		PageSize:     10,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询近期开课失败", err)
	}

	return &dto.DirectorDashboard{
		CourseCount:   int(courseCount),
		TeacherCount:  int(teacherCount),
		ClassCount:    int(classCount),
		ResourceCount: int(resourceCount),
		RecentCourses: toCourseListItems(rows),
	}, nil
}

func (s *dashboardService) teacherBoard(ctx context.Context, userID uint64) (*dto.TeacherDashboard, error) {
	filter := courseFilterFor(Scope{TeacherID: userID})

	courseCount, err := s.courses.Count(ctx, filter)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计我的课程数失败", err)
	}
	classCount, err := s.courses.CountClasses(ctx, filter)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计授课班级数失败", err)
	}
	studentCount, err := s.courses.SumStudents(ctx, filter)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计学生人次失败", err)
	}
	resourceCount, err := s.courses.CountResources(ctx, filter)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计资源数失败", err)
	}
	rows, _, err := s.courses.List(ctx, repository.CourseListParams{
		CourseFilter: filter,
		Page:         1,
		PageSize:     50,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询我的课程失败", err)
	}
	recent, err := s.resources.ListRecentByTeacher(ctx, userID, 5)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询近期上传资源失败", err)
	}

	return &dto.TeacherDashboard{
		CourseCount:     int(courseCount),
		ClassCount:      int(classCount),
		StudentCount:    int(studentCount),
		ResourceCount:   int(resourceCount),
		MyCourses:       toCourseListItems(rows),
		RecentResources: toResourceDTOs(recent),
	}, nil
}

// supervisorBoard 只聚合并返回督导核心任务所需数据：
//   - 全部听评课安排（前端按今日 / 本周 / 本月分档展示待评课队列）；
//   - 最近创建的三份草稿（工作台「草稿箱」区块）。
//
// 已移除课程数 / 覆盖率等与「记录课程并评估」无关的统计。
func (s *dashboardService) supervisorBoard(ctx context.Context, userID uint64) (*dto.SupervisorDashboard, error) {
	rows, _, err := s.supervision.ListPlans(ctx, repository.PlanListParams{
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询听评课安排失败", err)
	}

	draftRows, err := s.drafts.RecentBySupervisor(ctx, userID, 3)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询最近草稿失败", err)
	}
	drafts := make([]dto.DraftDTO, 0, len(draftRows))
	for _, row := range draftRows {
		drafts = append(drafts, toDraftDTOFromRow(row))
	}

	pendingRows, err := s.sessions.ListPending(ctx, 20)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询待评估授课记录失败", err)
	}

	return &dto.SupervisorDashboard{
		RecentPlans:     toPlanItems(rows),
		RecentDrafts:    drafts,
		PendingSessions: toPendingSessionItems(pendingRows),
	}, nil
}
