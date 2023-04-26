package database

import (
	"fmt"
)

type Config struct {
	DBHost           string `env:"DB_HOST,required"`
	DBPassword       string `env:"DB_PASSWORD,required"`
	DBUser           string `env:"DB_USER,required"`
	DBName           string `env:"DB_NAME,required"`
	DBPort           string `env:"DB_PORT,required"`
	SSLMode          string `env:"SSL_MODE,required"`
	CreateBatchSize  int    `env:"CREATE_BATCH_SIZE" envDefault:"2000"`
	ConnectionsLimit int    `env:"CONNECTIONS_LIMIT" envDefault:"1000"`
}

func (c *Config) DSN() string {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		c.DBHost,
		c.DBUser,
		c.DBPassword,
		c.DBName,
		c.DBPort,
		c.SSLMode,
	)
	return dsn
}
