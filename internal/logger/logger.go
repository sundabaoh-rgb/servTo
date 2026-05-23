package logger

type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
	With(kv ...any) Logger
	Named(name string) Logger
	Sync() error
}

type Config struct {
	Mode string
}

func New(cfg Config) (Logger, error) {
	switch cfg.Mode {
	case "dev", "development":
		return newZapDevLogger()
	case "prod", "production":
		return newZapProdLogger()
	default:
		return newZapDevLogger()
	}
}
