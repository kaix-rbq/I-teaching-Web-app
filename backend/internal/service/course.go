package service

import (
	"context"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// CourseService 负责课程查询与主任维护课程的业务规则。
type CourseService interface {
	List(ctx context.Context, role string, deptID, userID uint64, q dto.CourseListQuery) (*dto.PageResult[dto.CourseListItem], error)
	Detail(ctx context.Context, role string, deptID, userID, courseID uint64) (*dto.CourseDetail, error)
	Create(ctx context.Context, deptID uint64, req dto.CourseUpsertReq) (*dto.CourseDetail, error)
	Update(ctx context.Context, deptID, courseID uint64, req dto.CourseUpsertReq) (*dto.CourseDetail, error)
}

type courseService struct {
	courses repository.CourseRepository
	users   repository.UserRepository
}

// NewCourseService 构造课程服务。
func NewCourseService(courses repository.CourseRepository, users repository.UserRepository) CourseService {
	return &courseService{courses: courses, users: users}
}

// List 按角色裁剪数据范围，并叠加前端筛选参数。
func (s *courseService) List(
	ctx context.Context, role string, deptID, userID uint64, q dto.CourseListQuery,
) (*dto.PageResult[dto.CourseListItem], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	scope := ScopeFor(role, deptID, userID)

	rows, total, err := s.courses.List(ctx, repository.CourseListParams{
		CourseFilter: repository.CourseFilter{
			Semester:          q.Semester,
			DepartmentID:      q.DepartmentID,
			TeacherID:         q.TeacherID,
			Status:            q.Status,
			Keyword:           q.Keyword,
			ScopeDepartmentID: scope.DepartmentID,
			ScopeTeacherID:    scope.TeacherID,
		},
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询课程列表失败", err)
	}
	return dto.NewPageResult(toCourseListItems(rows), total, page, pageSize), nil
}

// Detail 返回课程详情（含班级），并校验课程落在当前角色的数据范围内。
func (s *courseService) Detail(
	ctx context.Context, role string, deptID, userID, courseID uint64,
) (*dto.CourseDetail, error) {
	row, err := s.courses.GetDetail(ctx, courseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "课程不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课程详情失败", err)
	}
	if err := checkCourseScope(ScopeFor(role, deptID, userID), row); err != nil {
		return nil, err
	}

	classes, err := s.courses.ListClasses(ctx, courseID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询开课班级失败", err)
	}
	return &dto.CourseDetail{
		CourseListItem: toCourseListItem(*row),
		Classes:        toClassInfoDTOs(classes),
	}, nil
}

// Create 新增课程：主任只能在本室开课，授课教师必须属于该教研室且角色为 teacher。
func (s *courseService) Create(ctx context.Context, deptID uint64, req dto.CourseUpsertReq) (*dto.CourseDetail, error) {
	if err := s.checkDirectorOwnDept(ctx, deptID, req.DepartmentID); err != nil {
		return nil, err
	}
	if err := s.checkTeacherInDepartment(ctx, req.TeacherID, req.DepartmentID); err != nil {
		return nil, err
	}

	exists, err := s.courses.ExistsCodeSemester(ctx, req.Code, req.Semester, 0)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "校验课程编码失败", err)
	}
	if exists {
		return nil, errcode.New(errcode.Conflict, "该学期已存在相同课程编码")
	}

	course := &model.Course{
		Code:         req.Code,
		Name:         req.Name,
		Credit:       req.Credit,
		Hours:        req.Hours,
		Semester:     req.Semester,
		DepartmentID: req.DepartmentID,
		TeacherID:    req.TeacherID,
		Description:  req.Description,
		Status:       req.Status,
	}
	if err := s.courses.Create(ctx, course); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "新增课程失败", err)
	}
	return s.Detail(ctx, RoleSupervisor, 0, 0, course.ID)
}

// Update 修改课程：课程必须属于主任本室；code 不可修改。
func (s *courseService) Update(
	ctx context.Context, deptID, courseID uint64, req dto.CourseUpsertReq,
) (*dto.CourseDetail, error) {
	row, err := s.courses.GetDetail(ctx, courseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "课程不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课程失败", err)
	}
	if row.DepartmentID != deptID {
		return nil, errcode.New(errcode.ForbiddenData, "无权修改其他教研室的课程")
	}
	if err := s.checkDirectorOwnDept(ctx, deptID, req.DepartmentID); err != nil {
		return nil, err
	}
	if err := s.checkTeacherInDepartment(ctx, req.TeacherID, req.DepartmentID); err != nil {
		return nil, err
	}

	// code 不可改；若学期变更，仍需保证「编码 + 学期」唯一。
	exists, err := s.courses.ExistsCodeSemester(ctx, row.Code, req.Semester, courseID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "校验课程编码失败", err)
	}
	if exists {
		return nil, errcode.New(errcode.Conflict, "该学期已存在相同课程编码")
	}

	fields := map[string]any{
		"name":          req.Name,
		"credit":        req.Credit,
		"hours":         req.Hours,
		"semester":      req.Semester,
		"department_id": req.DepartmentID,
		"teacher_id":    req.TeacherID,
		"description":   req.Description,
		"status":        req.Status,
	}
	if err := s.courses.Update(ctx, courseID, fields); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "修改课程失败", err)
	}
	return s.Detail(ctx, RoleSupervisor, 0, 0, courseID)
}

// checkDirectorOwnDept 校验目标教研室等于主任本室。
func (s *courseService) checkDirectorOwnDept(ctx context.Context, directorDeptID, targetDeptID uint64) error {
	if directorDeptID == 0 || directorDeptID != targetDeptID {
		return errcode.New(errcode.ForbiddenData, "只能维护本教研室的课程")
	}
	return nil
}

// checkTeacherInDepartment 校验教师存在、角色为 teacher 且属于指定教研室。
func (s *courseService) checkTeacherInDepartment(ctx context.Context, teacherID, departmentID uint64) error {
	teacher, err := s.users.GetByID(ctx, teacherID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errcode.New(errcode.ForbiddenData, "授课教师不存在")
		}
		return errcode.Wrap(errcode.Internal, "查询教师失败", err)
	}
	if teacher.Role != model.RoleTeacher || teacher.DeptID() != departmentID {
		return errcode.New(errcode.ForbiddenData, "授课教师不属于所选教研室")
	}
	return nil
}

// checkCourseScope 校验课程是否落在给定数据范围内。
func checkCourseScope(scope Scope, row *repository.CourseRow) error {
	if scope.DepartmentID > 0 && row.DepartmentID != scope.DepartmentID {
		return errcode.New(errcode.ForbiddenData, "无权查看其他教研室的课程")
	}
	if scope.TeacherID > 0 && row.TeacherID != scope.TeacherID {
		return errcode.New(errcode.ForbiddenData, "无权查看他人课程")
	}
	return nil
}

// courseFilterFor 由数据范围构造课程过滤条件（供工作台聚合复用）。
func courseFilterFor(scope Scope) repository.CourseFilter {
	return repository.CourseFilter{
		ScopeDepartmentID: scope.DepartmentID,
		ScopeTeacherID:    scope.TeacherID,
	}
}
