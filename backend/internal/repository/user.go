package repository

import (
	"context"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// UserRepository 定义用户与教研室的数据访问。
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByID(ctx context.Context, id uint64) (*model.User, error)
	ListTeachers(ctx context.Context, departmentID uint64) ([]model.User, error)
	ListDepartments(ctx context.Context) ([]model.Department, error)
	GetDepartmentName(ctx context.Context, id uint64) (string, error)
	CountByDepartment(ctx context.Context, departmentID uint64) (int64, error)
	UpdatePasswordHash(ctx context.Context, usernames []string, hash string) (int64, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).Take(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) ListTeachers(ctx context.Context, departmentID uint64) ([]model.User, error) {
	var users []model.User
	tx := r.db.WithContext(ctx).
		Where("role = ?", model.RoleTeacher).
		Where("status = 1")
	if departmentID > 0 {
		tx = tx.Where("department_id = ?", departmentID)
	}
	if err := tx.Order("name ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) ListDepartments(ctx context.Context) ([]model.Department, error) {
	var depts []model.Department
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&depts).Error; err != nil {
		return nil, err
	}
	return depts, nil
}

func (r *userRepository) GetDepartmentName(ctx context.Context, id uint64) (string, error) {
	var dept model.Department
	if err := r.db.WithContext(ctx).Select("name").Where("id = ?", id).Take(&dept).Error; err != nil {
		return "", err
	}
	return dept.Name, nil
}

func (r *userRepository) CountByDepartment(ctx context.Context, departmentID uint64) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("department_id = ?", departmentID).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *userRepository) UpdatePasswordHash(ctx context.Context, usernames []string, hash string) (int64, error) {
	tx := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("username IN ?", usernames).
		Update("password_hash", hash)
	return tx.RowsAffected, tx.Error
}
