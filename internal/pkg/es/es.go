package es

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
)

// 说明：Elasticsearch 对本项目是"搜索加速层"——未配置或连接失败时保持禁用，
// 搜索自动降级为数据库模糊匹配（ILIKE），系统功能不受影响。

const defaultIndex = "rednote_notes"

var (
	client    *elasticsearch.Client
	indexName = defaultIndex
)

// Init 连接 Elasticsearch。addr 为空表示未配置（禁用）；连接失败返回错误由调用方决定。
// index 为空时使用默认索引名。
func Init(addr, index string) error {
	if addr == "" {
		return nil
	}
	if index != "" {
		indexName = index
	}
	c, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{addr}})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := c.Ping(c.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("ES Ping 异常: %s", res.Status())
	}
	client = c
	return nil
}

// DeleteIndex 删除索引（测试清场用），不存在时忽略
func DeleteIndex(ctx context.Context) error {
	if client == nil {
		return nil
	}
	res, err := client.Indices.Delete([]string{indexName}, client.Indices.Delete.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("删除索引失败: %s", res.String())
	}
	return nil
}

func Enabled() bool {
	return client != nil
}

// NoteDoc 笔记的搜索文档：只放检索需要的字段，展示数据回 PostgreSQL 取
type NoteDoc struct {
	Id         int64    `json:"id"`
	UserId     int64    `json:"userId"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	CreateTime int64    `json:"createTime"`
	Status     int      `json:"status"`
}

// EnsureIndex 索引不存在时创建（含字段映射）
func EnsureIndex(ctx context.Context) error {
	if client == nil {
		return nil
	}
	res, err := client.Indices.Exists([]string{indexName}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode == 200 {
		return nil
	}
	if res.StatusCode != 404 {
		return fmt.Errorf("检查索引异常: %s", res.Status())
	}
	mapping := `{
		"settings": {"number_of_shards": 1, "number_of_replicas": 0},
		"mappings": {"properties": {
			"title":      {"type": "text"},
			"content":    {"type": "text"},
			"tags":       {"type": "text"},
			"userId":     {"type": "long"},
			"createTime": {"type": "long"},
			"status":     {"type": "integer"}
		}}
	}`
	res, err = client.Indices.Create(indexName,
		client.Indices.Create.WithContext(ctx),
		client.Indices.Create.WithBody(strings.NewReader(mapping)))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("创建索引失败: %s", res.String())
	}
	return nil
}

func IndexNote(ctx context.Context, doc NoteDoc) error {
	if client == nil {
		return nil
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	// refresh=wait_for：等这条文档对搜索可见后再返回（ES 默认近实时，约 1 秒延迟；
	// 这里用少量写入延迟换"发布完立刻能搜到"的确定性）
	res, err := client.Index(indexName, bytes.NewReader(body),
		client.Index.WithDocumentID(strconv.FormatInt(doc.Id, 10)),
		client.Index.WithRefresh("wait_for"),
		client.Index.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("写入索引失败: %s", res.String())
	}
	return nil
}

func DeleteNote(ctx context.Context, id int64) error {
	if client == nil {
		return nil
	}
	res, err := client.Delete(indexName, strconv.FormatInt(id, 10),
		client.Delete.WithRefresh("wait_for"),
		client.Delete.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// 删除不存在的文档不算错误（幂等）
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("删除索引失败: %s", res.String())
	}
	return nil
}

// CountDocs 索引里的文档总数（用于启动时判断是否需要回填）
func CountDocs(ctx context.Context) (int64, error) {
	if client == nil {
		return 0, errors.New("ES 未启用")
	}
	res, err := client.Count(
		client.Count.WithContext(ctx),
		client.Count.WithIndex(indexName))
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	var ret struct {
		Count int64 `json:"count"`
	}
	if err := json.Unmarshal(body, &ret); err != nil {
		return 0, err
	}
	return ret.Count, nil
}

// SearchNoteIds 全文检索，按相关性（title 权重更高）返回笔记 id，平局按发布时间新→旧。
// 中文无需额外插件：ES 默认分词器对汉字按单字切分，天然支持中文模糊匹配。
func SearchNoteIds(ctx context.Context, keyword string, from, size int) ([]int64, int64, error) {
	if client == nil {
		return nil, 0, errors.New("ES 未启用")
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{"multi_match": map[string]interface{}{
						"query":  keyword,
						"fields": []string{"title^2", "content", "tags"},
					}},
				},
				"filter": []interface{}{
					map[string]interface{}{"term": map[string]interface{}{"status": 0}},
				},
			},
		},
		"sort": []interface{}{
			"_score",
			map[string]interface{}{"createTime": map[string]interface{}{"order": "desc"}},
		},
		"from":             from,
		"size":             size,
		"track_total_hits": true,
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, 0, err
	}
	res, err := client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex(indexName),
		client.Search.WithBody(bytes.NewReader(body)))
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, 0, fmt.Errorf("ES 查询失败: %s", res.String())
	}
	var ret struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID string `json:"_id"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(ret.Hits.Hits))
	for _, h := range ret.Hits.Hits {
		if id, err := strconv.ParseInt(h.ID, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, ret.Hits.Total.Value, nil
}
