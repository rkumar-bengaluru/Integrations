package repository

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ZapGormLogger struct {
	logger *zap.Logger
}

func NewZapGormLogger(logger *zap.Logger) *ZapGormLogger {
	return &ZapGormLogger{logger: logger}
}

func (z *ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	// You can adjust zap log level mapping here
	return z
}

func (z *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	z.logger.Info(msg, zap.Any("data", data))
}

func (z *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	z.logger.Warn(msg, zap.Any("data", data))
}

func (z *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	z.logger.Error(msg, zap.Any("data", data))
}

func (z *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	duration := time.Since(begin)

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", duration),
	}
	if err != nil {
		z.logger.Error("gorm trace", append(fields, zap.Error(err))...)
	} else {
		z.logger.Debug("gorm trace", fields...)
	}
}

// Store interface defines the contract for our database operations.
// It groups together interfaces for specific models.
type Store interface {
	// Add other model repositories here
}

// SQLStore is the concrete implementation of the Store interface.
// It holds the shared GORM DB client.
type SQLStore struct {
	db *gorm.DB
	// ... other concrete repository implementations
}

// NewSQLStore creates a new SQLStore instance.
// It initializes all individual model repositories with the shared DB client.
func NewSQLStore(dialect gorm.Dialector, zapLogger *zap.Logger) (Store, *gorm.DB) {
	db := getGormInstance(dialect, zapLogger)
	s := &SQLStore{db: db.instance}
	return s, db.instance
}

type GDBPointer struct {
	instance *gorm.DB
}

var (
	once   sync.Once
	holder *GDBPointer
)

func getGormInstance(dialect gorm.Dialector, zapLogger *zap.Logger) *GDBPointer {
	once.Do(func() {
		// Initialize GORM with PostgreSQL
		gormLogger := NewZapGormLogger(zapLogger)

		instance, err := gorm.Open(dialect, &gorm.Config{
			Logger: gormLogger,
		})
		if err != nil {
			zapLogger.Fatal("failed to connect database", zap.Error(err))
		}

		if err != nil {
			log.Fatalf("failed to connect database: %v", err)
		}

		// AutoMigrate the schema
		err = instance.AutoMigrate(
			&models.Credential{},
			&models.Integration{},
			&models.PlatformCredential{},
			&models.ActionDefinition{},
			&models.IntegrationBinding{},
		)

		if err != nil {
			fmt.Printf("AutoMigrate failed: %+v\n", err)

			log.Fatalf("failed to migrate schema: %v", err)
		}

		instance.Exec(`
			CREATE UNIQUE INDEX uniq_tenant_integration_user
			ON integration_bindings (tenant_id, integration_id, user_id)
			WHERE deleted_at IS NULL;
			`)

		gormLogger.Info(context.Background(), "db initialized...")
		holder = &GDBPointer{
			instance: instance,
		}
	})

	return holder
}
