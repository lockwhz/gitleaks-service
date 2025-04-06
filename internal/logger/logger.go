package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	once   sync.Once
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

var (
	AppName = "CloneScanService"
	Env     = "production"
	LogPath = "logs/app.log"
)

func Init() error {
	once.Do(func() {
		encoderCfg := zap.NewProductionEncoderConfig()
		encoderCfg.TimeKey = "timestamp"
		encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderCfg.CallerKey = "caller"
		encoderCfg.LevelKey = "level"
		encoderCfg.MessageKey = "message"
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

		consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)

		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   LogPath,
			MaxSize:    50,
			MaxBackups: 7,
			MaxAge:     30,
			Compress:   true,
		})

		logLevel := zapcore.DebugLevel

		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), logLevel),
			zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), fileWriter, logLevel),
		)

		logger = zap.New(core,
			zap.AddCaller(),
			zap.AddStacktrace(zapcore.ErrorLevel),
			zap.Fields(
				zap.String("app", AppName),
				zap.String("env", Env),
			),
		)

		sugar = logger.Sugar()
	})
	return nil
}

func GetLogger() *zap.Logger {
	Init()
	return logger
}

func GetSugaredLogger() *zap.SugaredLogger {
	Init()
	return sugar
}

func Trace(fn string, start time.Time) {
	elapsed := time.Since(start)
	sugar.Debugf("%s executed in %d ms", fn, elapsed.Milliseconds())
}

func TraceAuto() func() {
	start := time.Now()
	pc, _, _, ok := runtime.Caller(1)
	funcName := "unknown"
	if ok {
		fullName := runtime.FuncForPC(pc).Name()
		funcName = trimPackagePath(fullName)
	}
	sugar.Infow("Início da função", "function", funcName, "start", time.Now().Format(time.RFC3339Nano))
	return func() {
		elapsed := time.Since(start)
		sugar.Infow("Fim da função", "function", funcName, "duration", elapsed.String())
	}
}

func trimPackagePath(fullName string) string {
	if idx := strings.LastIndex(fullName, "/"); idx != -1 {
		fullName = fullName[idx+1:]
	}
	if idx := strings.Index(fullName, "."); idx != -1 {
		return fullName[idx+1:]
	}
	return fullName
}
