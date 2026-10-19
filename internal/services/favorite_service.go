package services

import (
	"errors"
	"time"

	"rednote/internal/models"
	"rednote/internal/models/constants"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/sqls"
	"rednote/internal/repositories"

	"gorm.io/gorm"
)

var FavoriteService = newFavoriteService()

func newFavoriteService() *favoriteService {
	return &favoriteService{}
}

type favoriteService struct {
}

func (s *favoriteService) GetBy(userId, noteId int64) *models.Favorite {
	return repositories.FavoriteRepository.FindOne(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId).Eq("note_id", noteId))
}

// Favorite 收藏：与点赞同款事务+行锁写法
func (s *favoriteService) Favorite(userId, noteId int64) error {
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		note, err := repositories.NoteRepository.GetForUpdate(tx, noteId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.NotFound("笔记不存在")
			}
			return err
		}
		if note.Status == constants.StatusDeleted {
			return errs.NotFound("笔记不存在")
		}
		if repositories.FavoriteRepository.FindOne(tx,
			sqls.NewCnd().Eq("user_id", userId).Eq("note_id", noteId)) != nil {
			return nil // 已收藏，幂等
		}
		if err := repositories.FavoriteRepository.Create(tx, &models.Favorite{
			UserId:     userId,
			NoteId:     noteId,
			CreateTime: time.Now().UnixMilli(),
		}); err != nil {
			return err
		}
		return repositories.NoteRepository.Updates(tx, noteId,
			map[string]interface{}{"favorite_count": note.FavoriteCount + 1})
	})
	if err == nil {
		NoteService.InvalidateNoteCache(noteId)
	}
	return err
}

func (s *favoriteService) Unfavorite(userId, noteId int64) error {
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		note, err := repositories.NoteRepository.GetForUpdate(tx, noteId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		favorite := repositories.FavoriteRepository.FindOne(tx,
			sqls.NewCnd().Eq("user_id", userId).Eq("note_id", noteId))
		if favorite == nil {
			return nil // 未收藏，幂等
		}
		repositories.FavoriteRepository.Delete(tx, favorite.Id)
		count := note.FavoriteCount - 1
		if count < 0 {
			count = 0
		}
		return repositories.NoteRepository.Updates(tx, noteId,
			map[string]interface{}{"favorite_count": count})
	})
	if err == nil {
		NoteService.InvalidateNoteCache(noteId)
	}
	return err
}

func (s *favoriteService) GetFavoritedNoteIds(userId int64, noteIds []int64) map[int64]bool {
	ret := map[int64]bool{}
	if userId <= 0 || len(noteIds) == 0 {
		return ret
	}
	// FavoriteRepository 没有 Find，用 FindPageByCnd 大页查询代替
	list, _ := repositories.FavoriteRepository.FindPageByCnd(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId).In("note_id", noteIds).Page(1, len(noteIds)))
	for _, f := range list {
		ret[f.NoteId] = true
	}
	return ret
}
