package configs

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/chenzanhong/zlog"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port        int    `yaml:"port"`
	MetricsPort int    `yaml:"metrics_port"`
	PprofPort   int    `yaml:"pprof_port"`
	EnableTrace bool   `yaml:"enable_trace"`
	Env         string `yaml:"env"`
}

type JWTConfig struct {
	Key                    string `yaml:"key"`
	AccessTokenExpireTime  int    `yaml:"access_token_expire_time"`  // 单位秒
	RefreshTokenExpireTime int    `yaml:"refresh_token_expire_time"` // 单位秒
}

type PGConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type RedisConfig struct {
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	Password    string `yaml:"password"`
	DB          int    `yaml:"db"`
	MaxConn     int    `yaml:"max_conn"`
	MaxIdleConn int    `yaml:"max_idle_conn"`
}

// type EMAILConfig struct {
// 	Name     string `yaml:"email_name"`
// 	Password string `yaml:"email_password"`
// }
// type SMTPServerConfig struct {
// 	Host string `yaml:"SMTPServer_host"`
// 	Port string `yaml:"SMTPServer_port"`
// }

type RateConfig struct {
	UserRate  int `yaml:"user_rate"`
	UserBurst int `yaml:"user_burst"`
}

type KafkaConfig struct {
	Brokers string `yaml:"brokers"`
	Topic   string `yaml:"topic"`
}

