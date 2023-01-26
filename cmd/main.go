package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/dereulenspiegel/raucgithub"
	"github.com/dereulenspiegel/raucgithub/repository/github"
	"github.com/dereulenspiegel/raucgithub/server/dbus"
)

func setDefaults() {
	viper.SetDefault("dbus.enabled", true)
}

func main() {
	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl, syscall.SIGTERM, syscall.SIGINT)

	ctx := context.Background()
	logger := logrus.WithField("app", "raucgithub")
	viper.SetConfigName("raucgithub")
	viper.AddConfigPath("/etc")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("RAUCGITHUB")
	viper.AutomaticEnv()

	var contextCancels []context.CancelFunc
	var closers []io.Closer
	defer func() {
		for _, cancelFunc := range contextCancels {
			cancelFunc()
		}
		for _, closer := range closers {
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
		manager, err := raucgithub.NewUpdateManager(githubRepo)
		if err != nil {
			logger.WithError(err).Fatal("failed to create update manager")
		}

		dbusConfig := viper.Sub("dbus")
		dbusCtx, dbusCancel := context.WithCancel(ctx)
		contextCancels = append(contextCancels, dbusCancel)
		dbusServer, err := dbus.StartWithConfig(dbusCtx, manager, dbusConfig)
		if err != nil {
			logger.WithError(err).Error("failed to start dbus server")
		} else {
			closers = append(closers, dbusServer)
		}
	}()

	<-sigchnl
	logger.Info("Shutting down raucgithub")
	os.Exit(0)
}
