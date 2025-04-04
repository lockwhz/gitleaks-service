package logger

import (
	"time"

	"go.uber.org/zap"
)

var Log *zap.SugaredLogger

func Init() error {
	l, err := zap.NewProduction()
	if err != nil {
		return err
	}
	Log = l.Sugar()
	return nil
}

func Trace(fn string, start time.Time) {
	elapsed := time.Since(start)
	Log.Debugf("%s executed in %d ms", fn, elapsed.Milliseconds())
}
