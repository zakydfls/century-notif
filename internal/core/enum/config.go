package enum

const (
	EnvLocal         = "local"
	EnvProduction    = "production"
	EnvDevelopment   = "development"
	EnvStaging       = "staging"
	DefaultHTTPAddr  = ":8080"
	DefaultLogFormat = "json"
)

const (
	HeaderContentType     = "Content-Type"
	HeaderApplicationJSON = "application/json"
	HeaderRetryCount      = "x-retry-count"
	HeaderRequestID       = "x-request-id"
	HeaderNotificationKey = "x-notification-key"
	HeaderCenturyService  = "X-Century-Service"
)
