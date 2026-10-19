package services

import (
	"errors"
	"strings"
	"time"

	"rednote/internal/models"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/sqls"
	"rednote/internal/repositories"

	"gorm.io/gorm"
)

var CommentService = newCommentService()

func newCommentService() *commentService {
	return &commentService{}
}

type commentService struct {
}

type CommentResponse struct {
	*models.Comment
	User     *AuthorInfo        `json:"user"`
	ReplyTo  *AuthorInfo        `json:"replyTo"`
	Children []*CommentResponse `json:"children"`
}

func (s *commentService) Create(userId, noteId, parentId, replyToUserId int64, content string) (*models.Comment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errs.ParamsError("评论内容不能为空")
	}
	if len([]rune(content)) > 500 {
		return nil, errs.ParamsError("评论最多 500 字")
	}
	comment := &models.Comment{
		NoteId:        noteId,
		UserId:        userId,
		ParentId:      parentId,
		ReplyToUserId: replyToUserId,
		Content:       content,
		CreateTime:    time.Now().UnixMilli(),
	}
	// 插入评论与更新计数在同一事务内完成，先锁笔记行保证并发下计数准确
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		note, err := repositories.NoteRepository.GetForUpdate(tx, noteId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.NotFound("笔记不存在")
			}
			return err
		}
		if note.Status == models.StatusDeleted {
			return errs.NotFound("笔记不存在")
		}
		if parentId > 0 {
			parent := repositories.CommentRepository.Get(tx, parentId)
			if parent == nil || parent.NoteId != noteId || parent.Status == models.StatusDeleted {
				return errs.NotFound("父评论不存在")
			}
			// 统一挂到一级评论下，最多两级
			if parent.ParentId > 0 {
				comment.ParentId = parent.ParentId
			}
		}
		if err := repositories.CommentRepository.Create(tx, comment); err != nil {
			return err
		}
		return repositories.NoteRepository.Updates(tx, noteId,
			map[string]interface{}{"comment_count": note.CommentCount + 1})
	})
	if err != nil {
		return nil, err
	}
	NoteService.InvalidateNoteCache(noteId)
	return comment, nil
}

func (s *commentService) Delete(userId, commentId int64) error {
	var noteId int64
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		comment := repositories.CommentRepository.Get(tx, commentId)
		if comment == nil || comment.Status == models.StatusDeleted {
			return errs.NotFound("评论不存在")
		}
		if comment.UserId != userId {
			return errs.NoPermission()
		}
		noteId = comment.NoteId
		note, err := repositories.NoteRepository.GetForUpdate(tx, comment.NoteId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.NotFound("笔记不存在")
			}
			return err
		}
		if err := repositories.CommentRepository.Updates(tx, commentId, map[string]interface{}{
			"status": models.StatusDeleted,
		}); err != nil {
			return err
		}
		count := note.CommentCount - 1
		if count < 0 {
			count = 0
		}
		return repositories.NoteRepository.Updates(tx, comment.NoteId,
			map[string]interface{}{"comment_count": count})
	})
	if err == nil && noteId > 0 {
		NoteService.InvalidateNoteCache(noteId)
	}
	return err
}

// ListByNote 一级评论分页，每个一级评论携带其全部回复
func (s *commentService) ListByNote(noteId int64, page, limit int) ([]*CommentResponse, *sqls.Paging) {
	roots, paging := repositories.CommentRepository.FindPageByCnd(sqls.DB(),
		sqls.NewCnd().Eq("note_id", noteId).Eq("parent_id", 0).
			Eq("status", models.StatusOk).Desc("id").Page(page, limit))

	rootIds := make([]int64, 0, len(roots))
	for _, c := range roots {
		rootIds = append(rootIds, c.Id)
	}
	var children []models.Comment
	if len(rootIds) > 0 {
		children = repositories.CommentRepository.Find(sqls.DB(),
			sqls.NewCnd().In("parent_id", rootIds).Eq("status", models.StatusOk).Asc("id"))
	}

	// 收集涉及的用户 id
	userIdSet := map[int64]bool{}
	userIds := []int64{}
	addUser := func(id int64) {
		if id > 0 && !userIdSet[id] {
			userIdSet[id] = true
			userIds = append(userIds, id)
		}
	}
	for _, c := range roots {
		addUser(c.UserId)
	}
	for _, c := range children {
		addUser(c.UserId)
		addUser(c.ReplyToUserId)
	}
	users := UserService.GetUsersByIds(userIds)
	authorOf := func(id int64) *AuthorInfo {
		if u := users[id]; u != nil {
			return &AuthorInfo{Id: u.Id, Nickname: u.Nickname, Avatar: u.Avatar}
		}
		return nil
	}

	childrenMap := map[int64][]*CommentResponse{}
	for i := range children {
		c := &children[i]
		childrenMap[c.ParentId] = append(childrenMap[c.ParentId], &CommentResponse{
			Comment: c,
			User:    authorOf(c.UserId),
			ReplyTo: authorOf(c.ReplyToUserId),
		})
	}
	ret := make([]*CommentResponse, 0, len(roots))
	for i := range roots {
		c := &roots[i]
		resp := &CommentResponse{
			Comment:  c,
			User:     authorOf(c.UserId),
			Children: childrenMap[c.Id],
		}
		if resp.Children == nil {
			resp.Children = []*CommentResponse{}
		}
		ret = append(ret, resp)
	}
	return ret, paging
}
