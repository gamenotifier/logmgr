package logmgr

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"strings"
)

// Logger extends a logrus logger by providing path-like names.
type Logger struct {
	*logrus.Entry

	manager *SentryManager
	name    []string
}

// Extend returns a Logger with an extended name path.
func (l *Logger) Extend(name string) *Logger {
	log := logrus.New()
	log.AddHook(l.manager) // inherit manager

	newName := append(l.name, name)
	newNameStr := strings.Join(newName, ".")
	return &Logger{
		Entry:   log.WithField(keyLoggerName, newNameStr),
		manager: l.manager,
		name:    newName,
	}
}

func (l *Logger) ensureContext() {
	if l.Entry.Context == nil {
		l.Entry.Context = context.Background()
	}
}

// WithUser includes the given user id in the logger context.
func (l *Logger) WithUser(userID string) *Logger {
	l.ensureContext()
	l.Entry.Context = context.WithValue(l.Entry.Context, keyUserID, userID)
	return &Logger{
		Entry:   l.WithField(keyUserID, userID),
		manager: l.manager,
		name:    l.name,
	}
}

// WithGin includes the given gin request context in the logger context.
func (l *Logger) WithGin(c *gin.Context) *Logger {
	l.ensureContext()
	l.Entry.Context = context.WithValue(l.Entry.Context, keyGinContext, c)
	return l
}
