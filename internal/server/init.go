package server

import (
	"fmt"
	"log/slog"
	"time"

	"rednote/internal/models"
	"rednote/internal/pkg/cache"
	"rednote/internal/pkg/config"
	"rednote/internal/pkg/es"
	"rednote/internal/pkg/sqls"
	"rednote/internal/services"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Init() error {
	if err := config.Init(); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	if err := initDB(); err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	services.FollowService.BackfillCounts()
	initRedis()
	initElasticsearch()
	return nil
}

// initElasticsearch ES 是搜索加速层：连不上只警告，搜索降级为数据库模糊匹配
func initElasticsearch() {
	conf := config.Instance.Elasticsearch
	if conf.Addr == "" {
		slog.Info("未配置 Elasticsearch，搜索使用数据库模糊匹配")
		return
	}
	if err := es.Init(conf.Addr, conf.Index); err != nil {
		slog.Warn("Elasticsearch 连接失败，搜索将降级数据库模糊匹配", "addr", conf.Addr, "err", err)
		return
	}
	slog.Info("Elasticsearch 连接成功", "addr", conf.Addr)
	services.NoteService.BackfillSearchIndex()
}

// initRedis Redis 是加速层而非必需品：连不上只警告，缓存退化直查、限流关闭
func initRedis() {
	conf := config.Instance.Redis
	if conf.Addr == "" {
		slog.Info("未配置 Redis，缓存与限流已禁用")
		return
	}
	if err := cache.Init(conf.Addr, conf.Password, conf.DB); err != nil {
		slog.Warn("Redis 连接失败，缓存与限流已禁用", "addr", conf.Addr, "err", err)
		return
	}
	slog.Info("Redis 连接成功", "addr", conf.Addr)
}

func initDB() error {
	conf := config.Instance.DB

	logLevel := gormlogger.Warn
	switch conf.LogLevel {
	case "silent":
		logLevel = gormlogger.Silent
	case "error":
		logLevel = gormlogger.Error
	case "info":
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(conf.Url), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if conf.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(conf.MaxIdleConns)
	}
	if conf.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(conf.MaxOpenConns)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := db.AutoMigrate(models.Models...); err != nil {
		return err
	}
	sqls.Init(db)
	slog.Info("数据库连接成功，自动迁移完成")
	return nil
}
