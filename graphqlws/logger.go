package graphqlws

type Logger interface {
	Infof(format string, data ...interface{})
	Debugf(format string, data ...interface{})
	Errorf(format string, data ...interface{})
	Warnf(format string, data ...interface{})
}

type noopLogger struct{}

func (n *noopLogger) Infof(format string, data ...interface{})  {}
func (n *noopLogger) Debugf(format string, data ...interface{}) {}
func (n *noopLogger) Errorf(format string, data ...interface{}) {}
func (n *noopLogger) Warnf(format string, data ...interface{})  {}
