package resp

import "rednote/internal/models"

// AuthorInfo 笔记/评论作者的精简信息
type AuthorInfo struct {
	Id       int64  `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// NoteResponse 笔记响应：模型字段 + 解析后的图片/标签 + 作者与互动状态
type NoteResponse struct {
	*models.Note
	Images    []string    `json:"images"`
	Tags      []string    `json:"tags"`
	Author    *AuthorInfo `json:"author"`
	Liked     bool        `json:"liked"`
	Favorited bool        `json:"favorited"`
}
