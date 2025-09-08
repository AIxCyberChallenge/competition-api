package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

type Team struct {
	CRS    *CRSConfig `mapstructure:"crs"     json:"crs"     validate:"required"`
	ID     string     `mapstructure:"id"      json:"id"      validate:"required,uuid_rfc4122"`
	Note   string     `mapstructure:"note"    json:"note"    validate:"required"`
	APIKey APIKey     `mapstructure:"api_key" json:"api_key" validate:"required"`
}

type APIKeyPermissions struct {
	CRS                   bool `mapstructure:"crs"                    json:"crs"`
	CompetitionManagement bool `mapstructure:"competition_management" json:"competition_managment"`
	JobRunner             bool `mapstructure:"job_runner"             json:"job_runner"`
}

type APIKey struct {
	Active      *bool             `mapstructure:"active"      json:"active"      validate:"required"`
	Token       string            `mapstructure:"token"       json:"token"       validate:"required"`
	Permissions APIKeyPermissions `mapstructure:"permissions" json:"permissions"`
}

type PostgresConfig struct {
	User               string        `validate:"required"`
	Password           string        `validate:"required"`
	Host               string        `validate:"required"`
	Database           string        `validate:"required"`
	MaxIdleConnections int           `validate:"required" mapstructure:"max_idle_connections"`
	MaxOpenConnections int           `validate:"required" mapstructure:"max_open_connections"`
	ConnectionTTL      time.Duration `validate:"required" mapstructure:"connection_ttl"`
	Port               int16         `validate:"required"`
}

type AzureConfig struct {
	StorageAccount *AzureStorageAccountConfig `mapstructure:"storage_account" validate:"required"`
	Dev            bool                       `mapstructure:"dev"`
}

type AzureStorageAccountConfig struct {
	Containers *AzureStorageAccountContainerConfig `mapstructure:"containers" validate:"required"`
	Queues     *AzureStorageAccountQueueConfig     `mapstructure:"queues"     validate:"required"`
	Name       string                              `mapstructure:"name"       validate:"required"`
	Key        string                              `mapstructure:"key"        validate:"required"`
}

type AzureStorageAccountContainerConfig struct {
	URL         string `mapstructure:"url"         validate:"required"`
	Artifacts   string `mapstructure:"artifacts"   validate:"required"`
	Submissions string `mapstructure:"submissions" validate:"required"`
	Sources     string `mapstructure:"sources"     validate:"required"`
}

type AzureStorageAccountQueueConfig struct {
	URL     string `mapstructure:"url"     validate:"required"`
	Results string `mapstructure:"results" validate:"required"`
}

type SlogConfig struct {
	Level int `mapstructure:"level"`
}

type GormLogConfig struct {
	Level        int  `mapstructure:"level"`
	TraceQueries bool `mapstructure:"trace_queries"`
}

type LoggingConfig struct {
	Gorm    GormLogConfig `mapstructure:"gorm"`
	App     SlogConfig    `mapstructure:"app"`
	UseOTLP bool          `mapstructure:"use_otlp"`
}

type K8SLabel struct {
	Key   string `mapstructure:"key"   validate:"required"`
	Value string `mapstructure:"value" validate:"required"`
}

type NodeAssignment struct {
	NodeAffinityLabel *K8SLabel `mapstructure:"node_affinity_label" validate:"required"`
	Toleration        *K8SLabel `mapstructure:"toleration"          validate:"required"`
}

type K8sConfig struct {
	EvalNodeAssignment      *NodeAssignment `mapstructure:"eval_node_assignment"      validate:"required"`
	BroadcastNodeAssignment *NodeAssignment `mapstructure:"broadcast_node_assignment" validate:"required"`
	ScoringNodeAssignment   *NodeAssignment `mapstructure:"scoring_node_assignment"   validate:"required"`
	Namespace               string          `mapstructure:"namespace"                 validate:"required"`
	JobImage                string          `mapstructure:"job_image"                 validate:"required"`
	DINDImage               string          `mapstructure:"dind_image"                validate:"required"`
	InCluster               bool            `mapstructure:"in_cluster"`
}

