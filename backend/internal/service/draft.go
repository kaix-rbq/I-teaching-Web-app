package service

import (
	"context"
	"time"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// DraftService 负责督导评估草稿（保存 / 查询 / 提交 / 删除）。
// 草稿与正式评价分离：草稿只属于创建它的督导，提交时复用 SessionService 的
// 正式提交口径（五维校验 + 覆盖率状态更新），避免两套计分逻辑漂移。
type DraftService interface {
	Save(ctx context.Context, supervisorID, sessionID uint64, req dto.DraftUpsertReq) (*dto.DraftDTO, error)
	GetBySession(ctx context.Context, supervisorID, sessionID uint64) (*dto.DraftDTO, error)
	List(ctx context.Context, supervisorID uint64, q dto.DraftListQuery) (*dto.PageResult[dto.DraftDTO], error)
	Recent(ctx context.Context, supervisorID uint64, limit int) ([]dto.DraftDTO, error)
	Delete(ctx context.Context, supervisorID, id uint64) error
	Submit(ctx context.Context, supervisorID, id uint64) (*dto.EvaluationDTO, error)
}

type draftService struct {
	drafts   repository.DraftRepository
	sessions SessionService
}

// NewDraftService 构造草稿服务。
func NewDraftService(drafts repository.DraftRepository, sessions SessionService) DraftService {
	return &draftService{drafts: drafts, sessions: sessions}
}

// Save 保存/覆盖某场次的督导草稿（允许部分维度为空）。
func (s *draftService) Save(
	ctx context.Context, supervisorID, sessionID uint64, req dto.DraftUpsertReq,
) (*dto.DraftDTO, error) {
	session, err := s.sessions.Detail(ctx, RoleSupervisor, 0, 0, sessionID)
	if err != nil {
		return nil, err
	}

	draft := &model.EvaluationDraft{
		SessionID:         sessionID,
		SupervisorID:      supervisorID,
		ObjectiveScore:    intPtrToUint8(req.Objective),
		ContentScore:      intPtrToUint8(req.Content),
		InteractionScore:  intPtrToUint8(req.Interaction),
		OrganizationScore: intPtrToUint8(req.Organization),
		FrontierScore:     intPtrToUint8(req.Frontier),
		Comment:           req.Comment,
		Highlights:        req.Highlights,
		Improvements:      req.Improvements,
		Suggestions:       req.Suggestions,
	}
	if err := s.drafts.Upsert(ctx, draft); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "保存草稿失败", err)
	}

	saved, err := s.drafts.GetBySession(ctx, sessionID, supervisorID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "读取草稿失败", err)
	}
	out := toDraftDTOFromModel(saved)
	out.CourseID = session.CourseID
	out.CourseCode = session.CourseCode
	out.CourseName = session.CourseName
	out.TeacherName = session.TeacherName
	out.SessionDate = session.SessionDate
	out.Period = session.Period
	out.Topic = session.Topic
	return &out, nil
}

// GetBySession 返回当前督导在某场次的草稿；不存在时返回 (nil, nil)，不视作错误。
func (s *draftService) GetBySession(ctx context.Context, supervisorID, sessionID uint64) (*dto.DraftDTO, error) {
	draft, err := s.drafts.GetBySession(ctx, sessionID, supervisorID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, nil
		}
		return nil, errcode.Wrap(errcode.Internal, "读取草稿失败", err)
	}
	out := toDraftDTOFromModel(draft)
	return &out, nil
}

// List 返回当前督导的草稿分页列表（可按课程名查询）。
func (s *draftService) List(
	ctx context.Context, supervisorID uint64, q dto.DraftListQuery,
) (*dto.PageResult[dto.DraftDTO], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	rows, total, err := s.drafts.ListBySupervisor(ctx, repository.DraftListParams{
		SupervisorID: supervisorID,
		Keyword:      q.Keyword,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询草稿失败", err)
	}
	items := make([]dto.DraftDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDraftDTOFromRow(row))
	}
	return dto.NewPageResult(items, total, page, pageSize), nil
}

