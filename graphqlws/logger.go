package graphqlws

import (
	"github.com/sirupsen/logrus"
)

// NewLogger returns a beautiful logger that logs messages with a
// given prefix (typically the name of a system component / subsystem).
func NewLogger(prefix string) *logrus.Entry {
	return logger.WithField("prefix", "graphqlws")
}
