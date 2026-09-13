package service

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/jwtutil"
)

// AuthService 负责登录鉴权与当前用户查询。
type AuthService interface {
	Login(ctx context.Context, req dto.LoginReq) (*dto.LoginResp, error)
	Me(ctx context.Context, userID uint64) (*dto.UserDTO, error)
}

type authService struct {
	users repository.UserRepository
	jwt   *jwtutil.Manager
	now   func() time.Time
}

// NewAuthService 构造认证服务。
func NewAuthService(users repository.UserRepository, jwt *jwtutil.Manager) AuthService {
	return &authService{users: users, jwt: jwt, now: time.Now}
}

// errBadCredentials 不区分「账号不存在」与「密码错误」，防账号枚举。
func errBadCredentials() error {
	return errcode.New(errcode.Unauthorized, "账号或密码错误")
}

func (s *authService) Login(ctx context.Context, req dto.LoginReq) (*dto.LoginResp, error) {
	user, err := s.users.GetByUsername(ctx, req.Username)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errBadCredentials()
		}
		return nil, errcode.Wrap(errcode.Internal, "查询用户失败", err)
	}
	if user.Status != 1 {
		return nil, errBadCredentials()
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errBadCredentials()
	}

	token, err := s.jwt.Sign(user.ID, user.Role, user.DeptID(), s.now())
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "签发令牌失败", err)
	}

	userDTO, err := s.toUserDTO(ctx, user)
	if err != nil {
		return nil, err
	}
	return &dto.LoginResp{Token: token, User: *userDTO}, nil
}

func (s *authService) Me(ctx context.Context, userID uint64) (*dto.UserDTO, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.Unauthorized, "登录状态已失效，请重新登录")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询用户失败", err)
	}
	return s.toUserDTO(ctx, user)
}

func (s *authService) toUserDTO(ctx context.Context, user *model.User) (*dto.UserDTO, error) {
	out := &dto.UserDTO{
		ID:           user.ID,
		Name:         user.Name,
		Role:         user.Role,
		JobNo:        user.JobNo,
		DepartmentID: user.DeptID(),
	}
	if deptID := user.DeptID(); deptID > 0 {
		name, err := s.users.GetDepartmentName(ctx, deptID)
		if err != nil && !repository.IsNotFound(err) {
			return nil, errcode.Wrap(errcode.Internal, "查询教研室失败", err)
		}
		out.Department = name
	}
	return out, nil
}
