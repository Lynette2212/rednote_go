package services

import (
	"log/slog"
	"time"

	"rednote/internal/models"
	"rednote/internal/models/constants"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/sqls"
	"rednote/internal/repositories"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var FollowService = newFollowService()

func newFollowService() *followService {
	return &followService{}
}

type followService struct {
}

func (s *followService) GetBy(userId, followUserId int64) *models.UserFollow {
	return repositories.UserFollowRepository.FindOne(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId).Eq("follow_user_id", followUserId))
}

func (s *followService) IsFollowing(userId, followUserId int64) bool {
	return userId > 0 && s.GetBy(userId, followUserId) != nil
}

// Follow 关注：关系写入与双方计数在同一事务内完成；重复关注幂等返回成功
func (s *followService) Follow(userId, followUserId int64) error {
	if userId == followUserId {
		return errs.ParamsError("不能关注自己")
	}
	if UserService.Get(followUserId) == nil {
		return errs.NotFound("用户不存在")
	}
	return sqls.DB().Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.UserFollow{
			UserId:       userId,
			FollowUserId: followUserId,
			CreateTime:   time.Now().UnixMilli(),
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := tx.Model(&models.User{}).Where("id = ?", userId).
			UpdateColumn("following_count", gorm.Expr("following_count + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", followUserId).
			UpdateColumn("fans_count", gorm.Expr("fans_count + 1")).Error
	})
}

// Unfollow 取关：删除关系与双方计数在同一事务内完成；未关注时幂等返回成功
func (s *followService) Unfollow(userId, followUserId int64) error {
	return sqls.DB().Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ? AND follow_user_id = ?", userId, followUserId).
			Delete(&models.UserFollow{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := tx.Model(&models.User{}).Where("id = ?", userId).
			UpdateColumn("following_count", gorm.Expr("GREATEST(following_count - 1, 0)")).Error; err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", followUserId).
			UpdateColumn("fans_count", gorm.Expr("GREATEST(fans_count - 1, 0)")).Error
	})
}

// BackfillCounts 启动时按关系表校准冗余计数（一次性修复历史数据）
func (s *followService) BackfillCounts() {
	err := sqls.DB().Exec(`
		UPDATE users u SET
			following_count = (SELECT COUNT(*) FROM user_follows f WHERE f.user_id = u.id),
			fans_count = (SELECT COUNT(*) FROM user_follows f WHERE f.follow_user_id = u.id)
	`).Error
	if err != nil {
		slog.Warn("回填关注计数失败", "err", err)
	}
}

func (s *followService) GetFollowingIds(userId int64) []int64 {
	list := repositories.UserFollowRepository.Find(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId))
	ids := make([]int64, 0, len(list))
	for _, f := range list {
		ids = append(ids, f.FollowUserId)
	}
	return ids
}

// FollowingCount 关注数（读取冗余字段）
func (s *followService) FollowingCount(userId int64) int64 {
	user := UserService.Get(userId)
	if user == nil {
		return 0
	}
	return user.FollowingCount
}

// FansCount 粉丝数（读取冗余字段）
func (s *followService) FansCount(userId int64) int64 {
	user := UserService.Get(userId)
	if user == nil {
		return 0
	}
	return user.FansCount
}

// NoteStats 用户的笔记数与获赞总数
func (s *followService) NoteCount(userId int64) int64 {
	return repositories.NoteRepository.Count(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId).Eq("status", constants.StatusOk))
}

func (s *followService) LikedCount(userId int64) int64 {
	return repositories.UserLikeRepository.Count(sqls.DB(), sqls.NewCnd().Eq("user_id", userId))
}
