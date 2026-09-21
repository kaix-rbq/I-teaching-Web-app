package repository

import (
	"context"
	"time"

	"aijiaoxue-api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RecordingRepository interface {
	CreateRecording(context.Context, *model.Recording) error
	GetRecordingBySession(context.Context, uint64) (*model.Recording, error)
	GetRecording(context.Context, uint64) (*model.Recording, error)
	CreateTranscript(context.Context, *model.Transcript) error
	GetTranscriptBySession(context.Context, uint64) (*model.Transcript, error)
	UpdateTranscript(context.Context, uint64, map[string]any) error
	UpsertAgentEvaluation(context.Context, *model.Evaluation) error
	LogPlayback(context.Context, uint64, uint64, time.Time) error
}

type recordingRepository struct{ db *gorm.DB }

func NewRecordingRepository(db *gorm.DB) RecordingRepository { return &recordingRepository{db: db} }

func (r *recordingRepository) CreateRecording(ctx context.Context, v *model.Recording) error {
	return r.db.WithContext(ctx).Create(v).Error
}
func (r *recordingRepository) GetRecordingBySession(ctx context.Context, id uint64) (*model.Recording, error) {
	var v model.Recording
	err := r.db.WithContext(ctx).Where("session_id = ?", id).Take(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func (r *recordingRepository) GetRecording(ctx context.Context, id uint64) (*model.Recording, error) {
	var v model.Recording
	err := r.db.WithContext(ctx).First(&v, id).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func (r *recordingRepository) CreateTranscript(ctx context.Context, v *model.Transcript) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "session_id"}}, DoUpdates: clause.AssignmentColumns([]string{"recording_id", "content", "segments", "engine", "engine_version", "status", "error_message"})}).Create(v).Error
}
func (r *recordingRepository) GetTranscriptBySession(ctx context.Context, id uint64) (*model.Transcript, error) {
	var v model.Transcript
	err := r.db.WithContext(ctx).Where("session_id = ?", id).Take(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func (r *recordingRepository) UpdateTranscript(ctx context.Context, id uint64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.Transcript{}).Where("id = ?", id).Updates(fields).Error
}
func (r *recordingRepository) UpsertAgentEvaluation(ctx context.Context, v *model.Evaluation) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "session_id"}, {Name: "evaluator_type"}, {Name: "evaluator_id"}}, DoUpdates: clause.AssignmentColumns([]string{"ai_model_version", "formula_version", "objective_score", "content_score", "interaction_score", "organization_score", "frontier_score", "total_score", "ai_confidence", "evidence"})}).Create(v).Error
}
func (r *recordingRepository) LogPlayback(ctx context.Context, recordingID, userID uint64, at time.Time) error {
	return r.db.WithContext(ctx).Table("recording_playback_logs").Create(map[string]any{"recording_id": recordingID, "user_id": userID, "started_at": at}).Error
}
