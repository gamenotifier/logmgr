package logmgr

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
)

// NopLogger is a logger which does nothing.
type NopLogger struct {
	*logrus.Entry
}

// NopLogger must implement these interfaces
var _ LoggerMaker = &NopLogger{}
var _ Logger = &NopLogger{}

// NewNopLogger creates a new NopLogger instance.
func NewNopLogger() *NopLogger {
	log := logrus.New()
	log.Out = io.Discard
	return &NopLogger{
		Entry: log.WithField("kind", "nop"),
	}
}

func (n *NopLogger) Extend(string) Logger {
	return n
}

func (n *NopLogger) WithUser(string) Logger {
	return n
}

func (n *NopLogger) WithGin(*gin.Context) Logger {
	return n
}

func (n *NopLogger) Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func (n *NopLogger) RecoveryWith(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if handler != nil {
			handler(c)
		}
		c.Next()
	}
}

func (n *NopLogger) NewLogger(string) Logger {
	return n
}
