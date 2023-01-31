package server

import (
	"context"
	"io"
	"sync"

	"github.com/dereulenspiegel/raucgithub"
	"github.com/spf13/viper"
)

var (
	registryLock = &sync.Mutex{}
	registry     = make(map[string]Builder)
)

func RegisterBuilder(name string, b Builder) {
	registryLock.Lock()
	defer registryLock.Unlock()
	registry[name] = b
}

func Builders() (builders []Builder) {
	registryLock.Lock()
	defer registryLock.Unlock()
	for _, b := range registry {
		builders = append(builders, b)
	}
	return
}

type Server interface {
	io.Closer
	Start(context.Context) error
}
type Builder interface {
	ConfigKey() string
	Name() string
	New(*raucgithub.UpdateManager, *viper.Viper) (Server, error)
}
