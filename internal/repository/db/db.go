package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rkumar-bengaluru/Integrations/internal/logger"
	"github.com/rkumar-bengaluru/Integrations/internal/utils"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	defaultMaxConns     = 10
	defaultIdleConns    = 5
	defaultMaxLifeTime  = 30
	defaultIdleLifeTime = 5
	minIdleConns        = 2
	minConnIdleTime     = 1
	defaultFactor       = 2
)

type Config struct {
	Host, Port, User, Password, DBName, Schema, SSLMode string
	MaxOpenConnections                                  int
	MaxIdleConnections                                  int
	MaxConnectionLifeTime                               int64
	MaxConnectionIdleTime                               int64
}

// POSTGRES_USER=myuser  -e POSTGRES_PASSWORD=mypassword -e POSTGRES_DB=mydb
func InitConfig(ctx context.Context) *Config {
	host := utils.ReadStr(ctx, "POSTGRES_HOST")
	port := utils.ReadStr(ctx, "POSTGRES_PORT")
	user := utils.ReadStr(ctx, "POSTGRES_USER")
	password := utils.ReadStr(ctx, "POSTGRES_PWD")
	dbname := utils.ReadStr(ctx, "POSTGRES_DBNAME")
	schema := utils.ReadStr(ctx, "POSTGRES_SCHEMA")
	sslMode := utils.ReadStr(ctx, "POSTGRES_SSLMODE")

	// host := "34.93.136.246"
	// port := "5432"
	// user := "arogya-dbuser"
	// password := "Pa55word@123"
	// dbname := "postgres"
	// sslMode := "disable"
	logger.Get(ctx).Info(fmt.Sprintf("user %v", user))
	return &Config{
		Host:                  host,
		Port:                  port,
		User:                  user,
		Password:              password,
		DBName:                dbname,
		Schema:                schema,
		SSLMode:               sslMode,
		MaxOpenConnections:    utils.ReadIntWithDefault(ctx, "POSTGRES_MAX_OPEN_CONNECTION", defaultMaxConns),
		MaxIdleConnections:    utils.ReadIntWithDefault(ctx, "POSTGRES_MAX_IDLE_CONNECTION", defaultIdleConns),
		MaxConnectionLifeTime: utils.ReadInt64WithDefault(ctx, "POSTGRES_MAX_CONNECTION_LIFE_TIME", defaultMaxLifeTime),
		MaxConnectionIdleTime: utils.ReadInt64WithDefault(ctx, "POSTGRES_MAX_CONNECTION_IDLE_TIME", defaultIdleLifeTime),
	}
}

func CreateDB(ctx context.Context, serviceName string) (*sqlx.DB, gorm.Dialector) {
	return CreateDBWithConfig(ctx, serviceName, InitConfig(ctx))
}

func (c *Config) normalize() {
	if c.MaxOpenConnections == 0 {
		c.MaxOpenConnections = defaultMaxConns
	}
	if c.MaxIdleConnections == 0 {
		c.MaxIdleConnections = defaultIdleConns
	}
	if c.MaxConnectionLifeTime == 0 {
		c.MaxConnectionLifeTime = defaultMaxLifeTime
	}
	if c.MaxConnectionIdleTime == 0 {
		c.MaxConnectionIdleTime = defaultIdleLifeTime
	}
	if c.MaxOpenConnections < c.MaxIdleConnections {
		c.MaxOpenConnections = c.MaxIdleConnections
		c.MaxIdleConnections /= defaultFactor
		if c.MaxIdleConnections < minIdleConns {
			c.MaxIdleConnections = minIdleConns
			c.MaxOpenConnections = defaultFactor * minIdleConns
		}
	}
	if c.MaxConnectionLifeTime < c.MaxConnectionIdleTime {
		c.MaxConnectionLifeTime = c.MaxConnectionIdleTime
		if c.MaxConnectionLifeTime < minConnIdleTime {
			c.MaxConnectionIdleTime = minConnIdleTime
			c.MaxConnectionLifeTime = defaultFactor * minConnIdleTime
		}
	}
}

func CreateDBWithConfig(ctx context.Context, serviceName string, postgresConfiguration *Config) (*sqlx.DB, gorm.Dialector) {
	postgresConfiguration.normalize()
	dsn := "host=" + postgresConfiguration.Host +
		" port=" + postgresConfiguration.Port +
		" user=" + postgresConfiguration.User +
		" password=" + postgresConfiguration.Password +
		" dbname=" + postgresConfiguration.DBName +
		" sslmode=" + postgresConfiguration.SSLMode +
		" search_path=public"

	db, err := sqlx.Connect("postgres", dsn)
	//db, err := sql.Open("postgres", dsn)
	logger.Get(ctx).With(zap.Error(err)).Info(fmt.Sprintf("connection string %v", dsn))
	if err != nil {
		logger.Get(ctx).With(zap.Error(err)).Panic("connection open pg failed")
	}
	db.SetMaxOpenConns(postgresConfiguration.MaxOpenConnections)
	db.SetMaxIdleConns(postgresConfiguration.MaxIdleConnections)
	db.SetConnMaxLifetime(time.Duration(postgresConfiguration.MaxConnectionLifeTime) * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(postgresConfiguration.MaxConnectionIdleTime) * time.Minute)
	dialect := postgres.Open(dsn)
	for i := 3; i >= 0; i-- {
		res, err := db.Exec("SELECT 1")
		if err == nil {
			//createTables(db)
			return db, dialect
		}
		logger.Get(ctx).With(zap.Error(err)).Error(fmt.Sprintf("res %v", res))
	}

	panic("DB connection failed")
}
