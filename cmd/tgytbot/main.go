package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"github.com/vm-affekt/tgytbot/internal/dialogs"
	"github.com/vm-affekt/tgytbot/internal/downloader"
	"github.com/vm-affekt/tgytbot/internal/logging"
	"github.com/vm-affekt/tgytbot/internal/telegram"
	"go.uber.org/zap"
)

const (
	modeEnvProduction = "prod"
	modeEnvDebug      = "debug"
)

func main() {
	var (
		debugMode bool

		logCfg zap.Config
	)

	viper.AddConfigPath("/etc/tgytbot")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("TGYTBOT")
	viper.SetConfigName("config")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Printf("%v. Environment variables will be used as config.\n", err)
		} else {
			panic(fmt.Errorf("failed to read config (used file: %q): %w", viper.ConfigFileUsed(), err))
		}
	}

	modeEnv := viper.GetString("MODE")
	logFilePath := viper.GetString("LOG_FILE_PATH")

	switch modeEnv {
	case modeEnvProduction:
		logCfg = zap.NewProductionConfig()

	case modeEnvDebug, "":
		modeEnv = modeEnvDebug
		debugMode = true
		logCfg = zap.NewDevelopmentConfig()
	default:
		fmt.Printf("ERROR! Unknown debug mode specified in MODE env var: '%s'. You can use only 'prod', 'debug' or leave this variable empty. Empty MODE will be treated as 'debug'!\n", modeEnv)
		os.Exit(1)
	}
	if logFilePath != "" {
		logCfg.OutputPaths = append(logCfg.OutputPaths, logFilePath)
	} else {
		fmt.Println("[WARN] No LOG_FILE_PATH specified! Using 'stderr' only.")
	}
	logger, err := logCfg.Build()
	if err != nil {
		panic(err)
	}
	logging.SetLogger(logger)
	log := logger.Sugar()

	log.Infof("[TELEGRAM YOUTUBE DOWNLOADER BOT] Application is running. Environment mode=%q", modeEnv)
	defer log.Sync()

	log.Infof("Used config file path: %v", viper.ConfigFileUsed())

	tgApiKey := viper.GetString("TELEGRAM_API_KEY")
	if tgApiKey == "" {
		log.Fatal("TELEGRAM_API_KEY can't be empty!")
	}

	downloadTimeout := viper.GetDuration("DOWNLOAD_TIMEOUT")
	if downloadTimeout == 0 {
		log.Warn("DOWNLOAD_TIMEOUT is zero!")
	}

	audioMaxFileSizeMB := viper.GetInt64("AUDIO_FILE_MAX_SIZE_MB")
	if audioMaxFileSizeMB == 0 {
		log.Warn("AUDIO_FILE_MAX_SIZE_MB is zero!")
	}

	downloadService := downloader.New(debugMode)

	container := dialogs.NewContainer(downloadService, downloadTimeout, audioMaxFileSizeMB)

	msgProc := telegram.NewMsgProcessor(viper.GetString("TELEGRAM_API_KEY"), debugMode, container)
	if err := msgProc.StartLongPolling(viper.GetInt32("TELEGRAM_LONG_POLLING_TIMEOUT")); err != nil {
		log.Fatalf("Failed to start long polling listener: %v", err)
	}

	log.Info("Long polling started. Bot is ready!")

	sigInt := make(chan os.Signal, 1)
	signal.Notify(sigInt, os.Interrupt, syscall.SIGTERM)
	shutSig := <-sigInt
	log.Infof("Signal received: %v. Shutdown server...", shutSig)
	// telegramSrv.Shutdown() TODO: SHUTDOWN
	log.Info("Shutdown work is over. Bye :-)")

}
