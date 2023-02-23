package config

import (
	"time"

	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/pg_pro/")
	viper.AddConfigPath("$HOME/.pg_pro")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

func Get(key string) any {
	return viper.Get(key)
}

func GetSlice(key string) []any {
	return viper.Get(key).([]any)
}

func GetStringSlice(key string) []string {
	return viper.Get(key).([]string)
}

func GetUint(key string) uint {
	return viper.GetUint(key)
}

func GetUint16(key string) uint16 {
	return viper.GetUint16(key)
}

func GetString(key string) string {
	return viper.GetString(key)
}

func GetInt32(key string) int32 {
	return viper.GetInt32(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetInt64(key string) int64 {
	return viper.GetInt64(key)
}

func GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}
