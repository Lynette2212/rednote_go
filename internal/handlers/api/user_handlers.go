package api

import (
	"rednote/internal/models/req"
	"rednote/internal/models/resp"
	"rednote/internal/pkg/common"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/ginx"
	"rednote/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// UserProfile 用户公开主页信息
func UserProfile(ctx *gin.Context) {
	userId := cast.ToInt64(ctx.Param("id"))
	user := services.UserService.Get(userId)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotFound("用户不存在"))
		return
	}
	currentUser := common.GetCurrentUser(ctx)
	var currentUserId int64
	if currentUser != nil {
		currentUserId = currentUser.Id
	}
	ginx.WriteJSON(ctx, gin.H{
		"user":           resp.NewUserResponse(user),
		"noteCount":      services.FollowService.NoteCount(userId),
		"followingCount": services.FollowService.FollowingCount(userId),
		"fansCount":      services.FollowService.FansCount(userId),
		"likedCount":     services.FollowService.LikedCount(userId),
		"following":      services.FollowService.IsFollowing(currentUserId, userId),
	})
}

// UserUpdateProfile 修改资料
func UserUpdateProfile(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	var r req.UpdateProfileReq
	if err := ginx.Bind(ctx, &r); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	if err := services.UserService.UpdateProfile(user.Id, r.Nickname, r.Avatar, r.Bio, r.Gender); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, resp.NewUserResponse(services.UserService.Get(user.Id)))
}
