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
}

type supervisionService struct {
	supervision repository.SupervisionRepository
}

// NewSupervisionService 构造督导服务。
func NewSupervisionService(supervision repository.SupervisionRepository) SupervisionService {
	return &supervisionService{supervision: supervision}
}

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
