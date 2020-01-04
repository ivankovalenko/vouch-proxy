package cfg

import (
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	log    *zap.SugaredLogger
	atom   zap.AtomicLevel
)

func initLogger() {
	atom = zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()

	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))

	defer logger.Sync() // flushes buffer, if any
	log = logger.Sugar()
	Cfg.FastLogger = logger
	Cfg.Logger = log

}

func setLoglevel() {

	if Cfg.HealthCheck {
		// just errors for healthcheck, unless debug is set
		atom.SetLevel(zap.ErrorLevel)
		return
	}

	switch strings.ToLower(Cfg.LogLevel) {
	case "debug":
		atom.SetLevel(zap.DebugLevel)
	case "info":
		atom.SetLevel(zap.InfoLevel)
	case "warn":
		atom.SetLevel(zap.WarnLevel)
	case "error":
		atom.SetLevel(zap.ErrorLevel)
	}

}

func configureLogger() {
	setLoglevel()

	if Cfg.Testing {
		setDevelopmentLogger()
	}

}

func setDevelopmentLogger() {
	// then configure the logger for development output
	logger = logger.WithOptions(
		zap.WrapCore(
			func(zapcore.Core) zapcore.Core {
				return zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(os.Stderr), atom)
			}))
	*Cfg.Logger = *logger.Sugar()
	*Cfg.FastLogger = *Cfg.Logger.Desugar()
	log.Infof("testing: %s, using development console logger", strconv.FormatBool(Cfg.Testing))
}
