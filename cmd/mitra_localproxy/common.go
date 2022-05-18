package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mizosukedev/securetunnel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logPriorityDebug = 1
	logPriorityInfo  = 2
	logPriorityWarn  = 3
	logPriorityError = 4
)

var (
	// map[log level]priority
	logLevelMap = map[string]int{
		"debug": logPriorityDebug,
		"info":  logPriorityInfo,
		"warn":  logPriorityWarn,
		"error": logPriorityError,
	}
)

func setupLogger(logLevel string) error {

	config := zap.NewDevelopmentConfig()
	config.DisableStacktrace = true
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.ConsoleSeparator = " "
	config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	logger, err := config.Build()
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	logLevel = strings.ToLower(logLevel)
	priority, ok := logLevelMap[logLevel]
	if !ok {
		return fmt.Errorf("invalid log level %s", logLevel)
	}

	if priority <= logPriorityDebug {
		log.Debug = logger.Sugar().Debug
		log.Debugf = logger.Sugar().Debugf
	}

	if priority <= logPriorityInfo {
		log.Info = logger.Sugar().Info
		log.Infof = logger.Sugar().Infof
	}

	if priority <= logPriorityWarn {
		log.Warn = logger.Sugar().Warn
		log.Warnf = logger.Sugar().Warnf
	}

	if priority <= logPriorityError {
		log.Error = logger.Sugar().Error
		log.Errorf = logger.Sugar().Errorf
	}

	return nil
}

func applicationExit(handler func(os.Signal)) {

	stopSignals := []os.Signal{
		syscall.SIGTERM,
		syscall.SIGINT,
	}

	chSignal := make(chan os.Signal, 1)
	signal.Notify(chSignal, stopSignals...)

	go func() {
		sig := <-chSignal
		handler(sig)
	}()
}
