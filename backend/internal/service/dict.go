package service

import (
	"context"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// DictService 提供前端筛选下拉所需的字典数据。
type DictService interface {
	Departments(ctx context.Context) ([]dto.DeptOption, error)
	Teachers(ctx context.Context, departmentID uint64) ([]dto.TeacherOption, error)
}

type dictService struct {
	users repository.UserRepository
}

// NewDictService 构造字典服务。
func NewDictService(users repository.UserRepository) DictService {
	return &dictService{users: users}
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
