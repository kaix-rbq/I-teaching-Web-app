package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// Notifier 是站内信通知接口（B12）；由 NotificationService 实现，注入到会话/AI 服务做副作用。
type Notifier interface {
	NotifySupervisorComment(ctx context.Context, sessionID uint64)
	NotifyAIIssue(ctx context.Context, sessionID uint64, title string)
}

// ImprovementService 负责待改进清单（教师自评完成即关闭）。
type ImprovementService interface {
	List(ctx context.Context, role string, userID uint64, q dto.ImprovementListQuery) (*dto.PageResult[dto.ImprovementItemDTO], error)
	Create(ctx context.Context, role string, userID uint64, req dto.ImprovementCreateReq) (*dto.ImprovementItemDTO, error)
	Update(ctx context.Context, role string, userID, id uint64, req dto.ImprovementUpdateReq) (*dto.ImprovementItemDTO, error)
	Complete(ctx context.Context, role string, userID, id uint64, req dto.ImprovementCompleteReq) (*dto.ImprovementItemDTO, error)
	Delete(ctx context.Context, role string, userID, id uint64) error
}

type improvementService struct {
	repo repository.ImprovementRepository
}

// NewImprovementService 构造待改进清单服务。
func NewImprovementService(repo repository.ImprovementRepository) ImprovementService {
	return &improvementService{repo: repo}
}

func (s *improvementService) List(ctx context.Context, role string, userID uint64, q dto.ImprovementListQuery) (*dto.PageResult[dto.ImprovementItemDTO], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	f := repository.ImprovementListFilter{Status: q.Status, Page: page, PageSize: pageSize}
	switch role {
	case RoleTeacher:
		f.TeacherID = userID
	case RoleSupervisor:
		f.TeacherID = q.TeacherID
	default:
		return nil, errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message())
	}
	rows, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询待改进清单失败", err)
	}
	items := make([]dto.ImprovementItemDTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, toImprovementDTO(r))
	}
	return dto.NewPageResult(items, total, page, pageSize), nil
}

func (s *improvementService) Create(ctx context.Context, role string, userID uint64, req dto.ImprovementCreateReq) (*dto.ImprovementItemDTO, error) {
	if role != RoleTeacher {
		return nil, errcode.New(errcode.ForbiddenRole, "仅教师可创建改进项")
	}
	source := req.Source
	if source == "" {
		source = model.ImprovementSourceSelf
	}
	it := &model.ImprovementItem{
		TeacherID: userID, SessionID: req.SessionID, Title: req.Title,
		Source: source, Status: model.ImprovementStatusOpen,
	}
	if err := s.repo.Create(ctx, it); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "创建改进项失败", err)
	}
	return s.reload(ctx, it.ID, RoleTeacher, userID)
}

func (s *improvementService) Update(ctx context.Context, role string, userID, id uint64, req dto.ImprovementUpdateReq) (*dto.ImprovementItemDTO, error) {
	if role != RoleTeacher {
		return nil, errcode.New(errcode.ForbiddenRole, "仅教师可修改改进项")
	}
	if _, err := s.ensureOwn(ctx, id, userID); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateTitle(ctx, id, req.Title); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "更新改进项失败", err)
	}
	return s.reload(ctx, id, RoleTeacher, userID)
}

func (s *improvementService) Complete(ctx context.Context, role string, userID, id uint64, req dto.ImprovementCompleteReq) (*dto.ImprovementItemDTO, error) {
	if role != RoleTeacher {
		return nil, errcode.New(errcode.ForbiddenRole, "仅教师可完成改进项")
	}
	it, err := s.ensureOwn(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if it.Status != model.ImprovementStatusOpen && it.Status != model.ImprovementStatusDoneSelf {
		return nil, errcode.New(errcode.BizRule, "当前状态不可提交自评完成")
	}
	// 2026-10-20 简化：教师自评完成直接关闭，不再等待督导确认。
	if err := s.repo.UpdateStatus(ctx, id, map[string]any{
		"status": model.ImprovementStatusClosed, "done_note": req.DoneNote,
		"closed_by": userID, "closed_at": time.Now(),
	}); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "提交自评失败", err)
	}
	return s.reload(ctx, id, RoleTeacher, userID)
}

// Delete 删除本人改进项（仅创建教师本人）。
func (s *improvementService) Delete(ctx context.Context, role string, userID, id uint64) error {
	if role != RoleTeacher {
		return errcode.New(errcode.ForbiddenRole, "仅教师可删除改进项")
	}
	if _, err := s.ensureOwn(ctx, id, userID); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errcode.Wrap(errcode.Internal, "删除改进项失败", err)
	}
	return nil
}

