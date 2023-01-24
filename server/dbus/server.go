package dbus

import (
	"context"
	"fmt"
	"time"

	"github.com/dereulenspiegel/raucgithub"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const intro = `
<node>
	<interface name="com.github.dereulenspiegel.rauc">
		<method name="NextUpdate">
			<arg direction="out" type="{ss}"/>
		</method>
		<method name="InstallNextUpdateAsync">
		</method>
	</interface>` + introspect.IntrospectDataString + `</node> `

type Server struct {
	conn       *dbus.Conn
	manager    *raucgithub.UpdateManager
	dbusCancel context.CancelFunc
	ctx        context.Context
}

func Start(ctx context.Context, manager *raucgithub.UpdateManager) (*Server, error) {
	s := &Server{}
	dbusContext, dbusCancel := context.WithCancel(ctx)
	conn, err := dbus.ConnectSystemBus(dbus.WithContext(dbusContext))
	s.dbusCancel = dbusCancel
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system DBus: %w", err)
	}
	s.conn = conn
	s.manager = manager
	if err := conn.Export(s, "/com/github/dereulenspiegel/rauc", "com.github.dereulenspiegel.rauc"); err != nil {
		return nil, fmt.Errorf("failed to register DBus service: %w", err)
	}
	if err := conn.Export(introspect.Introspectable(intro), "/com/github/dereulenspiegel/rauc",
		"org.freedesktop.DBus.Introspectable"); err != nil {
		return nil, fmt.Errorf("failed to register DBus introspection: %w", err)
	}
	return s, nil
}

func (s *Server) Close() error {
	s.dbusCancel()
	return s.conn.Close()
}

func (s *Server) NextUpdate() (map[string]string, *dbus.Error) {
	update, err := s.manager.CheckForUpdate(s.ctx)
	if err != nil {
		return nil, dbus.MakeFailedError(err)
	}

	return map[string]string{
		"name":        update.Name,
		"notes":       update.Notes,
		"version":     update.Version.String(),
		"releaseDate": update.ReleaseDate.Format(time.RFC3339),
	}, nil
}

func (s *Server) InstallNextUpdateAsync() *dbus.Error {
	progress := s.manager.InstallNextUpdateAsync(s.ctx, func(success bool, err error) {
		//Ignore, callback must not be empty
	})
	go func() {
		// Consume the channel
		<-progress
	}()
	return nil
}