type GithubConfig struct {
	WebhookSecret *string `mapstructure:"webhook_secret" validate:"required"`
	AppID         *int64  `mapstructure:"app_id"         validate:"required"`
	AppKeyPath    *string `mapstructure:"app_key_path"   validate:"required"`
}

type CRSConfig struct {
	TaskMe      *bool  `mapstructure:"task_me"       json:"task_me"       validate:"required"`
	URL         string `mapstructure:"url"           json:"url"           validate:"required"`
	APIKeyID    string `mapstructure:"api_key_id"    json:"api_key_id"    validate:"required"`
	APIKeyToken string `mapstructure:"api_key_token" json:"api_key_token" validate:"required"`
}

type S3ArchiveConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	BucketName      string `mapstructure:"bucket_name"`
	SSLEnabled      bool   `mapstructure:"ssl_enabled"`
}

type RateLimitConfig struct {
	RedisHost       string `mapstructure:"redis_host"`
	GlobalPerMinute int64  `mapstructure:"global_per_minute"`
	SubmitPerMinute int64  `mapstructure:"submit_per_minute"`
	FailOpen        bool   `mapstructure:"fail_open"`
}

type GenerateRepoConfig struct {
	RepoURL *string `mapstructure:"repo_url" validate:"required"`
	HeadRef *string `mapstructure:"head_ref" validate:"required"`
	BaseRef *string `mapstructure:"base_ref"`
}

type GenerateChallengeConfig struct {
	Name   *string            `mapstructure:"name"   validate:"required"`
	Config GenerateRepoConfig `mapstructure:"config" validate:"required"`
}

type GenerateConfig struct {
	Enabled        *bool                     `mapstructure:"enabled"         validate:"required"`
	InstallationID *int64                    `mapstructure:"installation_id" validate:"required"`
	RoundID        *string                   `mapstructure:"round_id"        validate:"required"`
	Challenges     []GenerateChallengeConfig `mapstructure:"challenges"      validate:"required"`
}

// See competitionapi.yaml for an example config
type Config struct {
	RoundID                  *string          `mapstructure:"round_id"                     validate:"required"`
	Postgres                 *PostgresConfig  `mapstructure:"postgres"                     validate:"required"`
	Azure                    *AzureConfig     `mapstructure:"azure"                        validate:"required"`
	Logging                  *LoggingConfig   `mapstructure:"logging"                      validate:"required"`
	K8s                      *K8sConfig       `mapstructure:"k8s"                          validate:"required"`
	Github                   *GithubConfig    `mapstructure:"github"                       validate:"required"`
	S3Archive                *S3ArchiveConfig `mapstructure:"s3_archive"                   validate:"required"`
	CRSStatusPollTimeSeconds *int             `mapstructure:"crs_status_poll_time_seconds"`
	RateLimit                *RateLimitConfig `mapstructure:"ratelimit"`
	TempDir                  *string          `mapstructure:"temp_dir"`
	Generate                 *GenerateConfig  `mapstructure:"generate"                     validate:"required"`
	IgnoredRepos             *[]string        `mapstructure:"ignored_repos"`
	CacheKey                 *string          `mapstructure:"cache_key"                    validate:"required"`
	ListenAddress            string           `mapstructure:"listen_address"               validate:"required"`
	Teams                    []Team           `mapstructure:"teams"                        validate:"required"`
	GracefulShutdownSecs     int64            `mapstructure:"graceful_shutdown_secs"`
}

