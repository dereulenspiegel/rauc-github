//go:build dbus

package server

import (
	"github.com/dereulenspiegel/raucgithub"
	"github.com/dereulenspiegel/raucgithub/server/dbus"
	"github.com/spf13/viper"
)

func init() {
	RegisterBuilder("dbus", DBusBuilder{})
}

type DBusBuilder struct{}

func (b DBusBuilder) Name() string {
	return "DBus"
}

func (b DBusBuilder) ConfigKey() string {
	return "dbus"
}

func (b DBusBuilder) New(manager *raucgithub.UpdateManager, conf *viper.Viper) (Server, error) {
	return dbus.NewWithConfig(manager, conf)
}
