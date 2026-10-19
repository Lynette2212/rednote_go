package cache

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// 说明：Redis 对本项目是"加速层"而不是必需品——连接失败时保持禁用，
// 缓存读取自动回源数据库，限流自动放行，系统功能不受影响。

var rdb *redis.Client

// Init 连接 Redis。addr 为空表示未配置（禁用）；连接失败返回错误，由调用方决定如何处理。
func Init(addr, password string, db int) error {
	if addr == "" {
		return nil
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return err
	}
	rdb = client
	return nil
}

func Enabled() bool {
	return rdb != nil
}

func NoteKey(id int64) string {
	return "rednote:note:" + strconv.FormatInt(id, 10)
}

func FeedKey(page, limit int) string {
	return "rednote:feed:" + strconv.Itoa(page) + ":" + strconv.Itoa(limit)
}

// DelPattern 按模式删除缓存（用于发布/删除笔记后失效信息流）
func DelPattern(ctx context.Context, pattern string) {
	if rdb == nil {
		return
	}
	var keys []string
	iter := rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 100 {
			rdb.Del(ctx, keys...)
			keys = keys[:0]
		}
	}
	if len(keys) > 0 {
		rdb.Del(ctx, keys...)
	}
}

func Get(ctx context.Context, key string) (string, bool) {
	if rdb == nil {
		return "", false
	}
	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

func Set(ctx context.Context, key, value string, ttl time.Duration) {
	if rdb == nil {
		return
	}
	rdb.Set(ctx, key, value, ttl)
}

func Del(ctx context.Context, keys ...string) {
	if rdb == nil || len(keys) == 0 {
		return
	}
	rdb.Del(ctx, keys...)
}

// Incr 限流用：对 key 原子自增，首次创建时设置过期（固定窗口计数）
func Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if rdb == nil {
		return 0, errors.New("redis 未启用")
	}
	n, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		rdb.Expire(ctx, key, ttl)
	}
	return n, nil
}
