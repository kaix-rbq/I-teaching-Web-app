package service

import (
	"context"

	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
)

// 以下 fake 以函数字段实现各 repository 接口，便于表驱动单测注入行为。

type fakeUserRepo struct {
	getByUsername  func(ctx context.Context, username string) (*model.User, error)
	getByID        func(ctx context.Context, id uint64) (*model.User, error)
	listTeachers   func(ctx context.Context, departmentID uint64) ([]model.User, error)
	listDepts      func(ctx context.Context) ([]model.Department, error)
	deptName       func(ctx context.Context, id uint64) (string, error)
	countByDept    func(ctx context.Context, departmentID uint64) (int64, error)
	updatePassHash func(ctx context.Context, usernames []string, hash string) (int64, error)
}

func (f *fakeUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	if f.getByUsername == nil {
		return nil, repository.ErrNotImplemented
	}
	return f.getByUsername(ctx, username)
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	if f.getByID == nil {
		return nil, repository.ErrNotImplemented
	}
	return f.getByID(ctx, id)
}

func (f *fakeUserRepo) ListTeachers(ctx context.Context, departmentID uint64) ([]model.User, error) {
	if f.listTeachers == nil {
		return nil, nil
	}
	return f.listTeachers(ctx, departmentID)
}

func (f *fakeUserRepo) ListDepartments(ctx context.Context) ([]model.Department, error) {
	if f.listDepts == nil {
		return nil, nil
	}
	return f.listDepts(ctx)
}

func (f *fakeUserRepo) GetDepartmentName(ctx context.Context, id uint64) (string, error) {
	if f.deptName == nil {
		return "", nil
	}
	return f.deptName(ctx, id)
}

func (f *fakeUserRepo) CountByDepartment(ctx context.Context, departmentID uint64) (int64, error) {
	if f.countByDept == nil {
		return 0, nil
	}
	return f.countByDept(ctx, departmentID)
}

func (f *fakeUserRepo) UpdatePasswordHash(ctx context.Context, usernames []string, hash string) (int64, error) {
	if f.updatePassHash == nil {
		return 0, nil
	}
	return f.updatePassHash(ctx, usernames, hash)
}

type fakeCourseRepo struct {
	list              func(ctx context.Context, p repository.CourseListParams) ([]repository.CourseRow, int64, error)
	getDetail         func(ctx context.Context, id uint64) (*repository.CourseRow, error)
	listClasses       func(ctx context.Context, courseID uint64) ([]model.ClassInfo, error)
	count             func(ctx context.Context, f repository.CourseFilter) (int64, error)
	countClasses      func(ctx context.Context, f repository.CourseFilter) (int64, error)
	sumStudents       func(ctx context.Context, f repository.CourseFilter) (int64, error)
	countResources    func(ctx context.Context, f repository.CourseFilter) (int64, error)
	existsCodeSem     func(ctx context.Context, code, semester string, excludeID uint64) (bool, error)
	create            func(ctx context.Context, c *model.Course) error
	update            func(ctx context.Context, id uint64, fields map[string]any) error
	lastListParams    *repository.CourseListParams
	lastExistsChecked bool
}

func (f *fakeCourseRepo) List(ctx context.Context, p repository.CourseListParams) ([]repository.CourseRow, int64, error) {
	f.lastListParams = &p
	if f.list == nil {
		return nil, 0, nil
	}
	return f.list(ctx, p)
}

func (f *fakeCourseRepo) GetDetail(ctx context.Context, id uint64) (*repository.CourseRow, error) {
	if f.getDetail == nil {
		return nil, repository.ErrNotImplemented
	}
	return f.getDetail(ctx, id)
}

func (f *fakeCourseRepo) ListClasses(ctx context.Context, courseID uint64) ([]model.ClassInfo, error) {
	if f.listClasses == nil {
		return nil, nil
	}
	return f.listClasses(ctx, courseID)
}

func (f *fakeCourseRepo) Count(ctx context.Context, flt repository.CourseFilter) (int64, error) {
	if f.count == nil {
		return 0, nil
	}
	return f.count(ctx, flt)
}

func (f *fakeCourseRepo) CountClasses(ctx context.Context, flt repository.CourseFilter) (int64, error) {
	if f.countClasses == nil {
		return 0, nil
	}
	return f.countClasses(ctx, flt)
}

