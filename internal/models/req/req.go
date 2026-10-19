package req

type RegisterReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Nickname string `json:"nickname" binding:"max=32"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required,max=32"`
	Password string `json:"password" binding:"required,max=64"`
}

type UpdateProfileReq struct {
	Nickname string `json:"nickname" binding:"required,max=32"`
	Avatar   string `json:"avatar" binding:"max=256"`
	Bio      string `json:"bio" binding:"max=512"`
	Gender   int    `json:"gender"`
}

type NoteCreateReq struct {
	Title    string   `json:"title" binding:"required,max=64"`
	Content  string   `json:"content" binding:"max=5000"`
	Images   []string `json:"images" binding:"max=9,dive,required"`
	Tags     []string `json:"tags" binding:"max=10,dive,max=20"`
	VideoUrl string   `json:"videoUrl" binding:"max=512"`
}

type CommentCreateReq struct {
	NoteId        int64  `json:"noteId" binding:"required"`
	ParentId      int64  `json:"parentId"`
	ReplyToUserId int64  `json:"replyToUserId"`
	Content       string `json:"content" binding:"required,max=500"`
}

type EntityActionReq struct {
	NoteId int64 `json:"noteId" binding:"required"`
}

type FollowReq struct {
	UserId int64 `json:"userId" binding:"required"`
}
