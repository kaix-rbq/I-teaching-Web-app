package dto

// DeptOption 是教研室字典项。
type DeptOption struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// TeacherOption 是教师字典项。
// departmentId 为前端「按教研室过滤教师下拉」所需的补充字段（与前端的 User.departmentId 对齐）。
type TeacherOption struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	DepartmentID uint64 `json:"departmentId,omitempty"`
}

// TimetableItem 是最小课表下拉行（B11）：建课时课程/班级下拉的数据源。
type TimetableItem struct {
	ID          uint64 `json:"id"`
	CourseID    uint64 `json:"courseId"`
	CourseCode  string `json:"courseCode"`
	CourseName  string `json:"courseName"`
	ClassID     uint64 `json:"classId"`
	ClassName   string `json:"className"`
	TeacherID   uint64 `json:"teacherId"`
	TeacherName string `json:"teacherName"`
	Semester    string `json:"semester"`
}

// TimetableCreateReq 是 POST /timetable 的请求体（director / supervisor 维护）。
type TimetableCreateReq struct {
	CourseID  uint64 `json:"courseId" binding:"required"`
	ClassID   uint64 `json:"classId" binding:"required"`
	TeacherID uint64 `json:"teacherId"`
	Semester  string `json:"semester" binding:"max=16"`
}