// ensureOwn 校验改进项属于该教师（越权 40302），不存在 40401。
func (s *improvementService) ensureOwn(ctx context.Context, id, teacherID uint64) (*model.ImprovementItem, error) {
	it, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "改进项不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询改进项失败", err)
	}
	if it.TeacherID != teacherID {
		return nil, errcode.New(errcode.ForbiddenData, "无权操作他人的改进项")
	}
	return it, nil
}

func (s *improvementService) reload(ctx context.Context, id uint64, _ string, _ uint64) (*dto.ImprovementItemDTO, error) {
	row, err := s.repo.GetRowByID(ctx, id)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "回读改进项失败", err)
	}
	out := toImprovementDTO(*row)
	return &out, nil
}

func toImprovementDTO(r repository.ImprovementItemRow) dto.ImprovementItemDTO {
	out := dto.ImprovementItemDTO{
		ID:           r.ID,
		TeacherID:    r.TeacherID,
		TeacherName:  r.TeacherName,
		SessionID:    r.SessionID,
		Title:        r.Title,
		Source:       r.Source,
		Status:       r.Status,
		DoneNote:     r.DoneNote,
		ClosedByName: r.ClosedByName,
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
	}
	if r.ClosedBy != nil {
		out.ClosedBy = *r.ClosedBy
	}
	if r.ClosedAt != nil {
		out.ClosedAt = r.ClosedAt.Format(time.RFC3339)
	}
	return out
}

// ReflectionService 负责教学反思（N5，教师私有；督导关闭清单时可读关联反思）。
type ReflectionService interface {
	List(ctx context.Context, role string, userID uint64, q dto.ReflectionListQuery) ([]dto.ReflectionDTO, error)
	Create(ctx context.Context, role string, userID uint64, req dto.ReflectionUpsertReq) (*dto.ReflectionDTO, error)
	Update(ctx context.Context, role string, userID, id uint64, req dto.ReflectionUpsertReq) (*dto.ReflectionDTO, error)
}

type reflectionService struct {
	repo repository.ReflectionRepository
}

// NewReflectionService 构造教学反思服务。
func NewReflectionService(repo repository.ReflectionRepository) ReflectionService {
	return &reflectionService{repo: repo}
}

func (s *reflectionService) List(ctx context.Context, role string, userID uint64, q dto.ReflectionListQuery) ([]dto.ReflectionDTO, error) {
	var teacherID uint64
	switch role {
	case RoleTeacher:
		teacherID = userID
	case RoleSupervisor:
		// 督导仅在关闭清单时按 sessionId 读关联反思（§5 权限矩阵）。
		if q.SessionID == 0 {
			return nil, errcode.New(errcode.Params, "督导查看反思需指定 sessionId")
		}
		rows, err := s.repo.ListBySession(ctx, q.SessionID)
		if err != nil {
			return nil, errcode.Wrap(errcode.Internal, "查询关联反思失败", err)
		}
		return toReflectionDTOs(rows), nil
	default:
		return nil, errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message())
	}
	rows, err := s.repo.List(ctx, teacherID, q.SessionID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询教学反思失败", err)
	}
	return toReflectionDTOs(rows), nil
}

func (s *reflectionService) Create(ctx context.Context, role string, userID uint64, req dto.ReflectionUpsertReq) (*dto.ReflectionDTO, error) {
	if role != RoleTeacher {
		return nil, errcode.New(errcode.ForbiddenRole, "仅教师可创建教学反思")
	}
	rf := &model.TeachingReflection{
		TeacherID: userID, SessionID: req.SessionID, SourceEvalID: req.SourceEvalID,
		Content: req.Content, Plan: req.Plan, NextAction: req.NextAction,
	}
	if err := s.repo.Create(ctx, rf); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "创建教学反思失败", err)
	}
	out := toReflectionDTO(*rf)
	return &out, nil
}

func (s *reflectionService) Update(ctx context.Context, role string, userID, id uint64, req dto.ReflectionUpsertReq) (*dto.ReflectionDTO, error) {
	if role != RoleTeacher {
		return nil, errcode.New(errcode.ForbiddenRole, "仅教师可修改教学反思")
	}
	rf, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "教学反思不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询教学反思失败", err)
	}
	if rf.TeacherID != userID {
		return nil, errcode.New(errcode.ForbiddenData, "无权修改他人的教学反思")
	}
	if err := s.repo.Update(ctx, id, map[string]any{
		"content": req.Content, "plan": req.Plan, "next_action": req.NextAction,
	}); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "更新教学反思失败", err)
	}
	rf.Content, rf.Plan, rf.NextAction = req.Content, req.Plan, req.NextAction
	out := toReflectionDTO(*rf)
	return &out, nil
}

