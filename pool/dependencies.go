package pool

type Logger interface {
	Error(...interface{})
	Info(...interface{})
	Fatal(...interface{})
	Panic(...interface{})
	Warn(...interface{})
	Errorf(string, ...interface{})
	Infof(string, ...interface{})
	Fatalf(string, ...interface{})
	Panicf(string, ...interface{})
	Warnf(string, ...interface{})
}