func (f *fakeCourseRepo) SumStudents(ctx context.Context, flt repository.CourseFilter) (int64, error) {
	if f.sumStudents == nil {
		return 0, nil
	}
	return f.sumStudents(ctx, flt)
}

func (f *fakeCourseRepo) CountResources(ctx context.Context, flt repository.CourseFilter) (int64, error) {
	if f.countResources == nil {
		return 0, nil
	}
	return f.countResources(ctx, flt)
}

func (f *fakeCourseRepo) ExistsCodeSemester(ctx context.Context, code, semester string, excludeID uint64) (bool, error) {
	f.lastExistsChecked = true
	if f.existsCodeSem == nil {
		return false, nil
	}
	return f.existsCodeSem(ctx, code, semester, excludeID)
}

func (f *fakeCourseRepo) Create(ctx context.Context, c *model.Course) error {
	if f.create == nil {
		return nil
	}
	return f.create(ctx, c)
}

func (f *fakeCourseRepo) Update(ctx context.Context, id uint64, fields map[string]any) error {
	if f.update == nil {
		return nil
	}
	return f.update(ctx, id, fields)
}

type fakeResourceRepo struct {
	listByCourse  func(ctx context.Context, courseID uint64) ([]repository.ResourceRow, error)
	listRecent    func(ctx context.Context, teacherID uint64, limit int) ([]repository.ResourceRow, error)
	getByID       func(ctx context.Context, id uint64) (*model.Resource, error)
	create        func(ctx context.Context, r *model.Resource) error
	delete        func(ctx context.Context, id uint64) error
	deleteInvoked bool
}

func (f *fakeResourceRepo) ListByCourse(ctx context.Context, courseID uint64) ([]repository.ResourceRow, error) {
	if f.listByCourse == nil {
		return nil, nil
	}
	return f.listByCourse(ctx, courseID)
}

func (f *fakeResourceRepo) ListRecentByTeacher(ctx context.Context, teacherID uint64, limit int) ([]repository.ResourceRow, error) {
	if f.listRecent == nil {
		return nil, nil
	}
	return f.listRecent(ctx, teacherID, limit)
}

func (f *fakeResourceRepo) GetByID(ctx context.Context, id uint64) (*model.Resource, error) {
	if f.getByID == nil {
		return nil, repository.ErrNotImplemented
	}
	return f.getByID(ctx, id)
}

func (f *fakeResourceRepo) Create(ctx context.Context, r *model.Resource) error {
	if f.create == nil {
		return nil
	}
	return f.create(ctx, r)
}

func (f *fakeResourceRepo) Delete(ctx context.Context, id uint64) error {
	f.deleteInvoked = true
	if f.delete == nil {
		return nil
	}
	return f.delete(ctx, id)
}

type fakeSupervisionRepo struct {
	coverage    func(ctx context.Context, semester string) (int64, int64, error)
	byDept      func(ctx context.Context, semester string) ([]repository.DeptCoverageRow, error)
	listPlans   func(ctx context.Context, p repository.PlanListParams) ([]repository.PlanRow, int64, error)
	countPlans  func(ctx context.Context, status string) (int64, error)
	lastPlanReq *repository.PlanListParams
}

func (f *fakeSupervisionRepo) Coverage(ctx context.Context, semester string) (int64, int64, error) {
	if f.coverage == nil {
		return 0, 0, nil
	}
	return f.coverage(ctx, semester)
}

func (f *fakeSupervisionRepo) CoverageByDepartment(ctx context.Context, semester string) ([]repository.DeptCoverageRow, error) {
	if f.byDept == nil {
		return nil, nil
	}
	return f.byDept(ctx, semester)
}

func (f *fakeSupervisionRepo) ListPlans(ctx context.Context, p repository.PlanListParams) ([]repository.PlanRow, int64, error) {
	f.lastPlanReq = &p
	if f.listPlans == nil {
		return nil, 0, nil
	}
	return f.listPlans(ctx, p)
}

func (f *fakeSupervisionRepo) CountPlans(ctx context.Context, status string) (int64, error) {
	if f.countPlans == nil {
		return 0, nil
	}
	return f.countPlans(ctx, status)
}
