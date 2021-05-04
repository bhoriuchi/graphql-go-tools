package logger

type Logger interface {
	Infof(format string, data ...interface{})
	Debugf(format string, data ...interface{})
	Errorf(format string, data ...interface{})
	Warnf(format string, data ...interface{})
}

type NoopLogger struct{}

func (n *NoopLogger) Infof(format string, data ...interface{})  {}
func (n *NoopLogger) Debugf(format string, data ...interface{}) {}
func (n *NoopLogger) Errorf(format string, data ...interface{}) {}
func (n *NoopLogger) Warnf(format string, data ...interface{})  {}
