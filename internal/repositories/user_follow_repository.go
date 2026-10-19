package repositories

import (
	"rednote/internal/models"
	"rednote/internal/pkg/sqls"

	"gorm.io/gorm"
)

var UserFollowRepository = newUserFollowRepository()

func newUserFollowRepository() *userFollowRepository {
	return &userFollowRepository{}
}

type userFollowRepository struct {
}

func (r *userFollowRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.UserFollow) {
	cnd.Find(db, &list)
	return
}

func (r *userFollowRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.UserFollow {
	ret := &models.UserFollow{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *userFollowRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.UserFollow{})
}

func (r *userFollowRepository) Create(db *gorm.DB, t *models.UserFollow) error {
	return db.Create(t).Error
}

func (r *userFollowRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.UserFollow{}, "id = ?", id)
}