func toReflectionDTOs(rows []model.TeachingReflection) []dto.ReflectionDTO {
	out := make([]dto.ReflectionDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toReflectionDTO(r))
	}
	return out
}

func toReflectionDTO(r model.TeachingReflection) dto.ReflectionDTO {
	return dto.ReflectionDTO{
		ID: r.ID, TeacherID: r.TeacherID, SessionID: r.SessionID, SourceEvalID: r.SourceEvalID,
		Content: r.Content, Plan: r.Plan, NextAction: r.NextAction,
		CreatedAt: r.CreatedAt.Format(time.RFC3339), UpdatedAt: r.UpdatedAt.Format(time.RFC3339),
	}
}

// NotificationService 负责站内信（B12）：只推 supervisor_comment / ai_issue，红色收敛到工作台与督导队列。
type NotificationService interface {
	Notifier
	List(ctx context.Context, userID uint64, q dto.NotificationListQuery) (*dto.NotificationListDTO, error)
	MarkRead(ctx context.Context, userID, id uint64) error
	MarkAllRead(ctx context.Context, userID uint64) error
}

type notificationService struct {
	repo     repository.NotificationRepository
	sessions repository.SessionRepository
}

// NewNotificationService 构造站内信服务。
func NewNotificationService(repo repository.NotificationRepository, sessions repository.SessionRepository) NotificationService {
	return &notificationService{repo: repo, sessions: sessions}
}

// NotifySupervisorComment 生成「督导评语」站内信，通知该场次授课教师。
func (s *notificationService) NotifySupervisorComment(ctx context.Context, sessionID uint64) {
	s.notifyTeacher(ctx, sessionID, model.NotificationTypeSupervisorComment, "收到新的督导评语")
}

// NotifyAIIssue 生成「AI 发现问题」站内信。
func (s *notificationService) NotifyAIIssue(ctx context.Context, sessionID uint64, title string) {
	s.notifyTeacher(ctx, sessionID, model.NotificationTypeAIIssue, title)
}

func (s *notificationService) notifyTeacher(ctx context.Context, sessionID uint64, ntype, title string) {
	row, err := s.sessions.GetDetail(ctx, sessionID)
	if err != nil {
		slog.Warn("创建站内信失败：查询授课记录", "sessionID", sessionID, "err", err)
		return
	}
	if row.TeacherID == 0 {
		return
	}
	payload, _ := json.Marshal(map[string]string{"title": title, "courseName": row.CourseName})
	p := string(payload)
	if err := s.repo.Create(ctx, &model.UserNotification{
		UserID: row.TeacherID, Type: ntype, RefID: sessionID, Payload: &p,
	}); err != nil {
		slog.Warn("创建站内信失败", "sessionID", sessionID, "err", err)
	}
}

func (s *notificationService) List(ctx context.Context, userID uint64, q dto.NotificationListQuery) (*dto.NotificationListDTO, error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	rows, total, unread, err := s.repo.List(ctx, userID, page, pageSize)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询站内信失败", err)
	}
	items := make([]dto.NotificationDTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, toNotificationDTO(r))
	}
	return &dto.NotificationListDTO{List: items, Total: total, Unread: unread, Page: page, PageSize: pageSize}, nil
}

func (s *notificationService) MarkRead(ctx context.Context, userID, id uint64) error {
	if err := s.repo.MarkRead(ctx, id, userID); err != nil {
		return errcode.Wrap(errcode.Internal, "标记已读失败", err)
	}
	return nil
}

func (s *notificationService) MarkAllRead(ctx context.Context, userID uint64) error {
	if err := s.repo.MarkAllRead(ctx, userID); err != nil {
		return errcode.Wrap(errcode.Internal, "全部已读失败", err)
	}
	return nil
}

func toNotificationDTO(r model.UserNotification) dto.NotificationDTO {
	title := ""
	if r.Payload != nil {
		var p struct {
			Title string `json:"title"`
		}
		if err := json.Unmarshal([]byte(*r.Payload), &p); err == nil {
			title = p.Title
		}
	}
	return dto.NotificationDTO{
		ID: r.ID, Type: r.Type, RefID: r.RefID, Title: title, IsRead: r.IsRead,
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
	}
}
