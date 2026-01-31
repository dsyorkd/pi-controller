package defaults

// Sentry defaults
const (
	// SentryDebug indicates whether Sentry debug mode is enabled
	SentryDebug = false

	// SentryTracesSampleRate is the default performance sampling rate (10%)
	SentryTracesSampleRate = 0.1

	// SentrySampleRate is the default error sampling rate (100%)
	SentrySampleRate = 1.0

	// SentryEnableTracing indicates whether tracing is enabled
	SentryEnableTracing = true

	// SentrySendDefaultPII indicates whether PII is sent (disabled for security)
	SentrySendDefaultPII = false

	// SentryMaxBreadcrumbs is the maximum number of breadcrumbs
	SentryMaxBreadcrumbs = 100

	// SentryAttachStacktrace indicates whether to attach stack traces
	SentryAttachStacktrace = true
)
