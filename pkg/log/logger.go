package log

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/go-logr/zapr"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"

	"github.com/selebrow/selebrow/pkg/config"
	"github.com/selebrow/selebrow/pkg/kubeapi"
)

const (
	encodingJSON    = "json"
	encodingConsole = "console"
)

var (
	SetupLogger  = NewConsoleLogger
	inKubernetes = kubeapi.InKubernetes

	once   sync.Once
	logger *zap.Logger
)

func GetLogger() *zap.Logger {
	once.Do(func() {
		logger = SetupLogger()
		klog.SetLogger(zapr.NewLogger(logger))
	})
	return logger
}

func NewConsoleLogger() *zap.Logger {
	zc := zap.NewProductionConfig()
	lvl := getLogLevel()
	var opts []zap.Option
	if lvl >= zap.InfoLevel {
		zc.DisableStacktrace = true
		zc.DisableCaller = true
	}
	zc.Level = zap.NewAtomicLevelAt(lvl)

	zc.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	output := os.Getenv(fmt.Sprintf("%s_LOG_OUTPUT", config.ConfigPrefix))
	if output != "" {
		zc.OutputPaths = []string{output}
	}

	// Auto-switch to json logs when running in k8s
	if inKubernetes() || strings.ToLower(os.Getenv(fmt.Sprintf("%s_LOG_FORMAT", config.ConfigPrefix))) == encodingJSON {
		zc.Encoding = encodingJSON
		zc.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
		zc.EncoderConfig.TimeKey = "@timestamp"
		zc.EncoderConfig.MessageKey = "message"
		zc.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	} else {
		zc.Encoding = encodingConsole
		zc.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
		if output == "" {
			// Add color when debugging locally
			zc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			// Workaround for Windows terminal color output
			if runtime.GOOS == "windows" {
				opts = append(opts, zap.WrapCore(func(_ zapcore.Core) zapcore.Core {
					return zapcore.NewCore(
						zapcore.NewConsoleEncoder(zc.EncoderConfig),
						zapcore.AddSync(colorable.NewColorableStdout()),
						lvl,
					)
				}))
			}
		}
	}

	z, err := zc.Build(opts...)
	if err != nil {
		panic(err)
	}

	return z
}

func getLogLevel() zapcore.Level {
	strLevel := os.Getenv(fmt.Sprintf("%s_LOG_LEVEL", config.ConfigPrefix))
	return config.ZapLogLevel(strLevel, zap.InfoLevel)
}
