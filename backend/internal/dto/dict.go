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

// TimetableItem 是课表行（B11 建课下拉 + V13 教师工作台周课表）。
// weekday：1=周一…7=周日；0=历史数据未排课；periodStart/periodEnd：1-11，0=未指定。
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
	Weekday     int    `json:"weekday"`
	PeriodStart int    `json:"periodStart"`
	PeriodEnd   int    `json:"periodEnd"`
	Classroom   string `json:"classroom"`
	Remark      string `json:"remark"`
}

// TimetableCreateReq 是 POST /timetable 的请求体（director / supervisor 维护）。
// V13 新字段全部可选：旧调用方只传课程/班级仍保持原语义。
type TimetableCreateReq struct {
	CourseID    uint64 `json:"courseId" binding:"required"`
	ClassID     uint64 `json:"classId" binding:"required"`
	TeacherID   uint64 `json:"teacherId"`
	Semester    string `json:"semester" binding:"max=16"`
	Weekday     *int   `json:"weekday" binding:"omitempty,min=1,max=7"`
	PeriodStart *int   `json:"periodStart" binding:"omitempty,min=1,max=11"`
	PeriodEnd   *int   `json:"periodEnd" binding:"omitempty,min=1,max=11"`
	Classroom   string `json:"classroom" binding:"max=64"`
	Remark      string `json:"remark" binding:"max=255"`
}

// TimetableRemarkReq 是 PATCH /timetable/:id/remark 的请求体（仅本人教师）。
type TimetableRemarkReq struct {
	Remark string `json:"remark" binding:"max=255"`
}
