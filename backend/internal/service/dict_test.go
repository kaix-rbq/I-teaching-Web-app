package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// fakeTimetableRepo 是课表仓储的内存假实现（B11）。
type fakeTimetableRepo struct {
	rows []repository.TimetableRow
}

func (f *fakeTimetableRepo) List(ctx context.Context, flt repository.TimetableFilter) ([]repository.TimetableRow, error) {
	out := make([]repository.TimetableRow, 0, len(f.rows))
	for _, r := range f.rows {
		if flt.TeacherID > 0 && r.TeacherID != flt.TeacherID {
			continue
		}
		if flt.Semester != "" && r.Semester != flt.Semester {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeTimetableRepo) Create(ctx context.Context, t *model.CourseTimetable) error {
	t.ID = uint64(len(f.rows) + 1)
	f.rows = append(f.rows, repository.TimetableRow{
		ID: t.ID, CourseID: t.CourseID, CourseCode: "SE3101", CourseName: "软件项目管理",
		ClassID: t.ClassID, ClassName: "软工2201", TeacherID: t.TeacherID,
		TeacherName: "李明", Semester: t.Semester,
	})
	return nil
}

func (f *fakeTimetableRepo) Delete(ctx context.Context, id uint64) error {
	for i, r := range f.rows {
		if r.ID == id {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeTimetableRepo) ExistsDuplicate(ctx context.Context, courseID, classID, teacherID uint64, semester string) (bool, error) {
	for _, r := range f.rows {
		if r.CourseID == courseID && r.ClassID == classID && r.TeacherID == teacherID && r.Semester == semester {
			return true, nil
		}
	}
	return false, nil
}

func newDictSvc() (*fakeTimetableRepo, DictService) {
	tables := &fakeTimetableRepo{rows: []repository.TimetableRow{
		{ID: 1, CourseID: 1, CourseCode: "SE3101", CourseName: "软件项目管理", ClassID: 7, ClassName: "软工2201", TeacherID: 2, TeacherName: "李明", Semester: CurrentSemester},
		{ID: 2, CourseID: 2, CourseCode: "SE2104", CourseName: "操作系统", ClassID: 3, ClassName: "软件2301", TeacherID: 3, TeacherName: "张华", Semester: CurrentSemester},
	}}
	courses := &fakeCourseRepo{
		getDetail: func(ctx context.Context, id uint64) (*repository.CourseRow, error) {
			if id != 1 {
				return nil, repository.ErrNotImplemented
			}
			return &repository.CourseRow{ID: 1, Code: "SE3101", Name: "软件项目管理", TeacherID: 2, Semester: CurrentSemester}, nil
		},
		listClasses: func(ctx context.Context, courseID uint64) ([]model.ClassInfo, error) {
			return []model.ClassInfo{{ID: 7, ClassName: "软工2201"}, {ID: 8, ClassName: "软工2202"}}, nil
		},
	}
	users := &fakeUserRepo{
		getByID: func(ctx context.Context, id uint64) (*model.User, error) {
			if id == 2 {
				return &model.User{ID: 2, Name: "李明", Role: model.RoleTeacher}, nil
			}
			return nil, repository.ErrNotImplemented
		},
	}
	return tables, NewDictService(users, courses, tables)
}

// TestMyTimetableScope 覆盖课表下拉的数据范围（B11）：teacher 只能看本人。
func TestMyTimetableScope(t *testing.T) {
	_, svc := newDictSvc()

	items, err := svc.MyTimetable(context.Background(), RoleTeacher, 0, 2, 0, "")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, uint64(2), items[0].TeacherID)

	// teacher 传他人 id 也只返回本人（不泄露他人课表）。
	items, err = svc.MyTimetable(context.Background(), RoleTeacher, 0, 2, 3, "")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, uint64(2), items[0].TeacherID)

	// supervisor 可按 teacherId 过滤。
	items, err = svc.MyTimetable(context.Background(), RoleSupervisor, 0, 5, 3, "")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, uint64(3), items[0].TeacherID)
}

// TestCreateTimetableEntryRules 覆盖课表维护校验（B11）。
func TestCreateTimetableEntryRules(t *testing.T) {
	t.Run("班级错配返回 40002", func(t *testing.T) {
		_, svc := newDictSvc()
		_, err := svc.CreateTimetableEntry(context.Background(), dto.TimetableCreateReq{CourseID: 1, ClassID: 99})
		require.Error(t, err)
		assert.Equal(t, errcode.BizRule, errcode.From(err).Code)
	})

	t.Run("重复条目返回 40901", func(t *testing.T) {
		_, svc := newDictSvc()
		_, err := svc.CreateTimetableEntry(context.Background(), dto.TimetableCreateReq{CourseID: 1, ClassID: 7, TeacherID: 2})
		require.Error(t, err)
		assert.Equal(t, errcode.Conflict, errcode.From(err).Code)
	})

	t.Run("缺省教师取课程任课教师", func(t *testing.T) {
		_, svc := newDictSvc()
		it, err := svc.CreateTimetableEntry(context.Background(), dto.TimetableCreateReq{CourseID: 1, ClassID: 8})
		require.NoError(t, err)
		assert.Equal(t, uint64(2), it.TeacherID)
		assert.Equal(t, CurrentSemester, it.Semester)
	})
}
