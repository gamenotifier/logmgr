package logmgr

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	keySentryHub  = "sentry_hub"
	keyLoggerName = "zone"

	breadcrumbLimit = 5
	fingerprintBase = "api"
)

type LoggerMaker interface {
	NewLogger(name string) *Logger
}

// SentryManager is a struct that issues loggers tied back to
// a central sentry.Hub instance.
type SentryManager struct {
	hub         *sentry.Hub
	errorLevels []logrus.Level
	logLevel    logrus.Level
}

// NewSentryManager creates a SentryManager with a logLevel output minimum.
func NewSentryManager(config *SentryConfig, logLevel string) (*SentryManager, error) {
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         config.DSN,
		Environment: config.Environment,
		Release:     config.Release,
	})
	if err != nil {
		return nil, err
	}

	baseScope := sentry.NewScope()

	// Parse errorLevels from config
	var levels []logrus.Level
	for _, level := range config.LogLevels {
		parsedLevel, err := logrus.ParseLevel(level)
		if err != nil {
			return nil, fmt.Errorf("set error levels: %w", err)
		}
		levels = append(levels, parsedLevel)
	}

	parsedLogLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return nil, fmt.Errorf("set log level: %w", err)
	}

	return &SentryManager{
		hub:         sentry.NewHub(client, baseScope),
		errorLevels: levels,
		logLevel:    parsedLogLevel,
	}, nil
}

// Levels implements the logrus.Hook interface.
func (m *SentryManager) Levels() []logrus.Level {
	return m.errorLevels
}

// Fire implements the logrus.Hook interface.
// Log entries will be logged to Sentry.
func (m *SentryManager) Fire(lEntry *logrus.Entry) error {
	entry := &logEntry{lEntry} // wrap for convenience functions

	// Check if one hub was set during HTTP middleware
	var hub *sentry.Hub = nil
	if entry.Context != nil {
		if ctxHub, ok := entry.Context.Value(keySentryHub).(*sentry.Hub); ok && ctxHub != nil {
			hub = ctxHub
		}
	}

	if hub == nil {
		hub = m.hub.Clone()
	}

	go func(hub *sentry.Hub) {
		defer recoverFromLogging(hub)

		// hub should be one cloned *sentry.Hub in both cases, so we are free to modify it within this goroutine
		event := sentry.NewEvent()
		event.Level = sentry.Level(entry.Level.String())
		event.Message = entry.Message
		event.Extra = entry.Data
		event.Timestamp = entry.Time

		loggerName := entry.LoggerName()
		if loggerName != "" {
			event.Logger = loggerName
			event.Fingerprint = append(event.Fingerprint, loggerName)
		}

		if ginCtx := entry.GinContext(); ginCtx != nil {
			event.Fingerprint = []string{fingerprintBase, ginCtx.Request.Method, ginCtx.FullPath(), entry.Message}
		} else if loggerName != "" {
			event.Fingerprint = []string{fingerprintBase, loggerName, entry.Message}
		}

		enrichEventWithError(hub, entry)

		// Add DB userID if included
		if userID := entry.UserID(); userID != "" {
			event.User = sentry.User{
				ID: userID,
			}
		}

		hub.CaptureEvent(event)
	}(hub)

	return nil
}

// recoverFromLogging attempts to recover from a panic within the sentry logging logic.
func recoverFromLogging(hub *sentry.Hub) {
	if err := recover(); err != nil {
		event := sentry.NewEvent()
		event.Level = sentry.LevelError
		event.Message = "recovered panic within sentry error logging"
		event.Fingerprint = []string{fingerprintBase, "sentry_panic"}

		if asErr, ok := err.(error); ok {
			hub.Scope().AddBreadcrumb(&sentry.Breadcrumb{
				Type:     "error",
				Category: "sentry.panic",
				Message:  asErr.Error(),
				Level:    "fatal",
			}, breadcrumbLimit)
		} else if asStr, ok := err.(string); ok {
			hub.Scope().AddBreadcrumb(&sentry.Breadcrumb{
				Type:     "error",
				Category: "sentry.panic",
				Message:  asStr,
				Level:    "fatal",
			}, breadcrumbLimit)
		}

		hub.CaptureEvent(event)
	}
}

// enrichEventWithError adds error and db query breadcrumbs if errors are present on entry.
func enrichEventWithError(hub *sentry.Hub, entry *logEntry) {
	// Add error breadcrumb if included
	if err := entry.Error(); err != nil {
		tryCoerceQuery(err, func(q query) {
			hub.Scope().AddBreadcrumb(&sentry.Breadcrumb{
				Type:     "query",
				Category: "db",
				Message:  "database query",
				Data: map[string]interface{}{
					"query": q.QueryBody(),
					"args":  q.QueryArgs(),
				},
				Level:     "info",
				Timestamp: entry.Time,
			}, breadcrumbLimit)
		})

		hub.Scope().AddBreadcrumb(&sentry.Breadcrumb{
			Type:      "error",
			Category:  entry.LoggerName(),
			Message:   err.Error(),
			Level:     "error",
			Timestamp: entry.Time,
		}, breadcrumbLimit)
	}
}

// WithRequestContext issues a new sentry.Hub and sets
// the Hub's current request to req.
// To be used in gin.Use as middleware.
func (m *SentryManager) WithRequestContext(ctx *gin.Context) {
	// Clone the hub to prevent interference with other requests
	hub := m.hub.Clone()
	hub.Scope().SetRequest(ctx.Request)
	ctx.Set(keySentryHub, hub)
}

// NewLogger returns a Logger with the given name, linked to this SentryManager object
func (m *SentryManager) NewLogger(name string) *Logger {
	log := logrus.New()
	log.AddHook(m)
	return &Logger{
		Entry:   log.WithField(keyLoggerName, name),
		manager: m,
		name:    []string{name},
	}
}
