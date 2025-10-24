package logger

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/kardianos/service"
	"github.com/matthieu/mcp-server-prtg/internal/cliArgs"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps zerolog.Logger.
type Logger = zerolog.Logger

// Module log level registry for selective debugging.
var (
	moduleLogLevels    = make(map[string]zerolog.Level)
	activeDebugModules = make(map[string]bool)
	selectiveDebugMode = false
	moduleLock         sync.RWMutex
)

// Available modules for debugging.
const (
	ModuleConfiguration = "configuration"
	ModuleServer        = "server"
	ModuleDatabase      = "database"
	ModuleHandlers      = "handlers"
	ModuleAuth          = "auth"
	ModuleService       = "service"
)

// NewSilentLogger creates a logger that discards all output (for quiet operations).
func NewSilentLogger() *Logger {
	// Create logger that writes to discard writer
	logger := zerolog.New(io.Discard).Level(zerolog.Disabled)
	return &logger
}

// NewLogger creates a new logger instance based on CLI arguments.
func NewLogger(args *cliArgs.ParsedArgs) *Logger {
	// Determine log level
	var level zerolog.Level
	if args.Verbose {
		level = zerolog.DebugLevel
		if len(args.DebugModules) > 0 {
			// Selective debug mode
			selectiveDebugMode = true
			for _, module := range args.DebugModules {
				SetModuleLogLevel(module, zerolog.DebugLevel)
				activeDebugModules[module] = true
			}
		} else {
			// Full verbose mode
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
	} else {
		level = parseLogLevel(args.LogLevel)
		zerolog.SetGlobalLevel(level)
	}

	// Build logger based on environment
	if service.Interactive() {
		return buildDevelopmentLogger(args, level)
	}

	return buildProductionLogger(args, level)
}

// buildDevelopmentLogger creates a logger for development/console mode.
func buildDevelopmentLogger(_ *cliArgs.ParsedArgs, level zerolog.Level) *Logger {
	// Console writer with colors
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
	}

	// Wrap with masking
	maskingWriter := NewMaskingWriter(consoleWriter)

	logger := zerolog.New(maskingWriter).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	return &logger
}

// buildProductionLogger creates a logger for production/service mode.
func buildProductionLogger(args *cliArgs.ParsedArgs, level zerolog.Level) *Logger {
	// Ensure log directory exists
	logDir := filepath.Dir(args.LogFile)
	if err := os.MkdirAll(logDir, 0750); err != nil {
		// Fallback to stderr
		logger := zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()
		logger.Error().Err(err).Msg("Failed to create log directory, using stderr")
		return &logger
	}

	// Configure log rotation
	logRotator := &lumberjack.Logger{
		Filename:   args.LogFile,
		MaxSize:    10,   // Megabytes
		MaxBackups: 5,    // Number of backups
		MaxAge:     30,   // Days
		Compress:   true, // Enable compression
	}

	// Multiple outputs
	writers := []io.Writer{NewMaskingWriter(logRotator)}

	// Add console in interactive mode
	if service.Interactive() {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		}
		writers = append(writers, NewMaskingWriter(consoleWriter))
	}

	// Multi-writer
	logWriter := zerolog.MultiLevelWriter(writers...)

	logger := zerolog.New(logWriter).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &logger
}

// parseLogLevel converts string to zerolog.Level.
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// SetModuleLogLevel sets the log level for a specific module.
func SetModuleLogLevel(module string, level zerolog.Level) {
	moduleLock.Lock()
	defer moduleLock.Unlock()
	moduleLogLevels[module] = level
}

// GetModuleLogLevel gets the log level for a specific module.
func GetModuleLogLevel(module string) zerolog.Level {
	moduleLock.RLock()
	defer moduleLock.RUnlock()

	if level, ok := moduleLogLevels[module]; ok {
		return level
	}

	return zerolog.GlobalLevel()
}

// ModuleLogger wraps zerolog.Logger with per-module level control.
type ModuleLogger struct {
	*zerolog.Logger
	module         string
	selectiveMode  bool
	enabledModules map[string]bool
}

// NewModuleLogger creates a logger with module-specific configuration.
func NewModuleLogger(baseLogger *Logger, module string) *ModuleLogger {
	logger := baseLogger.With().Str("module", module).Logger()

	moduleLock.RLock()
	enabledModules := copyMap(activeDebugModules)
	moduleLock.RUnlock()

	return &ModuleLogger{
		Logger:         &logger,
		module:         module,
		selectiveMode:  selectiveDebugMode,
		enabledModules: enabledModules,
	}
}

// Debug returns a debug event, filtered by module in selective mode.
func (m *ModuleLogger) Debug() *zerolog.Event {
	// In selective debug mode, filter by module
	if m.selectiveMode {
		if _, enabled := m.enabledModules[m.module]; !enabled {
			disabledLogger := m.Logger.Level(zerolog.Disabled)
			return disabledLogger.Debug()
		}
	}

	if GetModuleLogLevel(m.module) <= zerolog.DebugLevel {
		return m.Logger.Debug()
	}

	disabledLogger := m.Logger.Level(zerolog.Disabled)
	return disabledLogger.Debug()
}

// copyMap creates a copy of a map.
func copyMap(original map[string]bool) map[string]bool {
	copy := make(map[string]bool, len(original))
	for k, v := range original {
		copy[k] = v
	}
	return copy
}
