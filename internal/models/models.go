package models

var Models = []interface{}{
	&User{},
	&Note{}, &Comment{},
	&UserLike{}, &Favorite{}, &UserFollow{},
}

type Model struct {
	Id int64 `gorm:"primaryKey;autoIncrement" json:"id" form:"id"`
}

const (
	StatusOk      = 0 // 正常
	StatusDeleted = 1 // 删除
)

type User struct {
	Model
	Username   string `gorm:"uniqueIndex;size:32;not null" json:"username"`
	Nickname   string `gorm:"size:32" json:"nickname"`
	Avatar     string `gorm:"size:256" json:"avatar"`
	Bio        string `gorm:"size:512" json:"bio"`
	Gender     int    `gorm:"default:0" json:"gender"` // 0保密 1男 2女
	Password   string `gorm:"size:128" json:"-"`
	Status     int    `gorm:"default:0" json:"status"`
	CreateTime int64  `gorm:"not null;default:0" json:"createTime"`
	UpdateTime int64  `gorm:"not null;default:0" json:"updateTime"`
}

type Note struct {
	Model
	UserId        int64  `gorm:"index;not null" json:"userId"`
	Title         string `gorm:"size:128" json:"title"`
	Content       string `gorm:"type:text" json:"content"`
	Images        string `gorm:"type:text" json:"images"` // JSON 数组
	Tags          string `gorm:"type:text" json:"tags"`   // JSON 数组
	LikeCount     int64  `gorm:"default:0" json:"likeCount"`
	FavoriteCount int64  `gorm:"default:0" json:"favoriteCount"`
	CommentCount  int64  `gorm:"default:0" json:"commentCount"`
	Status        int    `gorm:"default:0;index" json:"status"`
	CreateTime    int64  `gorm:"not null;default:0;index" json:"createTime"`
	UpdateTime    int64  `gorm:"not null;default:0" json:"updateTime"`
}

type Comment struct {
	Model
	NoteId         int64  `gorm:"index;not null" json:"noteId"`
	UserId         int64  `gorm:"index;not null" json:"userId"`
	ParentId       int64  `gorm:"default:0;index" json:"parentId"`     // 父评论 id，0 为一级评论
	ReplyToUserId  int64  `gorm:"default:0" json:"replyToUserId"`      // 被回复的用户
	Content        string `gorm:"type:text" json:"content"`
	Status         int    `gorm:"default:0" json:"status"`
	CreateTime     int64  `gorm:"not null;default:0" json:"createTime"`
}

type UserLike struct {
	Model
	UserId     int64 `gorm:"uniqueIndex:idx_user_like;not null" json:"userId"`
	NoteId     int64 `gorm:"uniqueIndex:idx_user_like;not null;index" json:"noteId"`
	CreateTime int64 `gorm:"not null;default:0" json:"createTime"`
}

type Favorite struct {
	Model
	UserId     int64 `gorm:"uniqueIndex:idx_user_favorite;not null" json:"userId"`
	NoteId     int64 `gorm:"uniqueIndex:idx_user_favorite;not null;index" json:"noteId"`
	CreateTime int64 `gorm:"not null;default:0" json:"createTime"`
}

type UserFollow struct {
	Model
	UserId       int64 `gorm:"uniqueIndex:idx_user_follow;not null" json:"userId"`
	FollowUserId int64 `gorm:"uniqueIndex:idx_user_follow;not null;index" json:"followUserId"`
	CreateTime   int64 `gorm:"not null;default:0" json:"createTime"`
}
