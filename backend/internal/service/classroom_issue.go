package service

import (
	"context"
	"time"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// ClassroomIssueService 负责课堂问题登记与处理闭环。
// 范围：教师只看/确认本人；督导与主任看全部并处理（service 层裁剪）。
type ClassroomIssueService interface {
	List(ctx context.Context, role string, userID uint64) ([]dto.ClassroomIssueDTO, error)
	Create(ctx context.Context, role string, userID uint64, req dto.ClassroomIssueCreateReq) (*dto.ClassroomIssueDTO, error)
	Handle(ctx context.Context, role string, userID, id uint64, req dto.ClassroomIssueHandleReq) (*dto.ClassroomIssueDTO, error)
	Confirm(ctx context.Context, role string, userID, id uint64, req dto.ClassroomIssueConfirmReq) error
}

type classroomIssueService struct {
	repo repository.ClassroomIssueRepository
}

// NewClassroomIssueService 构造课堂问题服务。
func NewClassroomIssueService(repo repository.ClassroomIssueRepository) ClassroomIssueService {
	return &classroomIssueService{repo: repo}
}

func (s *classroomIssueService) List(ctx context.Context, role string, userID uint64) ([]dto.ClassroomIssueDTO, error) {
	var teacherID uint64
	switch role {
	case RoleTeacher:
		teacherID = userID
	case RoleSupervisor, RoleDirector:
		teacherID = 0
	default:
		return nil, errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message())
	}
	rows, err := s.repo.List(ctx, teacherID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询课堂问题失败", err)
	}
	out := make([]dto.ClassroomIssueDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, toClassroomIssueDTO(row))
	}
	return out, nil
}

func (s *classroomIssueService) Create(ctx context.Context, role string, userID uint64, req dto.ClassroomIssueCreateReq) (*dto.ClassroomIssueDTO, error) {
	if role != RoleTeacher {
		return nil, errcode.New(errcode.ForbiddenRole, "仅教师可登记课堂问题")
	}
	issue := &model.ClassroomIssue{
		TeacherID: userID,
		IssueType: req.IssueType,
		Content:   req.Content,
		Status:    model.IssueStatusOpen,
	}
	if err := s.repo.Create(ctx, issue); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "登记课堂问题失败", err)
	}
	return s.reload(ctx, issue.ID)
}

func (s *classroomIssueService) Handle(ctx context.Context, role string, userID, id uint64, req dto.ClassroomIssueHandleReq) (*dto.ClassroomIssueDTO, error) {
	if role != RoleSupervisor && role != RoleDirector {
		return nil, errcode.New(errcode.ForbiddenRole, "仅督导或主任可处理课堂问题")
	}
	issue, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "课堂问题不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课堂问题失败", err)
	}
	if issue.Status != model.IssueStatusOpen {
		return nil, errcode.New(errcode.BizRule, "该问题已处理，等待教师确认")
	}
	if err := s.repo.Handle(ctx, id, userID, req.HandleNote); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "处理课堂问题失败", err)
	}
	return s.reload(ctx, id)
}

func (s *classroomIssueService) Confirm(ctx context.Context, role string, userID, id uint64, req dto.ClassroomIssueConfirmReq) error {
	if role != RoleTeacher {
		return errcode.New(errcode.ForbiddenRole, "仅教师可确认课堂问题")
	}
	issue, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return errcode.New(errcode.NotFound, "课堂问题不存在")
		}
		return errcode.Wrap(errcode.Internal, "查询课堂问题失败", err)
	}
	if issue.TeacherID != userID {
		return errcode.New(errcode.ForbiddenData, "无权确认他人的课堂问题")
	}
	if issue.Status != model.IssueStatusHandled {
		return errcode.New(errcode.BizRule, "该问题尚未处理，暂不可确认")
	}
	if req.Resolved {
		if err := s.repo.Delete(ctx, id); err != nil {
			return errcode.Wrap(errcode.Internal, "删除已解决课堂问题失败", err)
		}
		return nil
	}
	if err := s.repo.Reopen(ctx, id); err != nil {
		return errcode.Wrap(errcode.Internal, "回退课堂问题失败", err)
	}
	return nil
}

func (s *classroomIssueService) reload(ctx context.Context, id uint64) (*dto.ClassroomIssueDTO, error) {
	rows, err := s.repo.List(ctx, 0)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "回读课堂问题失败", err)
	}
	for _, row := range rows {
		if row.ID == id {
			out := toClassroomIssueDTO(row)
			return &out, nil
		}
	}
	return nil, errcode.New(errcode.NotFound, "课堂问题不存在")
}

func toClassroomIssueDTO(row repository.ClassroomIssueRow) dto.ClassroomIssueDTO {
	out := dto.ClassroomIssueDTO{
		ID: row.ID, TeacherID: row.TeacherID, TeacherName: row.TeacherName,
		IssueType: row.IssueType, Content: row.Content, Status: row.Status,
		HandleNote: row.HandleNote, HandlerID: row.HandlerID, HandlerName: row.HandlerName,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
	}
	if row.HandledAt != nil {
		out.HandledAt = row.HandledAt.Format(time.RFC3339)
	}
	return out
}
