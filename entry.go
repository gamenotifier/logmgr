package logmgr

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	keyUserID     = "user_id"
	keyGinContext = "gin_context"
)

// Wrapper on a logrus.Entry to provide convenience functions
type logEntry struct {
	*logrus.Entry
}

// LoggerName returns the set logger name, or the empty string if not set
func (entry *logEntry) LoggerName() string {
	if entry.Data != nil {
		if name, ok := entry.Data[keyLoggerName]; ok {
			return name.(string)
		}
	}

	return ""
}

// UserID returns the set user id, or the empty string if not set
func (entry *logEntry) UserID() string {
	if entry.Context != nil {
		if userID := entry.Context.Value(keyUserID); userID != nil {
			return userID.(string)
		}
	}

	return ""
}

// GinContext returns the set gin request context, or the nil if not set
func (entry *logEntry) GinContext() *gin.Context {
	if entry.Context != nil {
		if c := entry.Context.Value(keyGinContext); c != nil {
			return c.(*gin.Context)
		}
	}

	return nil
}

// Error returns the set error, or the nil if not set
func (entry *logEntry) Error() error {
	if err, ok := entry.Data[logrus.ErrorKey].(error); ok {
		return err
	} else {
		return nil
	}
}
