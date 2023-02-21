package config

import "github.com/spf13/viper"

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

func GetUint(key string) uint {
	return viper.GetUint(key)
}

func GetUint16(key string) uint16 {
	return viper.GetUint16(key)
}

func GetString(key string) string {
	return viper.GetString(key)
}