// Recent 返回最近创建的草稿（工作台「草稿箱」区块）。
func (s *draftService) Recent(ctx context.Context, supervisorID uint64, limit int) ([]dto.DraftDTO, error) {
	if limit <= 0 {
		limit = 3
	}
	rows, err := s.drafts.RecentBySupervisor(ctx, supervisorID, limit)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询最近草稿失败", err)
	}
	items := make([]dto.DraftDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDraftDTOFromRow(row))
	}
	return items, nil
}

// Delete 删除当前督导的草稿（越权/不存在返回 40401）。
func (s *draftService) Delete(ctx context.Context, supervisorID, id uint64) error {
	if _, err := s.drafts.GetByID(ctx, id, supervisorID); err != nil {
		if repository.IsNotFound(err) {
			return errcode.New(errcode.NotFound, "草稿不存在")
		}
		return errcode.Wrap(errcode.Internal, "查询草稿失败", err)
	}
	if err := s.drafts.Delete(ctx, id, supervisorID); err != nil {
		return errcode.Wrap(errcode.Internal, "删除草稿失败", err)
	}
	return nil
}

// Submit 把草稿提升为正式评价：复用正式提交口径（五维必填、写库、更新场次状态），
// 成功后删除草稿。维度不全时返回 40002，草稿保留以便继续编辑。
func (s *draftService) Submit(ctx context.Context, supervisorID, id uint64) (*dto.EvaluationDTO, error) {
	row, err := s.drafts.GetByID(ctx, id, supervisorID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "草稿不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询草稿失败", err)
	}

	result, err := s.sessions.SubmitSupervisorEvaluation(ctx, supervisorID, row.SessionID, dto.SupervisorEvaluationReq{
		Objective:    uint8PtrToIntPtr(row.ObjectiveScore),
		Content:      uint8PtrToIntPtr(row.ContentScore),
		Interaction:  uint8PtrToIntPtr(row.InteractionScore),
		Organization: uint8PtrToIntPtr(row.OrganizationScore),
		Frontier:     uint8PtrToIntPtr(row.FrontierScore),
		Comment:      row.Comment,
		Highlights:   row.Highlights,
		Improvements: row.Improvements,
		Suggestions:  row.Suggestions,
	})
	if err != nil {
		return nil, err
	}

	if err := s.drafts.Delete(ctx, id, supervisorID); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "提交后清理草稿失败", err)
	}
	return result, nil
}

// toDraftDTOFromRow 映射带课程上下文的草稿行。
func toDraftDTOFromRow(row repository.DraftListRow) dto.DraftDTO {
	return dto.DraftDTO{
		ID:           row.ID,
		SessionID:    row.SessionID,
		CourseID:     row.CourseID,
		CourseCode:   row.CourseCode,
		CourseName:   row.CourseName,
		TeacherName:  row.TeacherName,
		SessionDate:  row.SessionDate.Format("2006-01-02"),
		Period:       row.Period,
		Topic:        row.Topic,
		Objective:    uint8PtrToIntPtr(row.ObjectiveScore),
		Content:      uint8PtrToIntPtr(row.ContentScore),
		Interaction:  uint8PtrToIntPtr(row.InteractionScore),
		Organization: uint8PtrToIntPtr(row.OrganizationScore),
		Frontier:     uint8PtrToIntPtr(row.FrontierScore),
		Comment:      row.Comment,
		Highlights:   row.Highlights,
		Improvements: row.Improvements,
		Suggestions:  row.Suggestions,
		CreatedAt:    row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    row.UpdatedAt.Format(time.RFC3339),
	}
}

// toDraftDTOFromModel 映射无课程上下文的草稿（调用方可按需补全）。
func toDraftDTOFromModel(d *model.EvaluationDraft) dto.DraftDTO {
	return dto.DraftDTO{
		ID:           d.ID,
		SessionID:    d.SessionID,
		Objective:    uint8PtrToIntPtr(d.ObjectiveScore),
		Content:      uint8PtrToIntPtr(d.ContentScore),
		Interaction:  uint8PtrToIntPtr(d.InteractionScore),
		Organization: uint8PtrToIntPtr(d.OrganizationScore),
		Frontier:     uint8PtrToIntPtr(d.FrontierScore),
		Comment:      d.Comment,
		Highlights:   d.Highlights,
		Improvements: d.Improvements,
		Suggestions:  d.Suggestions,
		CreatedAt:    d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    d.UpdatedAt.Format(time.RFC3339),
	}
}
