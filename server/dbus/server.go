//go:build dbus

package dbus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dereulenspiegel/raucgithub"
	"github.com/dereulenspiegel/raucgithub/repository"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const intro = `
<node>
	<interface name="com.github.dereulenspiegel.rauc">
		<method name="NextUpdate">
			<arg direction="out" type="a{ss}"/>
		</method>
		<method name="InstallNextUpdateAsync">
		</method>
		<method name="Status">
			<arg direction="out" type="s"/>
		</method>
		<method name="Progress">
			<arg direction="out" type="i"/>
		</method>
		<signal name="UpdateAvailable">
			<arg name="update" type="a{ss}"/>
		</signal>
		<property name="AvailableUpdate" type="a{ss}" access="read"/>
	</interface>` + introspect.IntrospectDataString + `</node> `

type Server struct {
	conn       *dbus.Conn
	manager    *raucgithub.UpdateManager
	dbusCancel context.CancelFunc
	ctx        context.Context
	logger     logrus.FieldLogger

	useSessionBus bool
}

type Option func(*Server) *Server

func NewWithConfig(manager *raucgithub.UpdateManager, conf *viper.Viper) (*Server, error) {
	enabled := viper.GetBool("enabled")
	if !enabled {
		return nil, errors.New("dbus server not enabled")
	}
	return New(manager)
}

func New(manager *raucgithub.UpdateManager, opts ...Option) (s *Server, err error) {
	s = &Server{
		logger:  logrus.WithField("component", "dbusServer"),
		manager: manager,
	}
	for _, opt := range opts {
		s = opt(s)
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) (err error) {
	dbusContext, dbusCancel := context.WithCancel(ctx)
	s.dbusCancel = dbusCancel

	var conn *dbus.Conn
	if s.useSessionBus {
		conn, err = dbus.ConnectSessionBus(dbus.WithContext(dbusContext))
	} else {
		conn, err = dbus.ConnectSystemBus(dbus.WithContext(dbusContext))
	}
	if err != nil {
		return fmt.Errorf("failed to connect to DBus: %w", err)
	}

	s.conn = conn
	s.ctx = ctx
	if err := conn.Export(s, "/com/github/dereulenspiegel/rauc", "com.github.dereulenspiegel.rauc"); err != nil {
		return fmt.Errorf("failed to register DBus service: %w", err)
	}
	if err := conn.Export(introspect.Introspectable(intro), "/com/github/dereulenspiegel/rauc",
		"org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("failed to register DBus introspection: %w", err)
	}

	reply, err := conn.RequestName("com.github.dereulenspiegel.rauc",
		dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request name on system DBus: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return errors.New("name on system DBus already taken")
	}

	s.manager.RegisterUpdateAvailableCallback(s.updateAvailable)
	return nil
}

func (s *Server) Close() error {
	s.dbusCancel()
	return s.conn.Close()
}

func (s *Server) updateAvailable(update *repository.Update) {
	if err := s.conn.Emit("/com/github/dereulenspiegel/rauc", "com.github.dereulenspiegel.rauc.UpdateAvailable", mapFromUpdate(update)); err != nil {
		s.logger.WithError(err).Error("failed to emit DBus signal on new update")
	}
}

func mapFromUpdate(update *repository.Update) map[string]string {
	return map[string]string{
		"name":        update.Name,
		"notes":       update.Notes,
		"version":     update.Version.String(),
		"releaseDate": update.ReleaseDate.Format(time.RFC3339),
	}
}

func (s *Server) NextUpdate() (map[string]string, *dbus.Error) {
	update, err := s.manager.CheckForUpdate(s.ctx)
	if err != nil {
		return nil, dbus.MakeFailedError(err)
	}

	return mapFromUpdate(update), nil
}

func (s *Server) InstallNextUpdateAsync() *dbus.Error {
	progress, err := s.manager.InstallNextUpdateAsync(s.ctx, func(success bool, err error) {
		//Ignore, callback must not be empty
	})
	if err != nil {
		return dbus.MakeFailedError(err)
	}
	go func() {
		// Consume the channel
		<-progress
	}()
	return nil
}

func (s *Server) Status() (string, *dbus.Error) {
	status, err := s.manager.Status(s.ctx)
	if err != nil {
		return "", dbus.MakeFailedError(err)
	}
	return status.String(), nil
}

func (s *Server) Progress() (int32, *dbus.Error) {
	progress, err := s.manager.Progress(s.ctx)
	if err != nil {
		return -1, dbus.MakeFailedError(err)
	}
	return progress, nil
}
