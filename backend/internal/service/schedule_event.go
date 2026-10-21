package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// ScheduleEventService 负责主任事务（周视图事务管理卡）。
// 数据范围：仅 owner 本人；service 层强制，不依赖前端。
type ScheduleEventService interface {
	List(ctx context.Context, role string, ownerID uint64, q dto.ScheduleEventListQuery) ([]dto.ScheduleEventDTO, error)
	Create(ctx context.Context, role string, ownerID uint64, req dto.ScheduleEventCreateReq) (*dto.ScheduleEventDTO, error)
	UpdateRemark(ctx context.Context, role string, ownerID, id uint64, req dto.ScheduleEventRemarkReq) (*dto.ScheduleEventDTO, error)
	Delete(ctx context.Context, role string, ownerID, id uint64) error
}

type scheduleEventService struct {
	repo repository.ScheduleEventRepository
}

// NewScheduleEventService 构造主任事务服务。
func NewScheduleEventService(repo repository.ScheduleEventRepository) ScheduleEventService {
	return &scheduleEventService{repo: repo}
}

func (s *scheduleEventService) List(ctx context.Context, role string, ownerID uint64, q dto.ScheduleEventListQuery) ([]dto.ScheduleEventDTO, error) {
	if role != RoleDirector {
		return nil, errcode.New(errcode.ForbiddenRole, "仅主任可查看事务管理")
	}
	from, to, err := scheduleRange(q.From, q.To)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, ownerID, from, to)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询事务失败", err)
	}
	out := make([]dto.ScheduleEventDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, toScheduleEventDTO(row))
	}
	return out, nil
}

func (s *scheduleEventService) Create(ctx context.Context, role string, ownerID uint64, req dto.ScheduleEventCreateReq) (*dto.ScheduleEventDTO, error) {
	if role != RoleDirector {
		return nil, errcode.New(errcode.ForbiddenRole, "仅主任可新增事务")
	}
	date, err := time.ParseInLocation("2006-01-02", req.EventDate, time.Local)
	if err != nil {
		return nil, errcode.New(errcode.Params, "eventDate 需为 YYYY-MM-DD 格式")
	}
	start, startClock, err := normalizeClock(req.StartTime, false)
	if err != nil {
		return nil, err
	}
	end, endClock, err := normalizeClock(req.EndTime, true)
	if err != nil {
		return nil, err
	}
	if start < 6*60 {
		return nil, errcode.New(errcode.BizRule, "事务开始时间不得早于 06:00")
	}
	if end > 24*60 {
		return nil, errcode.New(errcode.BizRule, "事务结束时间不得晚于 24:00")
	}
	if start%30 != 0 || end%30 != 0 {
		return nil, errcode.New(errcode.BizRule, "事务时间需按 30 分钟颗粒选择")
	}
	if end <= start {
		return nil, errcode.New(errcode.BizRule, "结束时间必须晚于开始时间")
	}

	conflict, err := s.repo.FindOverlap(ctx, ownerID, date, startClock, endClock, 0)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "校验事务时间冲突失败", err)
	}
	if conflict != nil {
		return nil, errcode.New(errcode.Conflict, fmt.Sprintf(
			"与「%s %s-%s」时间冲突", scheduleEventName(*conflict), clockOf(conflict.StartTime), clockOf(conflict.EndTime),
		))
	}

	event := &model.ScheduleEvent{
		OwnerUserID: ownerID,
		EventDate:   date,
		StartTime:   startClock,
		EndTime:     endClock,
		EventType:   req.EventType,
		TypeText:    strings.TrimSpace(req.TypeText),
		Remark:      strings.TrimSpace(req.Remark),
	}
	if err := s.repo.Create(ctx, event); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "创建事务失败", err)
	}
	out := toScheduleEventDTO(*event)
	return &out, nil
}

func (s *scheduleEventService) UpdateRemark(ctx context.Context, role string, ownerID, id uint64, req dto.ScheduleEventRemarkReq) (*dto.ScheduleEventDTO, error) {
	if role != RoleDirector {
		return nil, errcode.New(errcode.ForbiddenRole, "仅主任可修改事务")
	}
	event, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "事务不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询事务失败", err)
	}
	if event.OwnerUserID != ownerID {
		return nil, errcode.New(errcode.ForbiddenData, "无权修改他人的事务")
	}
	if err := s.repo.UpdateRemark(ctx, id, strings.TrimSpace(req.Remark)); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "修改事务备注失败", err)
	}
	event.Remark = strings.TrimSpace(req.Remark)
	out := toScheduleEventDTO(*event)
	return &out, nil
}

