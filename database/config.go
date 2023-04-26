package database

import "strings"

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
	return strings.Join([]string{
		"host=", c.DBHost,
		" user=", c.DBUser,
		" password=", c.DBPassword,
		" dbname=", c.DBName,
		" port=", c.DBPort,
		" sslmode=", c.SSLMode,
	}, "")
}
