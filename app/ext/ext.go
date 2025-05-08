package ext

type Logger interface {
	Debug(fmtStr string, vals ...any)
	Info(fmtStr string, vals ...any)
	Warning(fmtStr string, vals ...any)
	Error(fmtStr string, vals ...any)
}

type NullLogger struct{}

func (n NullLogger) Debug(fmtStr string, vals ...any) {
}

func (n NullLogger) Info(fmtStr string, vals ...any) {
}

func (n NullLogger) Warning(fmtStr string, vals ...any) {
}

func (n NullLogger) Error(fmtStr string, vals ...any) {
}