func (s *scheduleEventService) Delete(ctx context.Context, role string, ownerID, id uint64) error {
	if role != RoleDirector {
		return errcode.New(errcode.ForbiddenRole, "仅主任可删除事务")
	}
	event, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return errcode.New(errcode.NotFound, "事务不存在")
		}
		return errcode.Wrap(errcode.Internal, "查询事务失败", err)
	}
	if event.OwnerUserID != ownerID {
		return errcode.New(errcode.ForbiddenData, "无权删除他人的事务")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errcode.Wrap(errcode.Internal, "删除事务失败", err)
	}
	return nil
}

// scheduleRange 解析查询区间；缺省为当前周（周一至周日）。
func scheduleRange(fromRaw, toRaw string) (time.Time, time.Time, error) {
	if fromRaw == "" && toRaw == "" {
		now := time.Now()
		offset := (int(now.Weekday()) + 6) % 7 // 周一=0
		monday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -offset)
		return monday, monday.AddDate(0, 0, 6), nil
	}
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, errcode.New(errcode.Params, "from 与 to 需成对传入")
	}
	from, err := time.ParseInLocation("2006-01-02", fromRaw, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, errcode.New(errcode.Params, "from 需为 YYYY-MM-DD 格式")
	}
	to, err := time.ParseInLocation("2006-01-02", toRaw, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, errcode.New(errcode.Params, "to 需为 YYYY-MM-DD 格式")
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, errcode.New(errcode.Params, "to 不得早于 from")
	}
	return from, to, nil
}

// normalizeClock 解析 HH:MM（允许 24:00 作为结束时间），返回分钟数与规范字符串。
func normalizeClock(raw string, allow24 bool) (int, string, error) {
	text := strings.TrimSpace(raw)
	parts := strings.Split(text, ":")
	if len(parts) != 2 {
		return 0, "", errcode.New(errcode.Params, "时间需为 HH:MM 格式")
	}
	hour, errH := strconv.Atoi(parts[0])
	minute, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil || minute < 0 || minute > 59 || hour < 0 || hour > 24 {
		return 0, "", errcode.New(errcode.Params, "时间需为 HH:MM 格式")
	}
	if hour == 24 {
		if !allow24 || minute != 0 {
			return 0, "", errcode.New(errcode.Params, "24:00 仅可作为结束时间")
		}
		return 24 * 60, "24:00", nil
	}
	return hour*60 + minute, fmt.Sprintf("%02d:%02d", hour, minute), nil
}

// clockOf 把数据库 TIME（可能带秒）归一为 HH:MM。
func clockOf(value string) string {
	parts := strings.Split(value, ":")
	if len(parts) >= 2 {
		hour, errH := strconv.Atoi(parts[0])
		minute, errM := strconv.Atoi(parts[1])
		if errH == nil && errM == nil {
			return fmt.Sprintf("%02d:%02d", hour, minute)
		}
	}
	return value
}

func scheduleEventName(e model.ScheduleEvent) string {
	if txt := strings.TrimSpace(e.TypeText); txt != "" {
		return txt
	}
	labels := map[string]string{
		model.ScheduleEventMeeting:   "会议",
		model.ScheduleEventOpenClass: "公开课",
		model.ScheduleEventLecture:   "讲座",
		model.ScheduleEventOther:     "其他事务",
	}
	if label, ok := labels[e.EventType]; ok {
		return label
	}
	return "事务"
}

func toScheduleEventDTO(e model.ScheduleEvent) dto.ScheduleEventDTO {
	return dto.ScheduleEventDTO{
		ID: e.ID, OwnerUserID: e.OwnerUserID,
		EventDate: e.EventDate.Format("2006-01-02"),
		StartTime: clockOf(e.StartTime), EndTime: clockOf(e.EndTime),
		EventType: e.EventType, TypeText: e.TypeText, Remark: e.Remark,
		CreatedAt: e.CreatedAt.Format(time.RFC3339), UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
	}
}
