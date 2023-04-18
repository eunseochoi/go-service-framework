package database

import (
	"github.com/coherentopensource/go-service-framework/util"
	"gorm.io/gorm"
	"time"
)

type Database struct {
	Connection *gorm.DB
	Config     *Config
	Logger     util.Logger
	driver     Driver
}

func MustNewDB(db *gorm.DB, driver Driver, cfg *Config, logger util.Logger) *Database {
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatalf("failed to initialize db: %v", err)
	}
	logger.Infof("db initialized with connection limit: %d", cfg.ConnectionsLimit)
	sqlDB.SetMaxOpenConns(cfg.ConnectionsLimit)
	sqlDB.SetMaxIdleConns(cfg.ConnectionsLimit)
	sqlDB.SetConnMaxLifetime(time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Minute)

	return &Database{
		db,
		cfg,
		logger,
		driver,
	}
}

func (db *Database) Migrate(models ...interface{}) error {
	return db.Connection.AutoMigrate(models...)
}

func (db *Database) Close() error {
	sqlDB, err := db.Connection.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func (db *Database) Upsert(object interface{}, model interface{}) error {
	return db.driver.Upsert(object, model)
}

func (db *Database) UpsertBatch(objects []interface{}, model interface{}) error {
	return db.driver.UpsertBatch(objects, model)
}

func (db *Database) Find(object interface{}, model interface{}) ([]interface{}, error) {
	return db.driver.Find(object, model)
}

func (db *Database) Delete(object interface{}, model interface{}) error {
	return db.driver.Delete(object, model)
}
