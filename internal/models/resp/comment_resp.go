package resp

import "rednote/internal/models"

// CommentResponse 评论响应：评论模型 + 作者/被回复人 + 子评论
type CommentResponse struct {
	*models.Comment
	User     *AuthorInfo        `json:"user"`
	ReplyTo  *AuthorInfo        `json:"replyTo"`
	Children []*CommentResponse `json:"children"`
}
