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

var LikeService = newLikeService()

func newLikeService() *likeService {
	return &likeService{}
}

type likeService struct {
}

func (s *likeService) GetBy(userId, noteId int64) *models.UserLike {
	return repositories.UserLikeRepository.FindOne(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId).Eq("note_id", noteId))
}

// Like 点赞：锁定笔记行后写入关系表并更新计数，保证并发下计数与关系表一致
func (s *likeService) Like(userId, noteId int64) error {
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
		if repositories.UserLikeRepository.FindOne(tx,
			sqls.NewCnd().Eq("user_id", userId).Eq("note_id", noteId)) != nil {
			return nil // 已点赞，幂等
		}
		if err := repositories.UserLikeRepository.Create(tx, &models.UserLike{
			UserId:     userId,
			NoteId:     noteId,
			CreateTime: time.Now().UnixMilli(),
		}); err != nil {
			return err
		}
		return repositories.NoteRepository.Updates(tx, noteId,
			map[string]interface{}{"like_count": note.LikeCount + 1})
	})
	if err == nil {
		NoteService.InvalidateNoteCache(noteId) // 计数变了，缓存作废
	}
	return err
}

func (s *likeService) Unlike(userId, noteId int64) error {
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		note, err := repositories.NoteRepository.GetForUpdate(tx, noteId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil // 笔记已不存在，无需处理
			}
			return err
		}
		like := repositories.UserLikeRepository.FindOne(tx,
			sqls.NewCnd().Eq("user_id", userId).Eq("note_id", noteId))
		if like == nil {
			return nil // 未点赞，幂等
		}
		repositories.UserLikeRepository.Delete(tx, like.Id)
		count := note.LikeCount - 1
		if count < 0 {
			count = 0
		}
		return repositories.NoteRepository.Updates(tx, noteId,
			map[string]interface{}{"like_count": count})
	})
	if err == nil {
		NoteService.InvalidateNoteCache(noteId)
	}
	return err
}

// GetLikedNoteIds 返回 noteIds 中 userId 已点赞的集合
func (s *likeService) GetLikedNoteIds(userId int64, noteIds []int64) map[int64]bool {
	ret := map[int64]bool{}
	if userId <= 0 || len(noteIds) == 0 {
		return ret
	}
	list := repositories.UserLikeRepository.Find(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId).In("note_id", noteIds))
	for _, l := range list {
		ret[l.NoteId] = true
	}
	return ret
}
