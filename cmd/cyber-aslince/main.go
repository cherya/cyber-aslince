package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cherya/cyber-aslince/internal/aslince"
	"github.com/cherya/cyber-aslince/internal/config"
	"github.com/cherya/cyber-aslince/internal/logger"

	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gographics/imagick.v2/imagick"
	tb "gopkg.in/tucnak/telebot.v2"
)

const logDateFormat = "02-01-2006 15:04:05"

func main() {
	logger.Init(log.DebugLevel, logDateFormat)

	initEnv()

	imagick.Initialize()
	defer imagick.Terminate()

	var redisPool = &redis.Pool{
		MaxActive: 5,
		MaxIdle:   5,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial(
				"tcp",
				config.GetValue(config.RedisAddress),
				redis.DialPassword(config.GetValue(config.RedisPassword)),
			)
		},
	}

	b, err := tb.NewBot(tb.Settings{
		URL:    "https://api.telegram.org",
		Token:  config.GetValue(config.TgBotToken),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}
	oslica := aslince.NewAslince(redisPool, *b, config.GetValue(config.TextGeneratorURL))
	stop := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)
		signal.Notify(sigint, syscall.SIGKILL)

		sig := <-sigint
		log.Infof("caught signal %+v", sig)

		if err := oslica.Shutdown(); err != nil {
			log.Errorf("Aslince shutdown: %v", err)
		}
		close(stop)
		log.Info("gracefully stopped")
	}()

	oslica.Start()
	<-stop
}

func initEnv() {
	env := flag.String("env", "local.env", "env file with config values")
	flag.Parse()
	log.Infof("Loading env from %s", *env)
	err := godotenv.Load(*env)

	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	if config.GetBool(config.Debug) {
		logEnv(env)
	}
}

func logEnv(env *string) {
	envMap, _ := godotenv.Read(".env", *env)
	for key, val := range envMap {
		log.Infof("%s = %s", key, val)
	}
}
