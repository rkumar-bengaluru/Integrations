package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"go.uber.org/zap"
)

var defaultLogger *zap.Logger

type loggerCtx string

const loggerCtxKey loggerCtx = "logger"

func MillisecondDurationEncoder(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(fmt.Sprintf("%d Millisecond", d.Milliseconds()))
}

// NewFileLogger returns a production-grade Zap logger writing to a rotating file.
func newFileLogger() (*zap.Logger, error) {
	// Configure log rotation
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "./app.log",
		MaxSize:    100, // MB
		MaxBackups: 3,
		MaxAge:     28, // days
	})

	// JSON encoder for structured logs
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		w,
		zap.DebugLevel, // allow debug logs
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	return logger, nil
}

func init() {
	var cfg zap.Config
	log.Println("init logger with IS_LOCAL=", isLocal())
	if isLocal() {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeDuration = MillisecondDurationEncoder
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := newFileLogger()
	if err != nil {
		log.Fatal(err)
	}
	defaultLogger = logger
}

func GetTracers(ctx context.Context) (string, string) {
	// ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	return traceID, spanID
}

// ExtractTenantAndUserID extracts both tenant and user IDs from context
func ExtractTenantAndUserID(c *gin.Context) (uuid.UUID, string, error) {
	tenantID, err := ExtractTenantID(c)
	if err != nil {
		return uuid.Nil, "", err
	}

	userID, exists := ExtractUserID(c)
	if exists != nil {
		return uuid.Nil, "", fmt.Errorf("user ID not found in context")
	}

	return tenantID, userID, nil
}
func ExtractUserID(c *gin.Context) (string, error) {

	userVal, exists := c.Get("user_id")
	if !exists {
		return "", fmt.Errorf("error - user_id not found")
	}

	userID, ok := userVal.(string)
	if !ok {
		return "", fmt.Errorf("error - user_id not found")
	}

	return userID, nil
}

func ExtractTenantID(c *gin.Context) (uuid.UUID, error) {
	tenantVal, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("error - tenant_id not found")
	}

	tenantID, ok := tenantVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("error - tenant_id not found")
	}

	return tenantID, nil
}

func WithLoggerContext(c *gin.Context, base *zap.Logger) *zap.Logger {
	traceID, spanID := GetTracers(c.Request.Context())
	tenantID, _ := ExtractTenantID(c)
	userID, _ := ExtractUserID(c)

	return base.With(
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("tenant_id", tenantID.String()),
		zap.String("user_id", userID),
	)
}

func LoggerMiddleware(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := WithLoggerContext(c, base)
		ctx := context.WithValue(c.Request.Context(), "logger", logger)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func LoggerFromCtx(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value("logger").(*zap.Logger); ok {
		return l
	}
	return defaultLogger
}

// Gets the zap logger from context, creates a new one if does not exists
func Get(ctx context.Context) *zap.Logger {
	logger := logger(ctx)
	if logger == nil {
		return defaultLogger
	}
	return logger
}

func WithLogger(ctx context.Context, l *zap.Logger) context.Context {
	newCtx := context.WithValue(ctx, loggerCtxKey, l)
	return newCtx
}

func logger(ctx context.Context) *zap.Logger {
	logger := ctx.Value(loggerCtxKey)
	if logger == nil {
		return nil
	}
	return logger.(*zap.Logger)
}

func isLocal() bool {
	isLocal, err := strconv.ParseBool(os.Getenv("IS_LOCAL"))
	if err != nil {
		return false
	}
	return isLocal
}
