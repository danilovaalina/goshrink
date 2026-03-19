package config

import (
	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
)

type Config struct {
	Addr        string `mapstructure:"addr"`
	DatabaseURL string `mapstructure:"db_url"`
	RedisURL    string `mapstructure:"redis_url"`
}

func Load() (Config, error) {
	v := viper.New()
	v.AddConfigPath(".")
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	v.AutomaticEnv()

	err := v.ReadInConfig()
	if err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return Config{}, errors.WithStack(err)
		}
	}

	var cfg Config
	if err = v.Unmarshal(&cfg); err != nil {
		return Config{}, errors.WithDetail(err, "unable to decode into struct")
	}

	return cfg, nil
}
