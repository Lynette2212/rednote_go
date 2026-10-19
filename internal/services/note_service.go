package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"path"
	"strings"
	"time"

	"rednote/internal/models"
	"rednote/internal/models/constants"
	"rednote/internal/models/resp"
	"rednote/internal/pkg/cache"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/es"
	"rednote/internal/pkg/sqls"
	"rednote/internal/repositories"

	"golang.org/x/sync/singleflight"
)

var NoteService = newNoteService()

func newNoteService() *noteService {
	return &noteService{}
}

type noteService struct {
}

func (s *noteService) Get(id int64) *models.Note {
	return repositories.NoteRepository.Get(sqls.DB(), id)
}

const noteCacheTTL = 5 * time.Minute

var noteCacheGroup singleflight.Group

// GetCached 笔记详情缓存（cache-aside）：命中直接返回；未命中回源数据库并写回缓存。
// singleflight 防止热点 key 过期瞬间的并发穿透。
func (s *noteService) GetCached(ctx context.Context, id int64) *models.Note {
	key := cache.NoteKey(id)
	if note := readNoteCache(ctx, key); note != nil {
		return note
	}

	value, _, _ := noteCacheGroup.Do(key, func() (interface{}, error) {
		if note := readNoteCache(ctx, key); note != nil {
			return note, nil
		}
		note := s.Get(id)
		if note != nil && note.Status != constants.StatusDeleted {
			if raw, err := json.Marshal(note); err == nil {
				cache.Set(ctx, key, string(raw), noteCacheTTL)
			}
		}
		return note, nil
	})
	note, _ := value.(*models.Note)
	return note
}

func readNoteCache(ctx context.Context, key string) *models.Note {
	if raw, ok := cache.Get(ctx, key); ok {
		note := &models.Note{}
		if err := json.Unmarshal([]byte(raw), note); err == nil {
			return note
		}
	}
	return nil
}

// InvalidateNoteCache 计数/状态变化后删除缓存，下次读取自动回源
func (s *noteService) InvalidateNoteCache(id int64) {
	cache.Del(context.Background(), cache.NoteKey(id))
}

func (s *noteService) Publish(userId int64, title, content string, images, tags []string, videoUrl string) (*models.Note, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	videoUrl = strings.TrimSpace(videoUrl)

	mediaType := "image"
	if videoUrl != "" {
		if !validVideoPath(videoUrl) {
			return nil, errs.ParamsError("视频地址不合法")
		}
		mediaType = "video"
	} else if len(images) == 0 {
		return nil, errs.ParamsError("请至少上传一张图片或一个视频")
	}
	if len(images) > 9 {
		return nil, errs.ParamsError("最多上传 9 张图片")
	}
	for _, image := range images {
		if !validUploadPath(image) {
			return nil, errs.ParamsError("图片地址不合法")
		}
	}
	if title == "" {
		return nil, errs.ParamsError("标题不能为空")
	}
	if len([]rune(title)) > 64 {
		return nil, errs.ParamsError("标题最多 64 字")
	}
	if len([]rune(content)) > 5000 {
		return nil, errs.ParamsError("正文最多 5000 字")
	}

	imagesJson, _ := json.Marshal(images)
	tagsJson, _ := json.Marshal(tags)
	now := time.Now().UnixMilli()
	note := &models.Note{
		UserId:     userId,
		Title:      title,
		Content:    content,
		Images:     string(imagesJson),
		Tags:       string(tagsJson),
		MediaType:  mediaType,
		VideoUrl:   videoUrl,
		CreateTime: now,
		UpdateTime: now,
	}
	if err := repositories.NoteRepository.Create(sqls.DB(), note); err != nil {
		return nil, err
	}
	cache.DelPattern(context.Background(), "rednote:feed:*")
	s.indexNote(note)
	return note, nil
}

func (s *noteService) Delete(userId, noteId int64) error {
	note := s.Get(noteId)
	if note == nil || note.Status == constants.StatusDeleted {
		return errs.NotFound("笔记不存在")
	}
	if note.UserId != userId {
		return errs.NoPermission()
	}
	if err := repositories.NoteRepository.Updates(sqls.DB(), noteId, map[string]interface{}{
		"status":      constants.StatusDeleted,
		"update_time": time.Now().UnixMilli(),
	}); err != nil {
		return err
	}
	s.InvalidateNoteCache(noteId)
	cache.DelPattern(context.Background(), "rednote:feed:*")
	s.deleteNoteIndex(noteId)
	return nil
}

// ===== 搜索索引：发布/删除时同步 ES；失败只记日志，不影响主流程 =====

