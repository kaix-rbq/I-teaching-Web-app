package dto

// ResourceDTO 是课程资源的展示结构（不含磁盘路径）。
type ResourceDTO struct {
	ID         uint64 `json:"id"`
	CourseID   uint64 `json:"courseId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Size       int64  `json:"size"`
	Uploader   string `json:"uploader"`
	UploadedAt string `json:"uploadedAt"`
}
