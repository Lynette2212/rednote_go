package api

import (
	"rednote/internal/models/req"
	"rednote/internal/pkg/common"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/ginx"
	"rednote/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// CommentCreate 发表评论/回复
func CommentCreate(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	var r req.CommentCreateReq
	if err := ginx.Bind(ctx, &r); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	comment, err := services.CommentService.Create(user.Id, r.NoteId, r.ParentId, r.ReplyToUserId, r.Content)
	if err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, comment)
}

// CommentList 笔记的评论列表
func CommentList(ctx *gin.Context) {
	page, limit := getPageParams(ctx)
	noteId := cast.ToInt64(ctx.Query("noteId"))
	comments, paging := services.CommentService.ListByNote(noteId, page, limit)
	ginx.WriteJSON(ctx, ginx.PageData(comments, paging))
}

// CommentDelete 删除评论
func CommentDelete(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	commentId := cast.ToInt64(ctx.Param("id"))
	if err := services.CommentService.Delete(user.Id, commentId); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, nil)
}
