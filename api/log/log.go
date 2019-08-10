package log

import (
	"context"
	"log"
	"os"

	"github.com/yansal/youtube-ar/api/log/logfmt"
)

// Logger is a logger.
type Logger interface {
	Log(ctx context.Context, msg string, fields ...Field)
}

// New returns a new logger.
func New() Logger {
	return &logger{logger: log.New(os.Stdout, "", 0)}
}

type logger struct {
	logger *log.Logger
}

func (l *logger) Log(ctx context.Context, msg string, fields ...Field) {
	fields = append(fields, String("msg", msg))
	event := make(map[string]string, len(fields))
	for _, field := range fields {
		event[field.key] = field.value
	}
	b, err := logfmt.Marshal(event)
	if err != nil {
		panic(err)
	}
	l.logger.Printf("%s\n", b)
}
