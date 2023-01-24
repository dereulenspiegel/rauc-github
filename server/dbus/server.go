package dbus

import (
	"context"
	"errors"
	"fmt"

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
	conn    *dbus.Conn
	manager *raucgithub.UpdateManager
}

func Start(ctx context.Context, manager *raucgithub.UpdateManager) (*Server, error) {
	s := &Server{}
	conn, err := dbus.ConnectSystemBus(dbus.WithContext(ctx))
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
	return s.conn.Close()
}

func (s *Server) NextUpdate() (map[string]string, *dbus.Error) {
	return nil, dbus.MakeFailedError(errors.New("not implemented"))
}

func (s *Server) InstallNextUpdateAsync() *dbus.Error {
	return dbus.MakeFailedError(errors.New("not implemented"))
}
