package repositories

import (
	"rednote/internal/models"
	"rednote/internal/pkg/sqls"

	"gorm.io/gorm"
)

var FavoriteRepository = newFavoriteRepository()

func newFavoriteRepository() *favoriteRepository {
	return &favoriteRepository{}
}

type favoriteRepository struct {
}

func (r *favoriteRepository) Get(db *gorm.DB, id int64) *models.Favorite {
	ret := &models.Favorite{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *favoriteRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Favorite {
	ret := &models.Favorite{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *favoriteRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Favorite) {
	cnd.Find(db, &list)
	return
}

func (r *favoriteRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Favorite, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Favorite{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *favoriteRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.Favorite{})
}

func (r *favoriteRepository) Create(db *gorm.DB, t *models.Favorite) error {
	return db.Create(t).Error
}

func (r *favoriteRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.Favorite{}, "id = ?", id)
}
