package utils

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"

	"agent.fabric.com/modules/internal/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/gomail.v2"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewFileLogger returns a production-grade Zap logger writing to a rotating file.
func NewFileLogger() (*zap.Logger, error) {
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

func newFileLogger() *zap.Logger {
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "./app.log",
		MaxSize:    100, // MB
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	})

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		w,
		zap.InfoLevel,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	return logger
}

func ReadStr(ctx context.Context, key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		logger.Get(ctx).Fatal(key + " is missing")
	}
	return val
}

func ReadIntWithDefault(ctx context.Context, key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	valInt, err := strconv.Atoi(val)
	if err != nil {
		logger.Get(ctx).With(zap.Error(err)).Error(key + " parsing failed")
		return defaultVal
	}
	return valInt
}

func ReadInt64WithDefault(ctx context.Context, key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	valInt, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		logger.Get(ctx).With(zap.Error(err)).Error(key + " parsing failed")
		return defaultVal
	}
	return valInt
}

// sendForgotPasswordEmail sends the password reset email
func SendForgotPasswordEmail(email, token string) error {
	m := gomail.NewMessage()
	domainEmail := ReadStr(context.Background(), "DOMAIN_EMAIL")
	hostUrl := ReadStr(context.Background(), "DOMAIN_URL")
	smtpHost := ReadStr(context.Background(), "SMTP_HOST")
	smtpPassword := ReadStr(context.Background(), "SMTP_PASSWORD")
	smtpPort := ReadIntWithDefault(context.Background(), "SMTP_PORT", 587)
	m.SetHeader("From", domainEmail) // Replace with your email
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Forgot Password Request")
	m.SetBody("text/plain", fmt.Sprintf("Click this link to reset your password: %s/reset-password?token=%s", hostUrl, token))

	d := gomail.NewDialer(smtpHost, smtpPort, domainEmail, smtpPassword) // Replace with your SMTP settings

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func Ptr[T any](v T) *T {
	return &v
}

// PrintCollectedParams prints the key-value pairs of a map[string]interface{} in a sorted and readable format.
// Useful for debugging or logging collected parameters.
func PrintMap(params map[string]interface{}) {
	if len(params) == 0 {
		fmt.Println("Collected parameters: (empty)")
		return
	}

	fmt.Println("Collected parameters:")

	// Get sorted keys for consistent output
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := params[key]

		// Optional: special formatting for common types
		switch v := value.(type) {
		case string:
			fmt.Printf("  %-24s : %q\n", key, v)
		case int, int64, float64:
			fmt.Printf("  %-24s : %v\n", key, v)
		case bool:
			fmt.Printf("  %-24s : %t\n", key, v)
		case nil:
			fmt.Printf("  %-24s : <nil>\n", key)
		default:
			// For other types (structs, slices, maps, etc.) use %#v for more detail
			fmt.Printf("  %-24s : %#v\n", key, v)
		}
	}
}
