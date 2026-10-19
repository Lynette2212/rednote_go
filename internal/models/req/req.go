package req

type RegisterReq struct {
	Username string `json:"username" binding:"required"`
	Nickname string `json:"nickname"`
	Password string `json:"password" binding:"required"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateProfileReq struct {
	Nickname string `json:"nickname" binding:"required"`
	Avatar   string `json:"avatar"`
	Bio      string `json:"bio"`
	Gender   int    `json:"gender"`
}

type NoteCreateReq struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content"`
	Images  []string `json:"images" binding:"required"`
	Tags    []string `json:"tags"`
}

type CommentCreateReq struct {
	NoteId        int64  `json:"noteId" binding:"required"`
	ParentId      int64  `json:"parentId"`
	ReplyToUserId int64  `json:"replyToUserId"`
	Content       string `json:"content" binding:"required"`
}

type EntityActionReq struct {
	NoteId int64 `json:"noteId" binding:"required"`
}

type FollowReq struct {
	UserId int64 `json:"userId" binding:"required"`
}
