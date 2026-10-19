package resp

import "rednote/internal/models"

// UserResponse 对外暴露的用户字段；不包含密码、状态等内部字段
type UserResponse struct {
	Id             int64  `json:"id"`
	Username       string `json:"username"`
	Nickname       string `json:"nickname"`
	Avatar         string `json:"avatar"`
	Bio            string `json:"bio"`
	Gender         int    `json:"gender"`
	FollowingCount int64  `json:"followingCount"`
	FansCount      int64  `json:"fansCount"`
	CreateTime     int64  `json:"createTime"`
	UpdateTime     int64  `json:"updateTime"`
}

func NewUserResponse(user *models.User) *UserResponse {
	if user == nil {
		return nil
	}
	return &UserResponse{
		Id:             user.Id,
		Username:       user.Username,
		Nickname:       user.Nickname,
		Avatar:         user.Avatar,
		Bio:            user.Bio,
		Gender:         user.Gender,
		FollowingCount: user.FollowingCount,
		FansCount:      user.FansCount,
		CreateTime:     user.CreateTime,
		UpdateTime:     user.UpdateTime,
	}
}
