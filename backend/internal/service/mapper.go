package service

import (
	"time"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
)

// toCourseListItem 把联表行映射为列表项 DTO。
func toCourseListItem(row repository.CourseRow) dto.CourseListItem {
	return dto.CourseListItem{
		ID:            row.ID,
		Code:          row.Code,
		Name:          row.Name,
		Credit:        row.Credit,
		Hours:         row.Hours,
		Description:   row.Description,
		TeacherID:     row.TeacherID,
		TeacherName:   row.TeacherName,
		DepartmentID:  row.DepartmentID,
		Department:    row.Department,
		Semester:      row.Semester,
		ClassCount:    row.ClassCount,
		StudentCount:  row.StudentCount,
		ResourceCount: row.ResourceCount,
		Status:        row.Status,
	}
}

// toCourseListItems 批量映射列表项。
func toCourseListItems(rows []repository.CourseRow) []dto.CourseListItem {
	items := make([]dto.CourseListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toCourseListItem(row))
	}
	return items
}

// toClassInfoDTOs 映射开课班级列表。
func toClassInfoDTOs(classes []model.ClassInfo) []dto.ClassInfoDTO {
	out := make([]dto.ClassInfoDTO, 0, len(classes))
	for _, c := range classes {
		out = append(out, dto.ClassInfoDTO{
			ID:           c.ID,
			ClassName:    c.ClassName,
			Schedule:     c.Schedule,
			Location:     c.Location,
			StudentCount: c.StudentCount,
		})
	}
	return out
}

// toResourceDTOs 映射资源列表（uploadedAt 输出 ISO 8601）。
func toResourceDTOs(rows []repository.ResourceRow) []dto.ResourceDTO {
	out := make([]dto.ResourceDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.ResourceDTO{
			ID:         row.ID,
			CourseID:   row.CourseID,
			Name:       row.Name,
			Type:       row.Type,
			Size:       row.Size,
			Uploader:   row.Uploader,
			UploadedAt: row.UploadedAt.Format(time.RFC3339),
		})
	}
	return out
}

// toPlanItems 映射听评课安排（plannedDate 输出 YYYY-MM-DD）。
func toPlanItems(rows []repository.PlanRow) []dto.PlanItem {
	out := make([]dto.PlanItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.PlanItem{
			ID:             row.ID,
			CourseID:       row.CourseID,
			CourseName:     row.CourseName,
			TeacherName:    row.TeacherName,
			SupervisorName: row.SupervisorName,
			PlannedDate:    row.PlannedDate.Format("2006-01-02"),
			Status:         row.Status,
		})
	}
	return out
}
