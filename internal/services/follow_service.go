package services

import (
	"time"

	"rednote/internal/models"
	"rednote/internal/models/constants"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/sqls"
	"rednote/internal/repositories"
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

func (s *followService) Follow(userId, followUserId int64) error {
	if userId == followUserId {
		return errs.ParamsError("不能关注自己")
	}
	if UserService.Get(followUserId) == nil {
		return errs.NotFound("用户不存在")
	}
	if s.GetBy(userId, followUserId) != nil {
		return nil
	}
	return repositories.UserFollowRepository.Create(sqls.DB(), &models.UserFollow{
		UserId:       userId,
		FollowUserId: followUserId,
		CreateTime:   time.Now().UnixMilli(),
	})
}

func (s *followService) Unfollow(userId, followUserId int64) error {
	follow := s.GetBy(userId, followUserId)
	if follow == nil {
		return nil
	}
	repositories.UserFollowRepository.Delete(sqls.DB(), follow.Id)
	return nil
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

// FollowingCount 关注数
func (s *followService) FollowingCount(userId int64) int64 {
	return repositories.UserFollowRepository.Count(sqls.DB(), sqls.NewCnd().Eq("user_id", userId))
}

// FansCount 粉丝数
func (s *followService) FansCount(userId int64) int64 {
	return repositories.UserFollowRepository.Count(sqls.DB(), sqls.NewCnd().Eq("follow_user_id", userId))
}

// NoteStats 用户的笔记数与获赞总数
func (s *followService) NoteCount(userId int64) int64 {
	return repositories.NoteRepository.Count(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId).Eq("status", constants.StatusOk))
}

func (s *followService) LikedCount(userId int64) int64 {
	return repositories.UserLikeRepository.Count(sqls.DB(), sqls.NewCnd().Eq("user_id", userId))
}
