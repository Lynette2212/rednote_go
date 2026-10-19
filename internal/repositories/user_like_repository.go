package repositories

import (
	"rednote/internal/models"
	"rednote/internal/pkg/sqls"

	"gorm.io/gorm"
)

var UserLikeRepository = newUserLikeRepository()

func newUserLikeRepository() *userLikeRepository {
	return &userLikeRepository{}
}

type userLikeRepository struct {
}

func (r *userLikeRepository) Get(db *gorm.DB, id int64) *models.UserLike {
	ret := &models.UserLike{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userLikeRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.UserLike) {
	cnd.Find(db, &list)
	return
}

func (r *userLikeRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.UserLike {
	ret := &models.UserLike{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *userLikeRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.UserLike, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.UserLike{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *userLikeRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.UserLike{})
}

func (r *userLikeRepository) Create(db *gorm.DB, t *models.UserLike) error {
	return db.Create(t).Error
}

func (r *userLikeRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.UserLike{}, "id = ?", id)
}
