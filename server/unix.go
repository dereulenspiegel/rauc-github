package server

import (
	"github.com/dereulenspiegel/raucgithub"
	"github.com/spf13/viper"
)

func init() {
	RegisterBuilder("unix", UnixBuilder{})
}

type UnixBuilder struct{}

func (u UnixBuilder) Name() string {
	return "Unix Socket Server"
}

func (u UnixBuilder) ConfigKey() string {
	return "unixSocket"
}

func (u UnixBuilder) New(manager *raucgithub.UpdateManager, conf *viper.Viper) (Server, error) {
	return nil, nil
}
