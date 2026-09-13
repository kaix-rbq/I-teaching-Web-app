package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// ResourceService 负责课程资源的查询、上传与删除。
type ResourceService interface {
	List(ctx context.Context, role string, deptID, userID, courseID uint64) ([]dto.ResourceDTO, error)
	Upload(ctx context.Context, userID, courseID uint64, file *multipart.FileHeader) (*dto.ResourceDTO, error)
	Delete(ctx context.Context, userID, resourceID uint64) error
	ForDownload(ctx context.Context, role string, deptID, userID, resourceID uint64) (*model.Resource, error)
}

type resourceService struct {
	resources repository.ResourceRepository
	courses   repository.CourseRepository
	cfg       config.UploadConfig
}

// NewResourceService 构造资源服务。
func NewResourceService(
	resources repository.ResourceRepository,
	courses repository.CourseRepository,
	cfg config.UploadConfig,
) ResourceService {
	return &resourceService{resources: resources, courses: courses, cfg: cfg}
}

// List 返回课程资源列表；沿用课程的读取数据范围。
func (s *resourceService) List(
	ctx context.Context, role string, deptID, userID, courseID uint64,
) ([]dto.ResourceDTO, error) {
	row, err := s.courses.GetDetail(ctx, courseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "课程不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课程失败", err)
	}
	if err := checkCourseScope(ScopeFor(role, deptID, userID), row); err != nil {
		return nil, err
	}
	rows, err := s.resources.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询课程资源失败", err)
	}
	return toResourceDTOs(rows), nil
}

// Upload 校验归属与文件合法性后落盘并写库。
func (s *resourceService) Upload(
	ctx context.Context, userID, courseID uint64, file *multipart.FileHeader,
) (*dto.ResourceDTO, error) {
	row, err := s.courses.GetDetail(ctx, courseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "课程不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课程失败", err)
	}
	if row.TeacherID != userID {
		return nil, errcode.New(errcode.ForbiddenData, "只能上传本人课程的资源")
	}
	if row.Status != model.CourseStatusOpen {
		return nil, errcode.New(errcode.ForbiddenData, "课程未开课，无法上传资源")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.extAllowed(ext) {
		return nil, errcode.Newf(errcode.Params, "不支持的文件类型 %s，允许：%s", ext, strings.Join(s.cfg.AllowExt, "/"))
	}
	if s.cfg.MaxSize > 0 && file.Size > s.cfg.MaxSize {
		return nil, errcode.Newf(errcode.Params, "文件超过大小上限 %d MB", s.cfg.MaxSize/1024/1024)
	}
	if err := sniffAllowed(file); err != nil {
		return nil, err
	}

	relPath, err := s.store(courseID, ext, file)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "保存文件失败", err)
	}

	record := &model.Resource{
		CourseID:   courseID,
		Name:       filepath.Base(file.Filename),
		Type:       mapResourceType(ext),
		Size:       file.Size,
		FilePath:   relPath,
		UploaderID: userID,
		// uploaded_at 非 GORM 约定字段，需显式赋值，否则写入零值时间被 MySQL 拒绝。
		UploadedAt: time.Now(),
	}
	if err := s.resources.Create(ctx, record); err != nil {
		// 落库失败时清理已写入的磁盘文件，避免孤儿文件。
		if rmErr := os.Remove(relPath); rmErr != nil {
			slog.Warn("cleanup orphan upload failed", "path", relPath, "error", rmErr)
		}
		return nil, errcode.Wrap(errcode.Internal, "保存资源记录失败", err)
	}

	rows, err := s.resources.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询课程资源失败", err)
	}
	for _, item := range toResourceDTOs(rows) {
		if item.ID == record.ID {
			return &item, nil
		}
	}
	return &dto.ResourceDTO{
		ID:         record.ID,
		CourseID:   record.CourseID,
		Name:       record.Name,
		Type:       record.Type,
		Size:       record.Size,
		UploadedAt: record.UploadedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// Delete 先删库记录，再删磁盘文件；删文件失败只记日志不回滚。
func (s *resourceService) Delete(ctx context.Context, userID, resourceID uint64) error {
	record, err := s.resources.GetByID(ctx, resourceID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errcode.New(errcode.NotFound, "资源不存在")
		}
		return errcode.Wrap(errcode.Internal, "查询资源失败", err)
	}

	row, err := s.courses.GetDetail(ctx, record.CourseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errcode.New(errcode.NotFound, "资源所属课程不存在")
		}
		return errcode.Wrap(errcode.Internal, "查询课程失败", err)
	}
	if row.TeacherID != userID {
		return errcode.New(errcode.ForbiddenData, "只能删除本人课程的资源")
	}

	if err := s.resources.Delete(ctx, resourceID); err != nil {
		return errcode.Wrap(errcode.Internal, "删除资源记录失败", err)
	}
	if err := os.Remove(record.FilePath); err != nil && !os.IsNotExist(err) {
		slog.Warn("remove resource file failed", "path", record.FilePath, "error", err)
	}
	return nil
}