const (
	AppLogLevel                string = "logging.app.level"
	AzureDev                   string = "azure.dev"
	AzureStorageAccountKey     string = "azure.storage_account.key"
	CRSStatusPollTimeSeconds   string = "crs_status_poll_time_seconds"
	EnvPrefix                  string = "competitionapi"
	UseOTLP                    string = "logging.use_otlp"
	GlobalPerMinute            string = "ratelimit.global_per_minute"
	GormLogLevel               string = "logging.gorm.level"
	GormTraceQueries           string = "logging.gorm.trace_queries"
	GracefulShutdownSecs       string = "graceful_shutdown_secs"
	K8sDINDImage               string = "k8s.dind_image"
	K8sJobImage                string = "k8s.job_image"
	ListenAddress              string = "listen_address"
	PostgresDatabase           string = "postgres.database"
	PostgresHost               string = "postgres.host"
	PostgresPassword           string = "postgres.password"
	PostgresPort               string = "postgres.port"
	PostgresUser               string = "postgres.user"
	PostgresMaxIdleConnections string = "postgres.max_idle_connections"
	PostgresMaxOpenConnections string = "postgres.max_open_connections"
	PostgresConnectonTTL       string = "postgres.connection_ttl"
	RateLimitFailOpen          string = "ratelimit.fail_open"
	RedisHost                  string = "ratelimit.redis_host"
	RoundID                    string = "round_id"
	S3AccessKeyID              string = "s3_archive.access_key_id"
	S3ArchiveEnabled           string = "s3_archive.enabled"
	S3SSLEnabled               string = "s3_archive.ssl_enabled"
	S3SecretAccessKey          string = "s3_archive.secret_access_key" // #nosec
	SubmitPerMinute            string = "ratelimit.submit_per_minute"
	TempDir                    string = "temp_dir"
	GenerateRoundID            string = "generate.round_id"
	CacheKey                   string = "cache_key"
)

var configReady = false
var config Config

func GetConfig() (*Config, error) {
	if configReady {
		logger.Logger.Debug("returning already-loaded config")
		return &config, nil
	}
	logger.Logger.Info("loading config")

	v := viper.New()

	v.SetConfigName("competitionapi")

	v.AddConfigPath("/etc/competitionapi/")
	v.AddConfigPath(".")

	v.SetConfigType("yaml")

	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.AutomaticEnv()

	// workaround for https://github.com/spf13/viper/issues/761
	// bind env vars explicitly so they unmarshal into the nested struct
	err := v.BindEnv(PostgresPassword)
	if err != nil {
		return nil, err
	}
	err = v.BindEnv(AzureStorageAccountKey)
	if err != nil {
		return nil, err
	}
	err = v.BindEnv(K8sJobImage)
	if err != nil {
		return nil, err
	}

	err = v.BindEnv(K8sDINDImage)
	if err != nil {
		return nil, err
	}

	err = v.BindEnv(CacheKey)
	if err != nil {
		return nil, err
	}

	err = v.BindEnv(S3AccessKeyID)
	if err != nil {
		return nil, err
	}

	err = v.BindEnv(S3SecretAccessKey)
	if err != nil {
		return nil, err
	}

	v.SetDefault(ListenAddress, "[::]:1323")
	v.SetDefault(PostgresHost, "localhost")
	v.SetDefault(PostgresPort, 5432)
	v.SetDefault(PostgresMaxIdleConnections, 2)
	v.SetDefault(PostgresMaxOpenConnections, 10)
	v.SetDefault(PostgresConnectonTTL, 10*time.Minute)
	v.SetDefault(AzureDev, false)
	v.SetDefault(GormLogLevel, int(slog.LevelDebug))
	v.SetDefault(GormTraceQueries, false)
	v.SetDefault(AppLogLevel, int(slog.LevelDebug))
	v.SetDefault(S3ArchiveEnabled, true)
	v.SetDefault(S3SSLEnabled, true)
	v.SetDefault(CRSStatusPollTimeSeconds, 60)

	v.SetDefault(RedisHost, "localhost")
	v.SetDefault(GlobalPerMinute, 0)
	v.SetDefault(SubmitPerMinute, 0)
	v.SetDefault(RateLimitFailOpen, true)

	v.SetDefault(UseOTLP, false)

	v.SetDefault(TempDir, "/tmp")
	v.SetDefault(GracefulShutdownSecs, 30)

	v.SetDefault(GenerateRoundID, "integration-testing-round-1234")

	err = v.ReadInConfig()
	if err != nil {
		// ignore config file not found to allow pure env config
		if _, ok := err.(*viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	err = v.Unmarshal(&config)
	if err != nil {
		configReady = false
		return nil, err
	}

	valid := validator.Create()
	err = valid.Validate(&config)
	if err != nil {
		configReady = false
		return nil, err
	}

	configReady = true
	return &config, nil
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s",
		url.QueryEscape(c.Postgres.User),
		url.QueryEscape(c.Postgres.Password),
		c.Postgres.Host, c.Postgres.Port,
		url.QueryEscape(c.Postgres.Database),
	)
}
