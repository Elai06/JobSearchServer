package env

import (
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

type Config struct {
	HttpPort     string        `env:"HTTP_PORT" default:"8080"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" default:"10s"`
	IntParseSize int           `env:"INT_PARSE_SIZE" default:"10"`
	IntParseBase int           `env:"INT_PARSE_BASE" default:"10"`
	ClientId     string        `env:"CLIENT_ID" default:""`
	SecretKey    string        `env:"SECRET_KEY" default:""`
	DBUrl        string        `env:"DATA_BASE_URL" default:""`
}

func LoadConfig() (Config, error) {
	cfg := Config{}
	err := cleanenv.ReadConfig(".env", &cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}