// ForDownload 返回可下载的资源记录（校验数据范围）。
func (s *resourceService) ForDownload(
	ctx context.Context, role string, deptID, userID, resourceID uint64,
) (*model.Resource, error) {
	record, err := s.resources.GetByID(ctx, resourceID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "资源不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询资源失败", err)
	}
	row, err := s.courses.GetDetail(ctx, record.CourseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "资源所属课程不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课程失败", err)
	}
	if err := checkCourseScope(ScopeFor(role, deptID, userID), row); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *resourceService) extAllowed(ext string) bool {
	for _, allowed := range s.cfg.AllowExt {
		if strings.EqualFold(allowed, ext) {
			return true
		}
	}
	return false
}

// store 把上传文件写入 {upload.dir}/{courseID}/{uuid}{ext}，返回可落库的路径。
func (s *resourceService) store(courseID uint64, ext string, file *multipart.FileHeader) (string, error) {
	dir := filepath.Join(s.cfg.Dir, fmt.Sprintf("%d", courseID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}
	target := filepath.Join(dir, uuid.NewString()+ext)

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("open upload: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(target)
	if err != nil {
		return "", fmt.Errorf("create target: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("copy upload: %w", err)
	}
	return target, nil
}

// sniffAllowed 嗅探文件头，拒绝可执行文件与脚本（扩展名白名单之外的第二道校验）。
func sniffAllowed(file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return errcode.Wrap(errcode.Params, "无法读取上传文件", err)
	}
	defer func() { _ = src.Close() }()

	head := make([]byte, 512)
	n, err := src.Read(head)
	if err != nil && err != io.EOF {
		return errcode.Wrap(errcode.Params, "无法读取上传文件", err)
	}
	detected := http.DetectContentType(head[:n])
	for _, deny := range deniedMIMEs {
		if strings.HasPrefix(detected, deny) {
			return errcode.Newf(errcode.Params, "文件内容校验未通过（%s）", detected)
		}
	}
	return nil
}

// deniedMIMEs 是嗅探出的可执行/脚本类型黑名单，命中即拒绝。
var deniedMIMEs = []string{
	"text/html",
	"text/x-shellscript",
	"application/x-msdownload",
	"application/x-dosexec",
	"application/x-executable",
	"application/x-sharedlib",
	"application/x-httpd-php",
	"application/x-mach-binary",
}

// mapResourceType 按扩展名映射资源类型枚举。
func mapResourceType(ext string) string {
	switch ext {
	case ".pdf":
		return model.ResourceTypePDF
	case ".doc", ".docx":
		return model.ResourceTypeDoc
	case ".ppt", ".pptx":
		return model.ResourceTypePPT
	case ".mp4", ".avi", ".mov":
		return model.ResourceTypeVideo
	case ".zip", ".rar", ".7z":
		return model.ResourceTypeZip
	default:
		return model.ResourceTypeOther
	}
}
