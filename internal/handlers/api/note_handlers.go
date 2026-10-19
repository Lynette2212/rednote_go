package api

import (
	"rednote/internal/models"
	"rednote/internal/models/constants"
	"rednote/internal/models/req"
	"rednote/internal/pkg/common"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/ginx"
	"rednote/internal/pkg/sqls"
	"rednote/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func getPageParams(ctx *gin.Context) (int, int) {
	page := cast.ToInt(ctx.DefaultQuery("page", "1"))
	limit := cast.ToInt(ctx.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return page, limit
}

func currentUserId(ctx *gin.Context) int64 {
	if user := common.GetCurrentUser(ctx); user != nil {
		return user.Id
	}
	return 0
}

func writeNotePage(ctx *gin.Context, notes []models.Note, paging *sqls.Paging) {
	ginx.WriteJSON(ctx, ginx.PageData(services.NoteService.BuildResponses(notes, currentUserId(ctx)), paging))
}

// NoteCreate 发布笔记
func NoteCreate(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	var r req.NoteCreateReq
	if err := ginx.Bind(ctx, &r); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	note, err := services.NoteService.Publish(user.Id, r.Title, r.Content, r.Images, r.Tags)
	if err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, services.NoteService.BuildResponse(note, user.Id))
}

// NoteDetail 笔记详情
func NoteDetail(ctx *gin.Context) {
	noteId := cast.ToInt64(ctx.Param("id"))
	note := services.NoteService.GetCached(ctx.Request.Context(), noteId)
	if note == nil || note.Status == constants.StatusDeleted {
		ginx.WriteJSON(ctx, errs.NotFound("笔记不存在"))
		return
	}
	ginx.WriteJSON(ctx, services.NoteService.BuildResponse(note, currentUserId(ctx)))
}

// NoteFeed 推荐流
func NoteFeed(ctx *gin.Context) {
	page, limit := getPageParams(ctx)
	notes, paging := services.NoteService.Feed(page, limit)
	writeNotePage(ctx, notes, paging)
}

// NoteFollowFeed 关注流
func NoteFollowFeed(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	page, limit := getPageParams(ctx)
	notes, paging := services.NoteService.FollowFeed(user.Id, page, limit)
	writeNotePage(ctx, notes, paging)
}

// NoteUserNotes 某用户的笔记
func NoteUserNotes(ctx *gin.Context) {
	page, limit := getPageParams(ctx)
	userId := cast.ToInt64(ctx.Query("userId"))
	notes, paging := services.NoteService.UserNotes(userId, page, limit)
	writeNotePage(ctx, notes, paging)
}

// NoteUserLikes 某用户点赞的笔记
func NoteUserLikes(ctx *gin.Context) {
	page, limit := getPageParams(ctx)
	userId := cast.ToInt64(ctx.Query("userId"))
	notes, paging := services.NoteService.UserLikedNotes(userId, page, limit)
	writeNotePage(ctx, notes, paging)
}

// NoteUserFavorites 某用户收藏的笔记
func NoteUserFavorites(ctx *gin.Context) {
	page, limit := getPageParams(ctx)
	userId := cast.ToInt64(ctx.Query("userId"))
	notes, paging := services.NoteService.UserFavoriteNotes(userId, page, limit)
	writeNotePage(ctx, notes, paging)
}

// NoteSearch 搜索
func NoteSearch(ctx *gin.Context) {
	page, limit := getPageParams(ctx)
	notes, paging := services.NoteService.Search(ctx.Query("q"), page, limit)
	writeNotePage(ctx, notes, paging)
}

// NoteDelete 删除笔记
func NoteDelete(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	noteId := cast.ToInt64(ctx.Param("id"))
	if err := services.NoteService.Delete(user.Id, noteId); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, nil)
}
