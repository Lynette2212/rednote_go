package config

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	Instance *Config
	v        *viper.Viper
	writeMx  sync.Mutex
)

func init() {
	v = viper.New()
	v.SetConfigFile("rednote.yaml")
	v.AddConfigPath(".")
	if workDir, err := os.Executable(); err == nil {
		v.AddConfigPath(filepath.Dir(workDir))
	}
	v.SetDefault("port", 8082)
	v.SetDefault("uploadDir", "./data/uploads")
	v.SetDefault("db.url", "")
	v.SetDefault("jwtSecret", "")
	v.SetDefault("db.maxIdleConns", 10)
	v.SetDefault("db.maxOpenConns", 25)
	v.SetDefault("db.logLevel", "warn")
	v.SetDefault("redis.addr", "")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("rateLimit.authPerMin", 20)
	v.SetDefault("rateLimit.apiPerMin", 300)
	v.SetDefault("elasticsearch.addr", "")
	v.SetDefault("elasticsearch.index", "rednote_notes")
	v.AutomaticEnv()
	v.SetEnvPrefix("REDNOTE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// 常用环境变量显式绑定，方便容器部署（无 rednote.yaml 时也能启动）
	_ = v.BindEnv("port", "REDNOTE_PORT")
	_ = v.BindEnv("db.url", "REDNOTE_DB_URL")
	_ = v.BindEnv("uploadDir", "REDNOTE_UPLOADDIR", "REDNOTE_UPLOAD_DIR")
	_ = v.BindEnv("jwtSecret", "REDNOTE_JWTSECRET", "REDNOTE_JWT_SECRET")
	_ = v.BindEnv("redis.addr", "REDNOTE_REDIS_ADDR")
	_ = v.BindEnv("redis.password", "REDNOTE_REDIS_PASSWORD")
	_ = v.BindEnv("redis.db", "REDNOTE_REDIS_DB")
	_ = v.BindEnv("elasticsearch.addr", "REDNOTE_ES_ADDR")
	_ = v.BindEnv("elasticsearch.index", "REDNOTE_ES_INDEX")
}

type Config struct {
	Port           int           `yaml:"port"`
	AllowedOrigins []string      `yaml:"allowedOrigins"`
	JwtSecret      string        `yaml:"jwtSecret"`
	UploadDir      string        `yaml:"uploadDir"`
	Logger         Logger        `yaml:"logger"`
	DB             DB            `yaml:"db"`
	Redis          Redis         `yaml:"redis"`
	RateLimit      RateLimit     `yaml:"rateLimit"`
	Elasticsearch  Elasticsearch `yaml:"elasticsearch"`
}

// Elasticsearch 全文检索。addr 为空则禁用（搜索自动降级为数据库模糊匹配）
type Elasticsearch struct {
	Addr  string `yaml:"addr"`
	Index string `yaml:"index"` // 索引名，默认 rednote_notes（测试用独立索引避免污染）
}

// Redis 缓存/限流。addr 为空则整体禁用（系统照常运行，缓存降级为直查数据库）
type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// RateLimit 每 IP 每分钟的接口上限，0 表示用默认值
type RateLimit struct {
	AuthPerMin int `yaml:"authPerMin"` // 登录/注册
	ApiPerMin  int `yaml:"apiPerMin"`  // 其余接口
}

type Logger struct {
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"maxSize"`
	MaxAge     int    `yaml:"maxAge"`
	MaxBackups int    `yaml:"maxBackups"`
}

type DB struct {
	Url          string `yaml:"url"`
	MaxIdleConns int    `yaml:"maxIdleConns"`
	MaxOpenConns int    `yaml:"maxOpenConns"`
	LogLevel     string `yaml:"logLevel"`
}

func Init() error {
	// 配置文件缺失不致命：容器里可以只用环境变量 + 默认值
	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		slog.Warn("未找到 rednote.yaml，改用环境变量与默认值")
	}
	conf := &Config{}
	if err := v.Unmarshal(conf); err != nil {
		return err
	}
	if conf.Port == 0 {
		conf.Port = 8082
	}
	if conf.UploadDir == "" {
		conf.UploadDir = "./data/uploads"
	}
	if conf.JwtSecret == "" {
		return errors.New("缺少 jwtSecret：请在 rednote.yaml 配置或设置 REDNOTE_JWT_SECRET")
	}
	if conf.RateLimit.AuthPerMin == 0 {
		conf.RateLimit.AuthPerMin = 20
	}
	if conf.RateLimit.ApiPerMin == 0 {
		conf.RateLimit.ApiPerMin = 300
	}
	Instance = conf
	return nil
}
