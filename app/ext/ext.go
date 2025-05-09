package ext

type Logger interface {
	Debug(fmtStr string, vals ...any)
	Info(fmtStr string, vals ...any)
	Warning(fmtStr string, vals ...any)
	Error(fmtStr string, vals ...any)
}

type FakeLogger struct{}

func (f FakeLogger) Debug(fmtStr string, vals ...any) {
}

func (f FakeLogger) Info(fmtStr string, vals ...any) {
}

func (f FakeLogger) Warning(fmtStr string, vals ...any) {
}

func (f FakeLogger) Error(fmtStr string, vals ...any) {
}
