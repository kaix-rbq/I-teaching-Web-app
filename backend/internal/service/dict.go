package service

import (
	"context"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// DictService 提供前端筛选下拉所需的字典数据。
type DictService interface {
	Departments(ctx context.Context) ([]dto.DeptOption, error)
	Teachers(ctx context.Context, departmentID uint64) ([]dto.TeacherOption, error)
	// MyTimetable 返回课表下拉行（B11）：teacher 只能看本人，supervisor / director
	// 可按 teacherID 过滤；semester 为空时默认当前学期。
	MyTimetable(ctx context.Context, role string, deptID, userID, teacherID uint64, semester string) ([]dto.TimetableItem, error)
	// CreateTimetableEntry / DeleteTimetableEntry 由 director / supervisor 手动维护课表。
	CreateTimetableEntry(ctx context.Context, req dto.TimetableCreateReq) (*dto.TimetableItem, error)
	DeleteTimetableEntry(ctx context.Context, id uint64) error
}

type dictService struct {
	users   repository.UserRepository
	courses repository.CourseRepository
	tables  repository.TimetableRepository
}

// NewDictService 构造字典服务。
func NewDictService(users repository.UserRepository, courses repository.CourseRepository, tables repository.TimetableRepository) DictService {
	return &dictService{users: users, courses: courses, tables: tables}
}

// Departments 返回教研室列表。
func (s *dictService) Departments(ctx context.Context) ([]dto.DeptOption, error) {
	depts, err := s.users.ListDepartments(ctx)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询教研室列表失败", err)
	}
	out := make([]dto.DeptOption, 0, len(depts))
	for _, d := range depts {
		out = append(out, dto.DeptOption{ID: d.ID, Name: d.Name})
	}
	return out, nil
}

// Teachers 返回教师列表，可按教研室过滤。
func (s *dictService) Teachers(ctx context.Context, departmentID uint64) ([]dto.TeacherOption, error) {
	users, err := s.users.ListTeachers(ctx, departmentID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询教师列表失败", err)
	}
	out := make([]dto.TeacherOption, 0, len(users))
	for _, u := range users {
		out = append(out, dto.TeacherOption{ID: u.ID, Name: u.Name, DepartmentID: u.DeptID()})
	}
	return out, nil
}

// MyTimetable 返回课表下拉行：teacher 强制只能看本人（越权按本人处理，不泄露他人课表）。
func (s *dictService) MyTimetable(ctx context.Context, role string, _, userID, teacherID uint64, semester string) ([]dto.TimetableItem, error) {
	if role == RoleTeacher {
		teacherID = userID
	}
	if semester == "" {
		semester = CurrentSemester
	}
	rows, err := s.tables.List(ctx, repository.TimetableFilter{TeacherID: teacherID, Semester: semester})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询课表失败", err)
	}
	out := make([]dto.TimetableItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.TimetableItem{
			ID: r.ID, CourseID: r.CourseID, CourseCode: r.CourseCode, CourseName: r.CourseName,
			ClassID: r.ClassID, ClassName: r.ClassName, TeacherID: r.TeacherID,
			TeacherName: r.TeacherName, Semester: r.Semester,
		})
	}
	return out, nil
}

// CreateTimetableEntry 手动新增一条课表（B11）：校验课程存在、班级归属该课程、教师存在。
func (s *dictService) CreateTimetableEntry(ctx context.Context, req dto.TimetableCreateReq) (*dto.TimetableItem, error) {
	course, err := s.courses.GetDetail(ctx, req.CourseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "课程不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课程失败", err)
	}
	classes, err := s.courses.ListClasses(ctx, req.CourseID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询开课班级失败", err)
	}
	found := false
	for _, c := range classes {
		if c.ID == req.ClassID {
			found = true
			break
		}
	}
	if !found {
		return nil, errcode.New(errcode.BizRule, "班级不归属该课程")
	}
	teacherID := req.TeacherID
	if teacherID == 0 {
		teacherID = course.TeacherID
	} else {
		t, terr := s.users.GetByID(ctx, teacherID)
		if terr != nil {
			if repository.IsNotFound(terr) {
				return nil, errcode.New(errcode.NotFound, "教师不存在")
			}
			return nil, errcode.Wrap(errcode.Internal, "查询教师失败", terr)
		}
		if t.Role != model.RoleTeacher {
			return nil, errcode.New(errcode.BizRule, "课表只能关联教师账号")
		}
	}
	semester := req.Semester
	if semester == "" {
		semester = CurrentSemester
	}
	dup, err := s.tables.ExistsDuplicate(ctx, req.CourseID, req.ClassID, teacherID, semester)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "校验课表唯一性失败", err)
	}
	if dup {
		return nil, errcode.New(errcode.Conflict, "该课表条目已存在")
	}
	t := &model.CourseTimetable{CourseID: req.CourseID, ClassID: req.ClassID, TeacherID: teacherID, Semester: semester}
	if err := s.tables.Create(ctx, t); err != nil {
		if repository.IsDuplicate(err) {
			return nil, errcode.New(errcode.Conflict, "该课表条目已存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "创建课表失败", err)
	}
	items, err := s.MyTimetable(ctx, RoleSupervisor, 0, 0, teacherID, semester)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.CourseID == req.CourseID && it.ClassID == req.ClassID {
			return &it, nil
		}
	}
	return nil, errcode.Wrap(errcode.Internal, "回读课表失败", errcode.New(errcode.Internal, "课表写入后未找到"))
}

// DeleteTimetableEntry 删除一条课表（不影响已有授课记录）。
func (s *dictService) DeleteTimetableEntry(ctx context.Context, id uint64) error {
	if err := s.tables.Delete(ctx, id); err != nil {
		return errcode.Wrap(errcode.Internal, "删除课表失败", err)
	}
	return nil
}