// validUploadPath 只允许本服务上传目录或 Web 内置演示目录下的图片
func validUploadPath(p string) bool {
	if !(strings.HasPrefix(p, "/uploads/") || strings.HasPrefix(p, "/videos/")) ||
		strings.Contains(p, "..") || strings.ContainsAny(p, "\\\x00") {
		return false
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

// validVideoPath 允许上传目录和 Web 内置演示视频，阻止路径穿越与外部 URL
func validVideoPath(p string) bool {
	if !(strings.HasPrefix(p, "/uploads/videos/") || strings.HasPrefix(p, "/videos/")) {
		return false
	}
	if strings.Contains(p, "..") || strings.ContainsAny(p, "\\\x00") {
		return false
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".mp4", ".webm":
		return true
	default:
		return false
	}
}

func (s *noteService) noteDoc(note *models.Note) es.NoteDoc {
	var tags []string
	_ = json.Unmarshal([]byte(note.Tags), &tags)
	if tags == nil {
		tags = []string{}
	}
	return es.NoteDoc{
		Id:         note.Id,
		UserId:     note.UserId,
		Title:      note.Title,
		Content:    note.Content,
		Tags:       tags,
		CreateTime: note.CreateTime,
		Status:     note.Status,
	}
}

func (s *noteService) indexNote(note *models.Note) {
	if !es.Enabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := es.IndexNote(ctx, s.noteDoc(note)); err != nil {
		slog.Warn("写入搜索索引失败", "noteId", note.Id, "err", err)
	}
}

func (s *noteService) deleteNoteIndex(noteId int64) {
	if !es.Enabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := es.DeleteNote(ctx, noteId); err != nil {
		slog.Warn("删除搜索索引失败", "noteId", noteId, "err", err)
	}
}

// BackfillSearchIndex 启动时调用：索引为空则把库里存量笔记全量灌入 ES，
// 否则老笔记搜不到（索引只覆盖它建立之后的变化）。
func (s *noteService) BackfillSearchIndex() {
	if !es.Enabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := es.EnsureIndex(ctx); err != nil {
		slog.Warn("初始化搜索索引失败", "err", err)
		return
	}
	total, err := es.CountDocs(ctx)
	if err != nil {
		slog.Warn("检查搜索索引失败", "err", err)
		return
	}
	if total > 0 {
		return
	}
	notes := repositories.NoteRepository.Find(sqls.DB(),
		sqls.NewCnd().Eq("status", constants.StatusOk))
	for i := range notes {
		s.indexNote(&notes[i])
	}
	slog.Info("搜索索引回填完成", "count", len(notes))
}

// Feed 推荐流（最新优先，Redis 缓存 30 秒）
func (s *noteService) Feed(page, limit int) ([]models.Note, *sqls.Paging) {
	ctx := context.Background()
	key := cache.FeedKey(page, limit)
	if raw, ok := cache.Get(ctx, key); ok {
		var cached struct {
			Notes  []models.Note `json:"notes"`
			Paging *sqls.Paging  `json:"paging"`
		}
		if json.Unmarshal([]byte(raw), &cached) == nil && cached.Paging != nil {
			return cached.Notes, cached.Paging
		}
	}

	cnd := sqls.NewCnd().Eq("status", constants.StatusOk).Desc("id").Page(page, limit)
	notes, paging := repositories.NoteRepository.FindPageByCnd(sqls.DB(), cnd)
	if payload, err := json.Marshal(struct {
		Notes  []models.Note `json:"notes"`
		Paging *sqls.Paging  `json:"paging"`
	}{Notes: notes, Paging: paging}); err == nil {
		cache.Set(ctx, key, string(payload), 30*time.Second)
	}
	return notes, paging
}

// FollowFeed 关注流
func (s *noteService) FollowFeed(userId int64, page, limit int) ([]models.Note, *sqls.Paging) {
	followIds := FollowService.GetFollowingIds(userId)
	cnd := sqls.NewCnd().Eq("status", constants.StatusOk).Desc("id").Page(page, limit)
	if len(followIds) == 0 {
		return []models.Note{}, &sqls.Paging{Page: page, Limit: limit, Total: 0}
	}
	cnd.In("user_id", followIds)
	return repositories.NoteRepository.FindPageByCnd(sqls.DB(), cnd)
}

// UserNotes 某用户的笔记
func (s *noteService) UserNotes(userId int64, page, limit int) ([]models.Note, *sqls.Paging) {
	cnd := sqls.NewCnd().Eq("status", constants.StatusOk).Eq("user_id", userId).Desc("id").Page(page, limit)
	return repositories.NoteRepository.FindPageByCnd(sqls.DB(), cnd)
}

// UserLikedNotes 某用户点赞过的
func (s *noteService) UserLikedNotes(userId int64, page, limit int) ([]models.Note, *sqls.Paging) {
	likes, paging := repositories.UserLikeRepository.FindPageByCnd(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId).Desc("id").Page(page, limit))
	return s.notesByLikesOrFavorites(likes, paging)
}

// UserFavoriteNotes 某用户收藏的
func (s *noteService) UserFavoriteNotes(userId int64, page, limit int) ([]models.Note, *sqls.Paging) {
	favorites, paging := repositories.FavoriteRepository.FindPageByCnd(sqls.DB(),
		sqls.NewCnd().Eq("user_id", userId).Desc("id").Page(page, limit))
	ids := make([]int64, 0, len(favorites))
	for _, f := range favorites {
		ids = append(ids, f.NoteId)
	}
	return s.notesByIds(ids, paging)
}

func (s *noteService) notesByLikesOrFavorites(likes []models.UserLike, paging *sqls.Paging) ([]models.Note, *sqls.Paging) {
	ids := make([]int64, 0, len(likes))
	for _, l := range likes {
		ids = append(ids, l.NoteId)
	}
	return s.notesByIds(ids, paging)
}

func (s *noteService) notesByIds(ids []int64, paging *sqls.Paging) ([]models.Note, *sqls.Paging) {
	if len(ids) == 0 {
		return []models.Note{}, paging
	}
	list := repositories.NoteRepository.Find(sqls.DB(),
		sqls.NewCnd().Eq("status", constants.StatusOk).In("id", ids))
	byId := make(map[int64]models.Note, len(list))
	for _, note := range list {
		byId[note.Id] = note
	}
	ordered := make([]models.Note, 0, len(list))
	for _, id := range ids {
		if note, ok := byId[id]; ok {
			ordered = append(ordered, note)
		}
	}
	return ordered, paging
}

// Search 搜索：优先走 Elasticsearch（相关性排序）；ES 未启用或查询出错时降级数据库模糊匹配
func (s *noteService) Search(keyword string, page, limit int) ([]models.Note, *sqls.Paging) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return repositories.NoteRepository.FindPageByCnd(sqls.DB(),
			sqls.NewCnd().Eq("status", constants.StatusOk).Desc("id").Page(page, limit))
	}
	if es.Enabled() {
		if notes, paging := s.searchByES(keyword, page, limit); notes != nil {
			return notes, paging
		}
	}
	// 降级：数据库模糊匹配（后端换 ES 前的原始实现）
	kw := "%" + escapeLike(keyword) + "%"
	cnd := sqls.NewCnd().Eq("status", constants.StatusOk).Desc("id").Page(page, limit).
		Where("title ILIKE ? OR content ILIKE ? OR tags ILIKE ?", kw, kw, kw)
	return repositories.NoteRepository.FindPageByCnd(sqls.DB(), cnd)
}

