package config

import (
	"fmt"
	"os"
	"strconv"
)

type configKey string

const (
	Debug            configKey = "DEBUG"
	TgBotToken       configKey = "TG_BOT_TOKEN"
	RedisAddress     configKey = "REDIS_ADDRESS"
	RedisPassword    configKey = "REDIS_PASSWORD"
	TextGeneratorURL configKey = "TEXT_GENERATOR_URL"
)

func GetValue(key configKey) string {
	return os.Getenv(string(key))
}

func GetInt(key configKey) int {
	val, err := strconv.Atoi(os.Getenv(string(key)))
	if err != nil {
		panic(fmt.Sprintf("%s env value is not integer", string(key)))
	}
	return val
}

func GetInt64(key configKey) int64 {
	val := GetInt(key)
	return int64(val)
}

func GetBool(key configKey) bool {
	val, found := os.LookupEnv(string(key))
	if val == "false" {
		return false
	}
	return found
}
