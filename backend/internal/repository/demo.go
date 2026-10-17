package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// DemoTranscript 是演示用转写（已脱敏文本 + 分段 JSON）。
type DemoTranscript struct {
	SessionID uint64
	Content   string
	Segments  string
}

// DemoRecording 是演示用录音占位（文件路径仅为展示，不落真实音频）。
type DemoRecording struct {
	SessionID    uint64
	UploadedBy   uint64
	FilePath     string
	OriginalName string
	Format       string
	Size         int64
	DurationSec  int
}

// DemoData 是一次演示载入的全部数据。
type DemoData struct {
	Recordings       []DemoRecording
	Transcripts      []DemoTranscript
	ImprovementItems []model.ImprovementItem
	Reflections      []model.TeachingReflection
	Notifications    []model.UserNotification
}

// DemoCounts 是载入结果计数。
type DemoCounts struct {
	Recordings    int
	Transcripts   int
	Improvements  int
	Reflections   int
	Notifications int
}

// DemoRepository 负责演示模式的一键载入（B6/N7）。仅 dev 环境使用。
type DemoRepository interface {
	Seed(ctx context.Context, data DemoData) (DemoCounts, error)
}

type demoRepository struct{ db *gorm.DB }

// NewDemoRepository 构造演示数据仓储。
func NewDemoRepository(db *gorm.DB) DemoRepository { return &demoRepository{db: db} }

// Seed 在单事务内清空示例相关表并写回演示数据集。
//
// 仅本地/dev：会清空 transcripts / recordings / recording_playback_logs /
// improvement_items / teaching_reflections / user_notifications（不触碰用户、课程、评价）。
func (r *demoRepository) Seed(ctx context.Context, data DemoData) (DemoCounts, error) {
	var counts DemoCounts
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, t := range []string{
			"recording_playback_logs", "transcripts", "recordings",
			"improvement_items", "teaching_reflections", "user_notifications",
		} {
			if err := tx.Exec("DELETE FROM " + t).Error; err != nil {
				return err
			}
		}

		recIDBySession := make(map[uint64]uint64, len(data.Recordings))
		for _, rec := range data.Recordings {
			record := &model.Recording{
				SessionID: rec.SessionID, FilePath: rec.FilePath, OriginalName: rec.OriginalName,
				Format: rec.Format, Size: rec.Size, DurationSec: rec.DurationSec, UploadedBy: rec.UploadedBy,
				UploadedAt: time.Now(),
			}
			if err := tx.Create(record).Error; err != nil {
				return err
			}
			recIDBySession[rec.SessionID] = record.ID
			counts.Recordings++
		}
		for _, tr := range data.Transcripts {
			seg := tr.Segments
			if err := tx.Create(&model.Transcript{
				SessionID: tr.SessionID, RecordingID: recIDBySession[tr.SessionID], Content: tr.Content, Segments: &seg,
				Engine: "demo", EngineVersion: "demo-v1", Status: model.TranscriptDone,
			}).Error; err != nil {
				return err
			}
			counts.Transcripts++
		}
		for i := range data.ImprovementItems {
			if err := tx.Create(&data.ImprovementItems[i]).Error; err != nil {
				return err
			}
			counts.Improvements++
		}
		for i := range data.Reflections {
			if err := tx.Create(&data.Reflections[i]).Error; err != nil {
				return err
			}
			counts.Reflections++
		}
		for i := range data.Notifications {
			if err := tx.Create(&data.Notifications[i]).Error; err != nil {
				return err
			}
			counts.Notifications++
		}
		return nil
	})
	return counts, err
}
