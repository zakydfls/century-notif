package infrastructure

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"notif-gateway/internal/core/enum"
)

type Config struct {
	App           AppConfig
	HTTP          HTTPConfig
	Postgres      PostgresConfig
	Redis         RedisConfig
	Rabbit        RabbitConfig
	Logger        LoggerConfig
	Provider      ProviderConfig
	EmailSecurity EmailSecurityConfig
	Worker        WorkerConfig
	Outbox        OutboxConfig
	Migration     MigrationConfig
}

type AppConfig struct {
	Env  string
	Name string
}

type HTTPConfig struct {
	Addr               string
	ShutdownTimeout    time.Duration
	NotificationAPIKey string
}

type PostgresConfig struct {
	DSN          string
	GormLogLevel string // silent | error | warn | info — resolved in NewConfig (see resolveGormLogLevel)
}

type RedisConfig struct {
	DSN string
}

type RabbitConfig struct {
	DSN            string
	RetryTTL       time.Duration
	PublishTimeout time.Duration
}

type LoggerConfig struct {
	Level  string
	Format string
}

type ProviderConfig struct {
	DefaultSMS      enum.Provider
	DefaultWhatsApp enum.Provider
	DefaultEmail    enum.Provider
	Twilio          TwilioConfig
	Alibaba         AlibabaConfig
	WAHA            WAHAConfig
	Whatsockt       WhatsocktConfig
	GoDaddy         GoDaddyConfig
	Resend          ResendConfig
}

type TwilioConfig struct {
	AccountSID               string
	AuthToken                string
	SMSFrom                  string
	SMSMessagingService      string
	WhatsAppFrom             string
	WhatsAppService          string
	WebhookBaseURL           string // public origin; paths under /api/v1/webhooks/twilio/...
	ValidateWebhookSignature bool   // when true and AuthToken non-empty, validate X-Twilio-Signature on Twilio webhooks
}

type AlibabaConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
	RegionID        string
	Mode            enum.AlibabaMode
	From            string
	ChannelID       string
}

type WAHAConfig struct {
	BaseURL                  string
	APIKey                   string
	Session                  string
	WebhookHMACKey           string
	ValidateWebhookSignature bool
}

type WhatsocktConfig struct {
	BaseURL                  string
	APIKey                   string
	DeviceID                 string
	Timeout                  time.Duration
	WebhookSecret            string
	ValidateWebhookSignature bool
}

type GoDaddyConfig struct {
	SMTPHost       string
	SMTPPort       int
	Username       string
	Password       string
	FromAddress    string
	FromName       string
	UseImplicitTLS bool // true = port 465 implicit TLS; false = STARTTLS (typically 587)
	Timeout        time.Duration
}

type ResendConfig struct {
	APIKey                   string
	FromAddress              string
	FromName                 string
	Timeout                  time.Duration
	WebhookSecret            string
	ValidateWebhookSignature bool
}

type EmailSecurityConfig struct {
	BlockDisposable     bool
	DisposableAllowlist []string
}

type WorkerConfig struct {
	Prefetch    int
	Concurrency int
	MaxRetry    int
}

type OutboxConfig struct {
	SweepInterval  time.Duration
	BatchSize      int
	LockTTL        time.Duration
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}

type MigrationConfig struct {
	AutoMigrate bool
}

