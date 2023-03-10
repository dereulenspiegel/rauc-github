package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/dereulenspiegel/raucgithub"
	"github.com/dereulenspiegel/raucgithub/repository/github"
	"github.com/dereulenspiegel/raucgithub/server"
)

func setDefaults() {
	viper.SetDefault("dbus.enabled", true)
}

var (
	version    string
	commit     string
	commitDate string
	builtBy    = "local"
)

func main() {
	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl, syscall.SIGTERM, syscall.SIGINT)

	ctx := context.Background()
	logger := logrus.WithFields(logrus.Fields{
		"version":    version,
		"commit":     commit,
		"commitDate": commitDate,
		"builtBy":    builtBy,
	})
	viper.SetConfigName("raucgithub")
	viper.AddConfigPath("/etc")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("RAUCGITHUB")
	viper.AutomaticEnv()
	setDefaults()
	logger.Info("Starting raucgithub")

	var contextCancels []context.CancelFunc
	var servers []server.Server
	defer func() {
		for _, cancelFunc := range contextCancels {
			cancelFunc()
		}
		for _, closer := range servers {
			closer.Close()
		}
	}()

	if err := viper.ReadInConfig(); err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	go func() {
		githubConfig := viper.Sub("repo.github")
		githubRepo, err := github.New(githubConfig)
		if err != nil {
			logger.WithError(err).Fatal("failed to create repository")
		}

		updateManagerConfig := viper.Sub("manager")
		manager, err := raucgithub.NewUpdateManagerFromConfig(githubRepo, updateManagerConfig)
		if err != nil {
			logger.WithError(err).Fatal("failed to create update manager")
		}

		serverBuilders := server.Builders()

		for _, builder := range serverBuilders {
			logger := logger.WithField("server", builder.Name())
			logger.Info("configuring server")
			serverConfig := viper.Sub(builder.ConfigKey())
			serverCtx, serverCancel := context.WithCancel(ctx)
			contextCancels = append(contextCancels, serverCancel)
			server, err := builder.New(manager, serverConfig)
			if err != nil {
				logger.WithError(err).Error("failed to configure server")
			}
			servers = append(servers, server)
			logger.Info("starting server")
			if err := server.Start(serverCtx); err != nil {
				logger.WithError(err).Error("failed to start server")
			}
		}

		logger.Info("Started successfully, waiting...")
	}()

	<-sigchnl
	logger.Info("Shutting down raucgithub")
	os.Exit(0)
}
