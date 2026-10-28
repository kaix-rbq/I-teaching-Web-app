package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/jwtutil"
)

// recordingRepoForMedia 复用 process 测试的 fake，仅覆盖录音读取。
type recordingRepoForMedia struct {
	fakeRecordingRepoForProcess
	rec *model.Recording
}

func (f *recordingRepoForMedia) GetRecordingBySession(context.Context, uint64) (*model.Recording, error) {
	return f.rec, nil
}

func (f *recordingRepoForMedia) GetRecording(context.Context, uint64) (*model.Recording, error) {
	return f.rec, nil
}

// TestTeacherCanPlayOwnCourseRecording 覆盖 D9：录音由督导上传时，授课教师本人
// 仍应拿到 playbackUrl 并可访问 stream；其他教师与主任保持 40302，督导仍全量可听。
func TestTeacherCanPlayOwnCourseRecording(t *testing.T) {
	const (
		sessionID    = uint64(7)
		recordingID  = uint64(21)
		teacherID    = uint64(2)
		supervisorID = uint64(5)
	)
	rec := &model.Recording{
		ID: recordingID, SessionID: sessionID, FilePath: "uploads/recordings/7/a.mp3",
		OriginalName: "a.mp3", Format: "mp3", UploadedBy: supervisorID,
	}
	repo := &recordingRepoForMedia{rec: rec}
	sessions := &fakeSessionRepo{
		getDetail: func(context.Context, uint64) (*repository.SessionDetailRow, error) {
			return sessionRow(), nil
		},
	}
	svc := NewRecordingService(repo, sessions, config.UploadConfig{}, config.TranscriptionConfig{},
		jwtutil.New("test-secret", time.Hour), nil, nil)
	ctx := context.Background()

	rd, _, err := svc.Media(ctx, RoleTeacher, 0, teacherID, sessionID)
	require.NoError(t, err)
	require.NotNil(t, rd)
	assert.NotEmpty(t, rd.PlaybackURL, "教师本人课程应获得播放票据")

	got, err := svc.Stream(ctx, RoleTeacher, 0, teacherID, recordingID)
	require.NoError(t, err)
	assert.Equal(t, recordingID, got.ID)

	_, err = svc.Stream(ctx, RoleTeacher, 0, 3, recordingID)
	require.Error(t, err)
	assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
	assert.Equal(t, "仅授课教师本人或督导可回听课堂音频", errcode.From(err).Msg)

	_, err = svc.Stream(ctx, RoleDirector, 0, 9, recordingID)
	require.Error(t, err, "主任不因本轮变更获得录音权限")
	assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)

	got, err = svc.Stream(ctx, RoleSupervisor, 0, supervisorID, recordingID)
	require.NoError(t, err)
	assert.Equal(t, recordingID, got.ID)
}