func NewConfig() (Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)
	_ = v.ReadInConfig()

	config := Config{
		App: AppConfig{
			Env:  v.GetString("APP_ENV"),
			Name: v.GetString("APP_NAME"),
		},
		HTTP: HTTPConfig{
			Addr:               v.GetString("HTTP_ADDR"),
			ShutdownTimeout:    v.GetDuration("HTTP_SHUTDOWN_TIMEOUT"),
			NotificationAPIKey: strings.TrimSpace(v.GetString("NOTIFICATION_API_KEY")),
		},
		Postgres: PostgresConfig{
			DSN:          v.GetString("POSTGRES_DSN"),
			GormLogLevel: resolveGormLogLevel(v.GetString("GORM_LOG_LEVEL"), v.GetString("APP_ENV"), v.GetString("LOG_LEVEL")),
		},
		Redis: RedisConfig{DSN: v.GetString("REDIS_DSN")},
		Rabbit: RabbitConfig{
			DSN:            v.GetString("RABBIT_DSN"),
			RetryTTL:       v.GetDuration("RABBIT_RETRY_TTL"),
			PublishTimeout: v.GetDuration("RABBIT_PUBLISH_TIMEOUT"),
		},
		Logger: LoggerConfig{
			Level:  v.GetString("LOG_LEVEL"),
			Format: v.GetString("LOG_FORMAT"),
		},
		Provider: ProviderConfig{
			DefaultSMS:      enum.Provider(v.GetString("PROVIDER_DEFAULT_SMS")),
			DefaultWhatsApp: enum.Provider(v.GetString("PROVIDER_DEFAULT_WHATSAPP")),
			DefaultEmail:    enum.Provider(v.GetString("PROVIDER_DEFAULT_EMAIL")),
			Twilio: TwilioConfig{
				AccountSID:               v.GetString("TWILIO_ACCOUNT_SID"),
				AuthToken:                v.GetString("TWILIO_AUTH_TOKEN"),
				SMSFrom:                  v.GetString("TWILIO_SMS_FROM"),
				SMSMessagingService:      v.GetString("TWILIO_SMS_MESSAGING_SERVICE"),
				WhatsAppFrom:             v.GetString("TWILIO_WHATSAPP_FROM"),
				WhatsAppService:          v.GetString("TWILIO_WHATSAPP_MESSAGING_SERVICE"),
				WebhookBaseURL:           resolveTwilioWebhookBaseURL(v),
				ValidateWebhookSignature: resolveTwilioValidateWebhookSignature(v),
			},
			Alibaba: AlibabaConfig{
				AccessKeyID:     v.GetString("ALIBABA_ACCESS_KEY_ID"),
				AccessKeySecret: v.GetString("ALIBABA_ACCESS_KEY_SECRET"),
				Endpoint:        v.GetString("ALIBABA_ENDPOINT"),
				RegionID:        v.GetString("ALIBABA_REGION_ID"),
				Mode:            enum.AlibabaMode(v.GetString("ALIBABA_MODE")),
				From:            v.GetString("ALIBABA_FROM"),
				ChannelID:       v.GetString("ALIBABA_CHANNEL_ID"),
			},
			WAHA: WAHAConfig{
				BaseURL:                  v.GetString("WAHA_BASE_URL"),
				APIKey:                   v.GetString("WAHA_API_KEY"),
				Session:                  v.GetString("WAHA_SESSION"),
				WebhookHMACKey:           v.GetString("WAHA_WEBHOOK_HMAC_KEY"),
				ValidateWebhookSignature: v.GetBool("WAHA_VALIDATE_WEBHOOK_SIGNATURE"),
			},
			Whatsockt: WhatsocktConfig{
				BaseURL:                  v.GetString("WHATSOCKT_BASE_URL"),
				APIKey:                   v.GetString("WHATSOCKT_API_KEY"),
				DeviceID:                 v.GetString("WHATSOCKT_DEVICE_ID"),
				Timeout:                  v.GetDuration("WHATSOCKT_TIMEOUT"),
				WebhookSecret:            v.GetString("WHATSOCKT_WEBHOOK_SECRET"),
				ValidateWebhookSignature: v.GetBool("WHATSOCKT_VALIDATE_WEBHOOK_SIGNATURE"),
			},
			GoDaddy: GoDaddyConfig{
				SMTPHost:       v.GetString("GODADDY_SMTP_HOST"),
				SMTPPort:       v.GetInt("GODADDY_SMTP_PORT"),
				Username:       v.GetString("GODADDY_USERNAME"),
				Password:       v.GetString("GODADDY_PASSWORD"),
				FromAddress:    v.GetString("GODADDY_FROM_ADDRESS"),
				FromName:       v.GetString("GODADDY_FROM_NAME"),
				UseImplicitTLS: v.GetBool("GODADDY_USE_IMPLICIT_TLS"),
				Timeout:        v.GetDuration("GODADDY_TIMEOUT"),
			},
			Resend: ResendConfig{
				APIKey:                   v.GetString("RESEND_API_KEY"),
				FromAddress:              v.GetString("RESEND_FROM_ADDRESS"),
				FromName:                 v.GetString("RESEND_FROM_NAME"),
				Timeout:                  v.GetDuration("RESEND_TIMEOUT"),
				WebhookSecret:            v.GetString("RESEND_WEBHOOK_SECRET"),
				ValidateWebhookSignature: v.GetBool("RESEND_VALIDATE_WEBHOOK_SIGNATURE"),
			},
		},
		EmailSecurity: EmailSecurityConfig{
			BlockDisposable:     v.GetBool("EMAIL_BLOCK_DISPOSABLE"),
			DisposableAllowlist: splitCommaSeparated(v.GetString("EMAIL_DISPOSABLE_ALLOWLIST")),
		},
		Worker: WorkerConfig{
			Prefetch:    v.GetInt("WORKER_PREFETCH"),
			Concurrency: v.GetInt("WORKER_CONCURRENCY"),
			MaxRetry:    v.GetInt("WORKER_MAX_RETRY"),
		},
		Outbox: OutboxConfig{
			SweepInterval:  v.GetDuration("OUTBOX_SWEEP_INTERVAL"),
			BatchSize:      v.GetInt("OUTBOX_SWEEP_BATCH_SIZE"),
			LockTTL:        v.GetDuration("OUTBOX_LOCK_TTL"),
			RetryBaseDelay: v.GetDuration("OUTBOX_RETRY_BASE_DELAY"),
			RetryMaxDelay:  v.GetDuration("OUTBOX_RETRY_MAX_DELAY"),
		},
		Migration: MigrationConfig{AutoMigrate: v.GetBool("MIGRATION_AUTO_MIGRATE")},
	}
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

