package logmgr

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"os"
	"strings"
)

// Logger extends a logrus logger by providing path-like names.
type Logger interface {
	logrus.FieldLogger

	// Extend returns a Logger with an extended name path.
	Extend(name string) Logger
	// WithUser includes the given user id in the logger context.
	WithUser(userID string) Logger
	// WithGin includes the given gin request context in the logger context.
	WithGin(c *gin.Context) Logger
	// Recovery returns a gin handler function that will recover and log panics.
	Recovery() gin.HandlerFunc
}

type logger struct {
	*logrus.Entry

	level   logrus.Level
	manager *SentryManager
	name    []string
}

// NewPlainLogger creates a Logger without a SentryManager.
func NewPlainLogger(name string, level logrus.Level) Logger {
	log := logrus.New()
	return &logger{
		Entry:   log.WithField(keyLoggerName, name),
		level:   level,
		manager: nil,
		name:    []string{name},
	}
}

func (l *logger) Extend(name string) Logger {
	log := logrus.New()
	log.SetLevel(l.level)
	if l.manager != nil {
		log.AddHook(l.manager) // inherit manager
	}

	newName := append(l.name, name)
	newNameStr := strings.Join(newName, ".")
	return &logger{
		Entry:   log.WithField(keyLoggerName, newNameStr),
		manager: l.manager,
		name:    newName,
	}
}

func (l *logger) ensureContext() {
	if l.Entry.Context == nil {
		l.Entry.Context = context.Background()
	}
}

func (l *logger) WithUser(userID string) Logger {
	l.ensureContext()
	l.Entry.Context = context.WithValue(l.Entry.Context, keyUserID, userID)
	return &logger{
		Entry:   l.WithField(keyUserID, userID),
		manager: l.manager,
		name:    l.name,
	}
}

func (l *logger) WithGin(c *gin.Context) Logger {
	l.ensureContext()
	l.Entry.Context = context.WithValue(l.Entry.Context, keyGinContext, c)
	return l
}

func (l *logger) Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// broken pipe handling taken from gin Recovery source code
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				if brokenPipe {
					// If the connection is dead, we can't write a status to it.
					_ = c.Error(err.(error))
					c.Abort()
				} else {
					l.
						WithGin(c).
						WithField("panic", err).
						Errorf("recovered from panic in %q", c.FullPath())
					c.AbortWithStatus(http.StatusInternalServerError)
				}
			}
		}()
		c.Next()
	}
}
