package postgres

import (
	"github.com/coherentopensource/go-service-framework/util"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"strings"
	"time"
)

type Config struct {
	DBHost           string `env:"DB_HOST,required"`
	DBPassword       string `env:"DB_PASSWORD,required"`
	DBUser           string `emv:"DB_USER,required"`
	DBName           string `env:"DB_NAME,required"`
	DBPort           string `env:"DB_PORT,required"`
	SSLMode          string `env:"SSL_MODE,required"`
	CreateBatchSize  int    `env:"CREATE_BATCH_SIZE" envDefault:"2000"`
	ConnectionsLimit int    `env:"CONNECTIONS_LIMIT" envDefault:"1000"`
}

func (c *Config) DSN() string {
	return strings.Join([]string{
		"host=", c.DBHost,
		" user=", c.DBUser,
		" password=", c.DBPassword,
		" dbname=", c.DBName,
		" port=", c.DBPort,
		" sslmode=", c.SSLMode,
	}, "")
}

type DB struct {
	connection *gorm.DB
	config     *Config
	logger     util.Logger
}

func NewDB(cfg *Config, logger util.Logger) (*DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		CreateBatchSize: cfg.CreateBatchSize,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	logger.Infof("db initialized with connection limit: %d", cfg.ConnectionsLimit)
	sqlDB.SetMaxOpenConns(cfg.ConnectionsLimit)
	sqlDB.SetMaxIdleConns(cfg.ConnectionsLimit)
	sqlDB.SetConnMaxLifetime(time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Minute)

	return &DB{
		connection: db,
		config:     cfg,
		logger:     logger,
	}, nil
}

func MustNewDB(cfg *Config, logger util.Logger) *DB {
	db, err := NewDB(cfg, logger)
	if err != nil {
		logger.Fatalf("failed to initialize db: %v", err)
	}
	return db
}
