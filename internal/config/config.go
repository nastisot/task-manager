package config

import (
	"fmt"
	"os"
)

type Config struct {
	MySQLDSN  string
	RedisAddr string
	JWTSecret string
	HTTPPort  string
}

func Load() Config {
	return Config{
		MySQLDSN: fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?parseTime=true",
			os.Getenv("MYSQL_USER"),
			os.Getenv("MYSQL_PASSWORD"),
			os.Getenv("MYSQL_HOST"),
			os.Getenv("MYSQL_PORT"),
			os.Getenv("MYSQL_DATABASE"),
		),
		RedisAddr: fmt.Sprintf(
			"%s:%s",
			os.Getenv("REDIS_HOST"),
			os.Getenv("REDIS_PORT"),
		),
		JWTSecret: os.Getenv("JWT_SECRET"),
		HTTPPort:  os.Getenv("HTTP_PORT"),
	}
}
