package service

import (
	"context"
	"encoding/json"
	"fmt"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// DemoService 负责演示模式一键载入示例转写/AI 报告（B6/N7）。
// 仅本地/dev 环境开启（config.demo.enabled），生产禁用；调用前由前端二次确认。
type DemoService interface {
	Seed(ctx context.Context) (*dto.DemoSeedResp, error)
}

type demoService struct {
	repo    repository.DemoRepository
	enabled bool
}

// NewDemoService 构造演示模式服务。
func NewDemoService(repo repository.DemoRepository, enabled bool) DemoService {
	return &demoService{repo: repo, enabled: enabled}
}

// demoSession 是演示讲义目录（session_id → 主题），对应种子 14 次授课。
type demoSession struct {
	ID    uint64
	Topic string
}

var demoSessions = []demoSession{
	{1, "项目立项与章程"}, {2, "需求调研与用户故事"}, {3, "迭代计划与估点"},
	{4, "进程与线程"}, {5, "死锁与调度"}, {6, "内存管理"},
	{7, "关系模型与范式"}, {8, "SQL 与查询优化"},
	{9, "软件生命周期"}, {10, "需求工程"},
	{11, "监督学习概览"}, {12, "模型评估指标"},
	{13, "指令系统"}, {14, "存储层次"},
}

// Seed 载入示例转写/AI 报告与提优闭环示例数据（幂等：先清空再写入）。
func (s *demoService) Seed(ctx context.Context) (*dto.DemoSeedResp, error) {
	if !s.enabled {
		return nil, errcode.New(errcode.ForbiddenRole, "演示模式未开启（仅本地/dev 可用）")
	}
	data := buildDemoData()
	counts, err := s.repo.Seed(ctx, data)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "载入演示数据失败", err)
	}
	return &dto.DemoSeedResp{
		Recordings:    counts.Recordings,
		Transcripts:   counts.Transcripts,
		Improvements:  counts.Improvements,
		Reflections:   counts.Reflections,
		Notifications: counts.Notifications,
		Message:       "演示数据已载入",
	}, nil
}

// buildDemoData 组装演示数据集：14 条示例转写 + 提优闭环示例。
func buildDemoData() repository.DemoData {
	const demoUploader = uint64(5) // 陈静（supervisor）
	data := repository.DemoData{}

	for _, ds := range demoSessions {
		content := fmt.Sprintf(
			"（演示脱敏转写）本次课主题：%s。教师：我们先回顾上节要点，再进入今天的核心内容。教师：请大家思考一个问题——这一部分的关键是什么？学生：关键是要先明确边界条件。教师：回答得很好，我们再看一个近两年的行业案例……",
			ds.Topic,
		)
		segments, _ := json.Marshal([]dto.TranscriptSegment{
			{Start: 0, End: 12.5, Speaker: "教师", Text: fmt.Sprintf("本次课主题：%s，先回顾上节要点。", ds.Topic)},
			{Start: 12.5, End: 40.0, Speaker: "教师", Text: "请大家思考一个问题——这一部分的关键是什么？"},
			{Start: 40.0, End: 70.0, Speaker: "学生", Text: "关键是要先明确边界条件。"},
			{Start: 70.0, End: 120.0, Speaker: "教师", Text: "回答得很好，我们再看一个近两年的行业案例。"},
		})
		data.Recordings = append(data.Recordings, repository.DemoRecording{
			SessionID: ds.ID, UploadedBy: demoUploader,
			FilePath:     fmt.Sprintf("uploads/recordings/%d/demo.mp3", ds.ID),
			OriginalName: fmt.Sprintf("demo-session-%d.mp3", ds.ID), Format: "mp3",
			Size: 1_000_000, DurationSec: 2700,
		})
		data.Transcripts = append(data.Transcripts, repository.DemoTranscript{
			SessionID: ds.ID, Content: content, Segments: string(segments),
		})
	}

	// 提优闭环示例（教师 2 = 李明）。
	data.ImprovementItems = []model.ImprovementItem{
		{TeacherID: 2, SessionID: 1, Title: "提问后延长等待时间至 3-5 秒", Source: model.ImprovementSourceAI, Status: model.ImprovementStatusOpen},
		{TeacherID: 2, SessionID: 3, Title: "为小组讨论设置明确时限", Source: model.ImprovementSourceSupervisor, Status: model.ImprovementStatusDoneSelf, DoneNote: "已在课堂中试行"},
	}
	data.Reflections = []model.TeachingReflection{
		{TeacherID: 2, SessionID: 1, Content: "本次课互动偏少，提问后立即给出了答案。", Plan: "下节课延长等待时间，并点名邀请学生表达。", NextAction: "准备 2 个开放性问题"},
	}
	payload, _ := json.Marshal(map[string]string{"title": "收到新的督导评语"})
	p := string(payload)
	data.Notifications = []model.UserNotification{
		{UserID: 2, Type: model.NotificationTypeSupervisorComment, RefID: 1, Payload: &p},
	}
	return data
}