// escapeLike 转义 ILIKE 通配符，避免用户输入 % / _ 被当作模式
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// searchByES 查 ES 拿 id 列表（按相关性排序），再回 PostgreSQL 取详情；出错返回 nil 触发降级
func (s *noteService) searchByES(keyword string, page, limit int) ([]models.Note, *sqls.Paging) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ids, total, err := es.SearchNoteIds(ctx, keyword, (page-1)*limit, limit)
	if err != nil {
		slog.Warn("ES 搜索失败，降级为数据库模糊匹配", "err", err)
		return nil, nil
	}
	paging := &sqls.Paging{Page: page, Limit: limit, Total: total}
	if len(ids) == 0 {
		return []models.Note{}, paging
	}
	list := repositories.NoteRepository.Find(sqls.DB(),
		sqls.NewCnd().Eq("status", constants.StatusOk).In("id", ids))
	byId := make(map[int64]models.Note, len(list))
	for _, n := range list {
		byId[n.Id] = n
	}
	// 按 ES 的相关性顺序重排（SQL 的 IN 查询不保证顺序）
	ordered := make([]models.Note, 0, len(ids))
	for _, id := range ids {
		if n, ok := byId[id]; ok {
			ordered = append(ordered, n)
		}
	}
	return ordered, paging
}

// BuildResponses 组装笔记响应：解析图片/标签、填充作者、当前用户点赞收藏状态
func (s *noteService) BuildResponses(notes []models.Note, currentUserId int64) []*resp.NoteResponse {
	userIds := make([]int64, 0, len(notes))
	noteIds := make([]int64, 0, len(notes))
	seen := map[int64]bool{}
	for _, n := range notes {
		noteIds = append(noteIds, n.Id)
		if !seen[n.UserId] {
			seen[n.UserId] = true
			userIds = append(userIds, n.UserId)
		}
	}
	users := UserService.GetUsersByIds(userIds)
	likedIds := LikeService.GetLikedNoteIds(currentUserId, noteIds)
	favoritedIds := FavoriteService.GetFavoritedNoteIds(currentUserId, noteIds)

	ret := make([]*resp.NoteResponse, 0, len(notes))
	for i := range notes {
		n := &notes[i]
		var images, tags []string
		json.Unmarshal([]byte(n.Images), &images)
		json.Unmarshal([]byte(n.Tags), &tags)
		if images == nil {
			images = []string{}
		}
		if tags == nil {
			tags = []string{}
		}
		item := &resp.NoteResponse{
			Note:      n,
			Images:    images,
			Tags:      tags,
			Liked:     likedIds[n.Id],
			Favorited: favoritedIds[n.Id],
		}
		if u := users[n.UserId]; u != nil {
			item.Author = &resp.AuthorInfo{Id: u.Id, Nickname: u.Nickname, Avatar: u.Avatar}
		}
		ret = append(ret, item)
	}
	return ret
}

func (s *noteService) BuildResponse(note *models.Note, currentUserId int64) *resp.NoteResponse {
	list := s.BuildResponses([]models.Note{*note}, currentUserId)
	if len(list) == 0 {
		return nil
	}
	return list[0]
}