type LogConfig struct {
	Level      string `yaml:"level"`
	Output     string `yaml:"output"` // "console", "file", "both"
	Format     string `yaml:"format"` // "json", "console" (只对终端输出生效)
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
	Sampling   bool   `yaml:"sampling"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	JWT    JWTConfig    `yaml:"jwt"`
	PG     PGConfig     `yaml:"pg"`
	Redis  RedisConfig  `yaml:"redis"`
	// Email      EMAILConfig      `yaml:"email"`
	// SMTPServer SMTPServerConfig `yaml:"smtp_server"`
	Rate  RateConfig        `yaml:"rate"`
	Kafka KafkaConfig       `yaml:"kafka"`
	Log   zlog.LoggerConfig `yaml:"log"`
}

// getConfigPath 获取数据库配置文件的路径
func getConfigPath() string {
	_, filename, _, ok := runtime.Caller(2) // 获取调用者的文件名
	if !ok {
		log.Fatal("无法获取运行时调用者信息")
	}

	// 获取当前文件所在的目录
	currentDir := filepath.Dir(filename)

	// 构建到项目根目录的相对路径
	dbConfigPath := filepath.Join(currentDir, "..", "configs", "config.yaml")

	// 将路径转换为绝对路径并简化路径
	absPath, err := filepath.Abs(dbConfigPath)
	if err != nil {
		log.Printf("无法获取绝对路径: %v", err)
	}

	simplifiedPath := filepath.Clean(absPath)

	return simplifiedPath
}

// GetConfigPath 返回数据库配置文件的路径
func GetConfigPath() string {
	return getConfigPath()
}

// LoadConfig 加载配置文件并返回 DBConfig
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()
	var config Config
	if yamlFile, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(yamlFile, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config.yaml: %w", err)
		}
	} else {
		// config.yaml 不存在，使用零值（后续会被环境变量覆盖）
		zap.L().Info("config.yaml not found, using defaults from environment variables")
	}

	// 用环境变量覆盖所有字段（必须）
	ApplyEnvToConfig(&config)

	// 可选：验证必要字段是否已设置
	if config.Server.Port == 0 {
		return nil, fmt.Errorf("required env SERVER_PORT is not set")
	}

	return &config, nil
}

// ApplyEnvToConfig 使用环境变量覆盖 config 中的字段（仅当环境变量非空时）
func ApplyEnvToConfig(cfg *Config) {
	getEnv := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	getEnvInt := func(key string, fallback int) int {
		if v := getEnv(key, ""); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
		return fallback
	}
	getEnvBool := func(key string, fallback bool) bool {
		if v := getEnv(key, ""); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		}
		return fallback
	}

	// Server
	cfg.Server.Port = getEnvInt("SERVER_PORT", cfg.Server.Port)
	cfg.Server.MetricsPort = getEnvInt("METRICS_PORT", cfg.Server.MetricsPort)
	cfg.Server.PprofPort = getEnvInt("PPROF_PORT", cfg.Server.PprofPort)
	cfg.Server.EnableTrace = getEnvBool("ENABLE_TRACE", cfg.Server.EnableTrace)
	cfg.Server.Env = getEnv("SERVER_ENV", cfg.Server.Env)

	// JWT
	cfg.JWT.Key = getEnv("JWT_KEY", cfg.JWT.Key)
	cfg.JWT.AccessTokenExpireTime = getEnvInt("JWT_ACCESS_TOKEN_EXPIRE_TIME", cfg.JWT.AccessTokenExpireTime)
	cfg.JWT.RefreshTokenExpireTime = getEnvInt("JWT_REFRESH_TOKEN_EXPIRE_TIME", cfg.JWT.RefreshTokenExpireTime)

	// PostgreSQL
	cfg.PG.Host = getEnv("DB_HOST", cfg.PG.Host)
	cfg.PG.Port = getEnv("DB_PORT", cfg.PG.Port)
	cfg.PG.Name = getEnv("DB_NAME", cfg.PG.Name)
	cfg.PG.User = getEnv("DB_USER", cfg.PG.User)
	cfg.PG.Password = getEnv("DB_PASSWORD", cfg.PG.Password)

	// Redis
	cfg.Redis.Host = getEnv("REDIS_HOST", cfg.Redis.Host)
	cfg.Redis.Port = getEnv("REDIS_PORT", cfg.Redis.Port)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", cfg.Redis.Password)
	cfg.Redis.DB = getEnvInt("REDIS_DB", cfg.Redis.DB)
	cfg.Redis.MaxConn = getEnvInt("REDIS_MAX_CONN", cfg.Redis.MaxConn)
	cfg.Redis.MaxIdleConn = getEnvInt("REDIS_MAX_IDLE_CONN", cfg.Redis.MaxIdleConn)

	// Rate
	cfg.Rate.UserRate = getEnvInt("RATE_USER_RATE", cfg.Rate.UserRate)
	cfg.Rate.UserBurst = getEnvInt("RATE_USER_BURST", cfg.Rate.UserBurst)

	// Kafka
	cfg.Kafka.Brokers = getEnv("KAFKA_BROKERS", cfg.Kafka.Brokers)
	cfg.Kafka.Topic = getEnv("KAFKA_TOPIC", cfg.Kafka.Topic)

	// Log
	// 注意：zlog.Level 需要能从字符串解析
	if levelStr := getEnv("LOG_LEVEL", cfg.Log.Level.String()); levelStr != "" {
		cfg.Log.Level = zlog.Level(levelStr)
	}
	cfg.Log.Output = getEnv("LOG_OUTPUT", cfg.Log.Output)
	cfg.Log.Format = getEnv("LOG_FORMAT", cfg.Log.Format)
	cfg.Log.FilePath = getEnv("LOG_FILE_PATH", cfg.Log.FilePath)
	cfg.Log.MaxSize = getEnvInt("LOG_MAX_SIZE", cfg.Log.MaxSize)
	cfg.Log.MaxBackups = getEnvInt("LOG_MAX_BACKUPS", cfg.Log.MaxBackups)
	cfg.Log.MaxAge = getEnvInt("LOG_MAX_AGE", cfg.Log.MaxAge)
	cfg.Log.Compress = getEnvBool("LOG_COMPRESS", cfg.Log.Compress)
	cfg.Log.Sampling = getEnvBool("LOG_SAMPLING", cfg.Log.Sampling)
	// 单独处理 Fields
	override := parseLogFieldsFromEnv()
	if override != nil {
		// 合并：保留 cfg.Log.Fields 已有字段，用 override 覆盖/新增
		if cfg.Log.Fields == nil {
			cfg.Log.Fields = make(map[string]string)
		}
		for k, v := range override {
			cfg.Log.Fields[k] = v
		}
	}
}

func parseLogFieldsFromEnv() map[string]string {
	raw := os.Getenv("LOG_FIELDS")
	if raw == "" {
		return nil // 或空 map
	}
	var fields map[string]string
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		log.Printf("Invalid LOG_FIELDS, ignoring: %v", err)
		return nil
	}
	return fields
}

func SyncConfigToEnv(config Config) {
	// config, err := LoadConfig()
	// if err != nil {
	// 	log.Fatalf("加载配置失败：%v", err.Error())
	// }

	// 辅助函数：如果 envVar 未设置，则用 fallback 值设置它
	setEnvIfNotSet := func(envVar, fallback string) {
		if os.Getenv(envVar) == "" {
			os.Setenv(envVar, fallback)
		}
	}
	// Server
	setEnvIfNotSet("SERVER_PORT", strconv.Itoa(config.Server.Port))
	setEnvIfNotSet("METRICS_PORT", strconv.Itoa(config.Server.MetricsPort))
	setEnvIfNotSet("PPROF_PORT", strconv.Itoa(config.Server.PprofPort))
	setEnvIfNotSet("ENABLE_TRACE", strconv.FormatBool(config.Server.EnableTrace))
	setEnvIfNotSet("SERVER_ENV", config.Server.Env)

	// jwt
	setEnvIfNotSet("JWT_KEY", config.JWT.Key)
	setEnvIfNotSet("JWT_ACCESS_TOKEN_EXPIRE_TIME", strconv.Itoa(config.JWT.AccessTokenExpireTime))
	setEnvIfNotSet("JWT_REFRESH_TOKEN_EXPIRE_TIME", strconv.Itoa(config.JWT.RefreshTokenExpireTime))

	// PostgreSQL
	setEnvIfNotSet("DB_USER", config.PG.User)
	setEnvIfNotSet("DB_PASSWORD", config.PG.Password)
	setEnvIfNotSet("DB_HOST", config.PG.Host)
	setEnvIfNotSet("DB_PORT", config.PG.Port)
	setEnvIfNotSet("DB_NAME", config.PG.Name)

	// Redis
	setEnvIfNotSet("REDIS_HOST", config.Redis.Host)
	setEnvIfNotSet("REDIS_PORT", config.Redis.Port)
	setEnvIfNotSet("REDIS_PASSWORD", config.Redis.Password)
	setEnvIfNotSet("REDIS_DB", strconv.Itoa(config.Redis.DB))
	setEnvIfNotSet("REDIS_MAX_CONN", strconv.Itoa(config.Redis.MaxConn))
	setEnvIfNotSet("REDIS_MAX_IDLE_CONN", strconv.Itoa(config.Redis.MaxIdleConn))
	/*
		// Email
		setEnvIfNotSet("EMAIL_NAME", config.Email.Name)
		setEnvIfNotSet("EMAIL_PASSWORD", config.Email.Password)

		// SMTP Server
		setEnvIfNotSet("SMTP_SERVER_HOST", config.SMTPServer.Host)
		setEnvIfNotSet("SMTP_SERVER_PORT", config.SMTPServer.Port)
	*/
	// Rate Limiting
	setEnvIfNotSet("RATE_USER_RATE", strconv.Itoa(config.Rate.UserRate))
	setEnvIfNotSet("RATE_USER_BURST", strconv.Itoa(config.Rate.UserBurst))

	// Kafka
	setEnvIfNotSet("KAFKA_BROKERS", config.Kafka.Brokers)
	setEnvIfNotSet("KAFKA_TOPIC", config.Kafka.Topic)

	// Log
	setEnvIfNotSet("LOG_LEVEL", config.Log.Level.String())
	setEnvIfNotSet("LOG_OUTPUT", config.Log.Output)
	setEnvIfNotSet("LOG_FORMAT", config.Log.Format)
	setEnvIfNotSet("LOG_FILE_PATH", config.Log.FilePath)
	setEnvIfNotSet("LOG_MAX_SIZE", strconv.Itoa(config.Log.MaxSize))
	setEnvIfNotSet("LOG_MAX_BACKUPS", strconv.Itoa(config.Log.MaxBackups))
	setEnvIfNotSet("LOG_MAX_AGE", strconv.Itoa(config.Log.MaxAge))
	setEnvIfNotSet("LOG_COMPRESS", strconv.FormatBool(config.Log.Compress))
	setEnvIfNotSet("LOG_SAMPLING", strconv.FormatBool(config.Log.Sampling))
}
