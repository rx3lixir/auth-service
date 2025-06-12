package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log - глобальный экземпляр логгера (SugaredLogger)
	Log *zap.SugaredLogger
	// RawLog - глобальный экземпляр обычного логгера (zap.Logger)
	RawLog *zap.Logger
	// используем ли мы быстрый немаршалированный логгер
	useRawLogger bool
)

// Logger представляет интерфейс для логирования
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
	Panic(msg string, args ...interface{})
}

// DefaultLogger реализует интерфейс Logger, оборачивая глобальные функции
type DefaultLogger struct{}

// Debug логирует с уровнем Debug
func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	Debug(msg, args...)
}

// Info логирует с уровнем Info
func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	Info(msg, args...)
}

// Warn логирует с уровнем Warn
func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	Warn(msg, args...)
}

// Error логирует с уровнем Error
func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	Error(msg, args...)
}

// Fatal логирует с уровнем Fatal
func (l *DefaultLogger) Fatal(msg string, args ...interface{}) {
	Fatal(msg, args...)
}

// Panic логирует с уровнем Panic
func (l *DefaultLogger) Panic(msg string, args ...interface{}) {
	Panic(msg, args...)
}

// NewLogger создает новый экземпляр логгера, который можно передавать в другие компоненты
func NewLogger() Logger {
	return &DefaultLogger{}
}

// Init инициализирует логгер на основе переданного окружения
// env: "prod" для продакшена, иначе используется development конфигурация
func Init(env string) {
	isProd := env == "prod"

	// Настраиваем логгер в зависимости от окружения
	if isProd {
		// В продакшен-окружении:
		// - Используем быстрый немаршалированный логгер
		// - Уровень логирования: InfoLevel
		// - Простой текстовый вывод без цветов
		useRawLogger = true
		initProductionLogger()
	} else {
		// В dev-окружении:
		// - Используем SugaredLogger
		// - Уровень логирования: DebugLevel
		// - Расширенный вывод с цветами
		useRawLogger = false
		initDevelopmentLogger()
	}
}

// initProductionLogger инициализирует оптимизированный логгер для продакшена
func initProductionLogger() {
	// Конфигурация для продакшен-окружения
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		FunctionKey:    zapcore.OmitKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	// Создаем JSON-энкодер для продакшена
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// Настройка вывода
	stdout := zapcore.AddSync(os.Stdout)
	stderr := zapcore.AddSync(os.Stderr)

	// Основной core для логирования с уровнем Info
	core := zapcore.NewCore(
		encoder,
		stdout,
		zap.NewAtomicLevelAt(zapcore.InfoLevel),
	)

	// Минимальный набор опций для производительности
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.ErrorOutput(stderr),
	}

	// Инициализируем логгер
	RawLog = zap.New(core, options...)
	Log = RawLog.Sugar()
}

// initDevelopmentLogger инициализирует расширенный логгер для разработки
func initDevelopmentLogger() {
	// Конфигурация для dev-окружения
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		FunctionKey:    zapcore.OmitKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	// Создаем JSON-энкодер для разработки
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// Настройка вывода
	stdout := zapcore.AddSync(os.Stdout)
	stderr := zapcore.AddSync(os.Stderr)

	// Основной core для логирования с уровнем Debug
	core := zapcore.NewCore(
		encoder,
		stdout,
		zap.NewAtomicLevelAt(zapcore.DebugLevel),
	)

	// Расширенный набор опций для dev-окружения
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.WarnLevel), // Стек с уровня Warning
		zap.Development(),                    // Настройки разработки
		zap.ErrorOutput(stderr),
	}

	// Инициализируем логгер
	RawLog = zap.New(core, options...)
	Log = RawLog.Sugar()
}

// Close закрывает логгер и освобождает ресурсы
func Close() error {
	if RawLog != nil {
		return RawLog.Sync()
	}
	return nil
}

// Debug логирует с уровнем Debug
func Debug(msg string, args ...interface{}) {
	if useRawLogger {
		if len(args) > 0 && len(args)%2 == 0 {
			fields := argsToFields(args)
			RawLog.Debug(msg, fields...)
		} else {
			RawLog.Debug(msg)
		}
	} else {
		Log.Debugw(msg, args...)
	}
}

// Info логирует с уровнем Info
func Info(msg string, args ...interface{}) {
	if useRawLogger {
		if len(args) > 0 && len(args)%2 == 0 {
			fields := argsToFields(args)
			RawLog.Info(msg, fields...)
		} else {
			RawLog.Info(msg)
		}
	} else {
		Log.Infow(msg, args...)
	}
}

// Warn логирует с уровнем Warn
func Warn(msg string, args ...interface{}) {
	if useRawLogger {
		if len(args) > 0 && len(args)%2 == 0 {
			fields := argsToFields(args)
			RawLog.Warn(msg, fields...)
		} else {
			RawLog.Warn(msg)
		}
	} else {
		Log.Warnw(msg, args...)
	}
}

// Error логирует с уровнем Error
func Error(msg string, args ...interface{}) {
	if useRawLogger {
		if len(args) > 0 && len(args)%2 == 0 {
			fields := argsToFields(args)
			RawLog.Error(msg, fields...)
		} else {
			RawLog.Error(msg)
		}
	} else {
		Log.Errorw(msg, args...)
	}
}

// Fatal логирует с уровнем Fatal и завершает программу с кодом 1
func Fatal(msg string, args ...interface{}) {
	if useRawLogger {
		if len(args) > 0 && len(args)%2 == 0 {
			fields := argsToFields(args)
			RawLog.Fatal(msg, fields...)
		} else {
			RawLog.Fatal(msg)
		}
	} else {
		Log.Fatalw(msg, args...)
	}
}

// Panic логирует с уровнем Panic и вызывает panic()
func Panic(msg string, args ...interface{}) {
	if useRawLogger {
		if len(args) > 0 && len(args)%2 == 0 {
			fields := argsToFields(args)
			RawLog.Panic(msg, fields...)
		} else {
			RawLog.Panic(msg)
		}
	} else {
		Log.Panicw(msg, args...)
	}
}

// argsToFields преобразует аргументы вида [key1, val1, key2, val2...] в поля zap.Field
func argsToFields(args []interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue // пропускаем если ключ не строка
		}

		// Преобразуем значение в zap.Field
		if i+1 < len(args) {
			fields = append(fields, zap.Any(key, args[i+1]))
		}
	}
	return fields
}
