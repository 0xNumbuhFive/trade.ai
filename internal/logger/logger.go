package logger

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// CustomLogger wraps logrus to add custom formatting
type CustomLogger struct {
	*logrus.Logger
}

// New creates a new logger instance
func New(level string, logFile string) (*CustomLogger, error) {
	logger := logrus.New()

	// Parse log level
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	logger.SetLevel(lvl)

	// Set formatter
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := f.File
			line := f.Line
			return "", fmt.Sprintf("%s:%d", filename, line)
		},
	})

	// Output to file if specified
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.SetOutput(file)
	} else {
		logger.SetOutput(os.Stdout)
	}

	return &CustomLogger{logger}, nil
}

// LogOrder logs order-related information
func (l *CustomLogger) LogOrder(orderType string, symbol string, quantity float64, price float64, status string) {
	l.WithFields(logrus.Fields{
		"type":     orderType,
		"symbol":   symbol,
		"qty":      quantity,
		"price":    price,
		"status":   status,
		"action":   "order",
	}).Info("Order update")
}

// LogSignal logs trading signals
func (l *CustomLogger) LogSignal(signal string, symbol string, reason string) {
	l.WithFields(logrus.Fields{
		"signal":  signal,
		"symbol":  symbol,
		"reason":  reason,
		"action": "signal",
	}).Info("Trading signal generated")
}

// LogIndicator logs indicator values
func (l *CustomLogger) LogIndicator(symbol string, rsi float64, macd float64, macdSignal float64, ema float64) {
	l.WithFields(logrus.Fields{
		"symbol":      symbol,
		"rsi":         rsi,
		"macd":        macd,
		"macd_signal": macdSignal,
		"ema":         ema,
		"timestamp":  time.Now().Unix(),
		"action":      "indicators",
	}).Debug("Indicator values")
}

// LogError logs errors with context
func (l *CustomLogger) LogError(err error, context string) {
	l.WithFields(logrus.Fields{
		"context": context,
		"action":  "error",
	}).Errorf("%v", err)
}

// LogInfo logs general information
func (l *CustomLogger) LogInfo(message string, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["action"] = "info"
	l.WithFields(fields).Info(message)
}
