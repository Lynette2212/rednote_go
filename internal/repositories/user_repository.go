package repositories

import (
	"rednote/internal/models"
	"rednote/internal/pkg/sqls"

	"gorm.io/gorm"
)

var UserRepository = newUserRepository()

func newUserRepository() *userRepository {
	return &userRepository{}
}

type userRepository struct {
}

func (r *userRepository) Get(db *gorm.DB, id int64) *models.User {
	ret := &models.User{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userRepository) Take(db *gorm.DB, where ...interface{}) *models.User {
	ret := &models.User{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.User) {
	cnd.Find(db, &list)
	return
}

func (r *userRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.User {
	ret := &models.User{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *userRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.User{})
}

func (r *userRepository) Create(db *gorm.DB, t *models.User) error {
	return db.Create(t).Error
}

func (r *userRepository) Update(db *gorm.DB, t *models.User) error {
	return db.Save(t).Error
}

func (r *userRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) error {
	return db.Model(&models.User{}).Where("id = ?", id).Updates(columns).Error
}
