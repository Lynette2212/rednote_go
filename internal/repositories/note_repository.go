package repositories

import (
	"rednote/internal/models"
	"rednote/internal/pkg/sqls"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var NoteRepository = newNoteRepository()

func newNoteRepository() *noteRepository {
	return &noteRepository{}
}

type noteRepository struct {
}

func (r *noteRepository) Get(db *gorm.DB, id int64) *models.Note {
	ret := &models.Note{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *noteRepository) Take(db *gorm.DB, where ...interface{}) *models.Note {
	ret := &models.Note{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *noteRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Note) {
	cnd.Find(db, &list)
	return
}

func (r *noteRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Note {
	ret := &models.Note{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *noteRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Note, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Note{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *noteRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.Note{})
}

func (r *noteRepository) Create(db *gorm.DB, t *models.Note) error {
	return db.Create(t).Error
}

func (r *noteRepository) Update(db *gorm.DB, t *models.Note) error {
	return db.Save(t).Error
}

func (r *noteRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) error {
	return db.Model(&models.Note{}).Where("id = ?", id).Updates(columns).Error
}

// GetForUpdate 在事务中锁定笔记行（SELECT ... FOR UPDATE）。
// 修改点赞/收藏/评论这类冗余计数前必须先取锁，否则并发请求会互相覆盖计数。
func (r *noteRepository) GetForUpdate(db *gorm.DB, id int64) (*models.Note, error) {
	ret := &models.Note{}
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(ret, "id = ?", id).Error
	return ret, err
}

func (r *noteRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.Note{}, "id = ?", id)
}
