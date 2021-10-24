package logmgr

type SentryConfig struct {
	// DSN to connect with Sentry
	DSN string

	// Environment passed to Sentry
	Environment string

	// Release passed to Sentry
	Release string

	// Log level strings to report messages to Sentry
	LogLevels []string
}