// resolveGormLogLevel picks GORM SQL logging: explicit GORM_LOG_LEVEL wins; else debug app log -> info;
// else local env -> info (handy for dev); else warn (fewer logs in prod).
func resolveGormLogLevel(explicitGorm, appEnv, logLevel string) string {
	explicitGorm = strings.TrimSpace(strings.ToLower(explicitGorm))
	if explicitGorm != "" {
		return explicitGorm
	}
	if strings.EqualFold(logLevel, "debug") {
		return "info"
	}
	if strings.EqualFold(appEnv, enum.EnvLocal) {
		return "info"
	}
	return "warn"
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.HTTP.NotificationAPIKey) == "" {
		return fmt.Errorf("NOTIFICATION_API_KEY is required")
	}
	if !config.Provider.DefaultSMS.Supports(enum.ChannelSMS) {
		return fmt.Errorf("invalid sms provider: %s", config.Provider.DefaultSMS)
	}
	if !config.Provider.DefaultWhatsApp.Supports(enum.ChannelWhatsApp) {
		return fmt.Errorf("invalid whatsapp provider: %s", config.Provider.DefaultWhatsApp)
	}
	if !config.Provider.DefaultEmail.Supports(enum.ChannelEmail) {
		return fmt.Errorf("invalid email provider: %s", config.Provider.DefaultEmail)
	}
	return nil
}

func resolveTwilioWebhookBaseURL(v *viper.Viper) string {
	if s := strings.TrimSpace(v.GetString("TWILIO_WEBHOOK_BASE_URL")); s != "" {
		return s
	}
	return strings.TrimSpace(v.GetString("TWILIO_STATUS_CALLBACK_BASE_URL"))
}

func resolveTwilioValidateWebhookSignature(v *viper.Viper) bool {
	if v.IsSet("TWILIO_VALIDATE_WEBHOOK_SIGNATURE") {
		return v.GetBool("TWILIO_VALIDATE_WEBHOOK_SIGNATURE")
	}
	if v.IsSet("TWILIO_VALIDATE_CALLBACK_SIGNATURE") {
		return v.GetBool("TWILIO_VALIDATE_CALLBACK_SIGNATURE")
	}
	return v.GetBool("TWILIO_VALIDATE_WEBHOOK_SIGNATURE")
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("APP_ENV", enum.EnvLocal)
	v.SetDefault("APP_NAME", "notif-gateway")
	v.SetDefault("HTTP_ADDR", enum.DefaultHTTPAddr)
	v.SetDefault("HTTP_SHUTDOWN_TIMEOUT", "10s")
	v.SetDefault("RABBIT_RETRY_TTL", "30s")
	v.SetDefault("RABBIT_PUBLISH_TIMEOUT", "5s")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", enum.DefaultLogFormat)
	v.SetDefault("GORM_LOG_LEVEL", "")
	v.SetDefault("PROVIDER_DEFAULT_SMS", enum.ProviderTwilio.String())
	v.SetDefault("PROVIDER_DEFAULT_WHATSAPP", enum.ProviderWhatsockt.String())
	v.SetDefault("PROVIDER_DEFAULT_EMAIL", enum.ProviderGoDaddy.String())
	v.SetDefault("GODADDY_SMTP_HOST", "smtpout.secureserver.net")
	v.SetDefault("GODADDY_SMTP_PORT", 587)
	v.SetDefault("GODADDY_USE_IMPLICIT_TLS", false)
	v.SetDefault("GODADDY_TIMEOUT", "15s")
	v.SetDefault("RESEND_TIMEOUT", "15s")
	v.SetDefault("RESEND_VALIDATE_WEBHOOK_SIGNATURE", true)
	v.SetDefault("ALIBABA_ENDPOINT", "dysmsapi.ap-southeast-1.aliyuncs.com")
	v.SetDefault("ALIBABA_REGION_ID", "ap-southeast-1")
	v.SetDefault("ALIBABA_MODE", enum.AlibabaModeGlobe.String())
	v.SetDefault("WAHA_SESSION", "default")
	v.SetDefault("WAHA_VALIDATE_WEBHOOK_SIGNATURE", true)
	v.SetDefault("WHATSOCKT_TIMEOUT", "15s")
	v.SetDefault("WHATSOCKT_VALIDATE_WEBHOOK_SIGNATURE", true)
	v.SetDefault("EMAIL_BLOCK_DISPOSABLE", true)
	v.SetDefault("WORKER_CONCURRENCY", 8)
	v.SetDefault("WORKER_PREFETCH", 16)
	v.SetDefault("WORKER_MAX_RETRY", 3)
	v.SetDefault("OUTBOX_SWEEP_INTERVAL", "2s")
	v.SetDefault("OUTBOX_SWEEP_BATCH_SIZE", 100)
	v.SetDefault("OUTBOX_LOCK_TTL", "30s")
	v.SetDefault("OUTBOX_RETRY_BASE_DELAY", "5s")
	v.SetDefault("OUTBOX_RETRY_MAX_DELAY", "5m")
	v.SetDefault("MIGRATION_AUTO_MIGRATE", true)
	v.SetDefault("TWILIO_VALIDATE_WEBHOOK_SIGNATURE", true)
}

func splitCommaSeparated(value string) []string {
	var result []string
	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
