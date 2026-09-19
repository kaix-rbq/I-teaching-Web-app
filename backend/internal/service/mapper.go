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

// toSessionDetail 映射单场授课记录详情。
func toSessionDetail(row repository.SessionDetailRow) dto.SessionDetail {
	return dto.SessionDetail{
		ID:          row.ID,
		CourseID:    row.CourseID,
		CourseCode:  row.CourseCode,
		CourseName:  row.CourseName,
		ClassID:     row.ClassID,
		ClassName:   row.ClassName,
		TeacherID:   row.TeacherID,
		TeacherName: row.TeacherName,
		Semester:    row.Semester,
		SessionDate: row.SessionDate.Format("2006-01-02"),
		Period:      row.Period,
		Topic:       row.Topic,
		Status:      row.Status,
	}
}

// toSessionListItems 映射授课记录列表（双侧评分摘要保留两位小数）。
func toSessionListItems(rows []repository.SessionListRow) []dto.SessionListItem {
	out := make([]dto.SessionListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.SessionListItem{
			ID:              row.ID,
			SessionDate:     row.SessionDate.Format("2006-01-02"),
			Period:          row.Period,
			Topic:           row.Topic,
			Status:          row.Status,
			SupervisorScore: round2Ptr(row.SupervisorTotal),
			AgentScore:      round2Ptr(row.AgentTotal),
			EvaluationCount: row.EvaluationCount,
		})
	}
	return out
}

// toEvaluationDTO 映射单条评价（时间输出 ISO 8601）。
func toEvaluationDTO(row repository.EvaluationRow) dto.EvaluationDTO {
	return dto.EvaluationDTO{
		EvaluatorID:    row.EvaluatorID,
		EvaluatorName:  row.EvaluatorName,
		EvaluatorType:  row.EvaluatorType,
		AIModelVersion: row.AIModelVersion,
		AIConfidence:   round2Ptr(row.AIConfidence),
		FormulaVersion: row.FormulaVersion,
		Objective:      uint8PtrToIntPtr(row.ObjectiveScore),
		Content:        uint8PtrToIntPtr(row.ContentScore),
		Interaction:    uint8PtrToIntPtr(row.InteractionScore),
		Organization:   uint8PtrToIntPtr(row.OrganizationScore),
		Frontier:       uint8PtrToIntPtr(row.FrontierScore),
		TotalScore:     round2Ptr(row.TotalScore),
		Comment:        row.Comment,
		Highlights:     row.Highlights,
		Improvements:   row.Improvements,
		Suggestions:    row.Suggestions,
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Format(time.RFC3339),
	}
}

// round2Ptr 保留两位小数（nil 透传）。
func round2Ptr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	rounded := round2(*v)
	return &rounded
}

// uint8PtrToIntPtr 把仓储层的维度原始分（1-5）转换为 DTO 的 *int。
func uint8PtrToIntPtr(v *uint8) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}
