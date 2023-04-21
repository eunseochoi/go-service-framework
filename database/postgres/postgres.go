package postgres

import (
	"github.com/coherentopensource/go-service-framework/database"
	"github.com/coherentopensource/go-service-framework/util"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresDB(driver database.Driver, cfg *database.Config, logger util.Logger) (*database.Database, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		CreateBatchSize: cfg.CreateBatchSize,
	})
	if err != nil {
		return nil, err
	}

	return database.MustNewDB(db, driver, cfg, logger), nil
}
