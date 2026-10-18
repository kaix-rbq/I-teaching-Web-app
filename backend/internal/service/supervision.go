package service

import (
	"context"
	"time"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// SupervisionService 负责督导覆盖率与听评课安排。
type SupervisionService interface {
	Coverage(ctx context.Context) (*dto.CoverageDTO, error)
	Plans(ctx context.Context, q dto.PlanListQuery) (*dto.PageResult[dto.PlanItem], error)
	// CommentTags 返回常用语标签库（N2，纯静态，供督导评语点选）。
	CommentTags() []dto.CommentTagDTO
}

type supervisionService struct {
	supervision repository.SupervisionRepository
}

// NewSupervisionService 构造督导服务。
func NewSupervisionService(supervision repository.SupervisionRepository) SupervisionService {
	return &supervisionService{supervision: supervision}
}

// commentTagLibrary 是常用语标签库（N2）：与前端评语输入框的可点选标签同源，纯静态。
var commentTagLibrary = []dto.CommentTagDTO{
	{Category: "亮点", Items: []string{
		"目标明确，重难点突出", "案例贴近实战", "环节清晰，节奏得当",
		"提问开放并留有思考时间", "学生参与度高，讨论充分",
	}},
	{Category: "主要问题", Items: []string{
		"提问后等待时间不足", "互动以自问自答为主", "内容以照本宣科为主",
		"环节过渡生硬", "时间分配失衡，节奏偏快",
	}},
	{Category: "学生参与", Items: []string{
		"出现学生主动提问", "小组讨论参与面广", "学生回答内容有深度",
		"学生话语痕迹较少",
	}},
	{Category: "改进建议", Items: []string{
		"每讲插入一道课堂小测", "为讨论设置明确时限", "压缩讲授、增加练习",
		"引入近两年行业案例", "给基础薄弱学生留缓冲",
	}},
	{Category: "复查要求", Items: []string{
		"请下次课重点复查互动环节", "请补充随堂练习设计", "请复查时间分配改进情况",
	}},
}

// CommentTags 返回常用语标签库（N2）。
func (s *supervisionService) CommentTags() []dto.CommentTagDTO { return commentTagLibrary }

// Coverage 计算当前学期开课课程的督导覆盖率（总体 + 分教研室）。
func (s *supervisionService) Coverage(ctx context.Context) (*dto.CoverageDTO, error) {
	total, supervised, err := s.supervision.Coverage(ctx, CurrentSemester)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计督导覆盖率失败", err)
	}

	rows, err := s.supervision.CoverageByDepartment(ctx, CurrentSemester)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "统计分教研室覆盖率失败", err)
	}

	byDept := make([]dto.DeptRate, 0, len(rows))
	for _, row := range rows {
		rate := 0.0
		if row.Total > 0 {
			rate = round2(float64(row.Supervised) / float64(row.Total))
		}
		byDept = append(byDept, dto.DeptRate{Department: row.Department, Rate: rate})
	}

	overall := 0.0
	if total > 0 {
		overall = round2(float64(supervised) / float64(total))
	}
	return &dto.CoverageDTO{
		TotalCourses:      int(total),
		SupervisedCourses: int(supervised),
		Rate:              overall,
		ByDepartment:      byDept,
	}, nil
}

// Plans 返回听评课安排分页列表。
func (s *supervisionService) Plans(ctx context.Context, q dto.PlanListQuery) (*dto.PageResult[dto.PlanItem], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	if q.DateFrom != "" {
		if _, err := time.Parse("2006-01-02", q.DateFrom); err != nil {
			return nil, errcode.New(errcode.Params, "dateFrom 需为 YYYY-MM-DD 格式")
		}
	}
	if q.DateTo != "" {
		if _, err := time.Parse("2006-01-02", q.DateTo); err != nil {
			return nil, errcode.New(errcode.Params, "dateTo 需为 YYYY-MM-DD 格式")
		}
	}

	rows, total, err := s.supervision.ListPlans(ctx, repository.PlanListParams{
		Status:   q.Status,
		DateFrom: q.DateFrom,
		DateTo:   q.DateTo,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询听评课安排失败", err)
	}
	return dto.NewPageResult(toPlanItems(rows), total, page, pageSize), nil
}
