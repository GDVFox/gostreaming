package util

// FakeLogger заглушка для sdk.Logger
type FakeLogger struct {
}

// Printf ничего не делает
func (l *FakeLogger) Printf(format string, args ...interface{}) {
}
