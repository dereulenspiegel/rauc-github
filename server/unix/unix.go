package unix

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/dereulenspiegel/raucgithub"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type contextKey string

type SocketServer struct {
	manager *raucgithub.UpdateManager
	logger  logrus.FieldLogger
	ctx     context.Context

	socketPath string
	server     *http.Server
}

func New(manager *raucgithub.UpdateManager, conf *viper.Viper) (*SocketServer, error) {
	enabled := viper.GetBool("enabled")
	if !enabled {
		return nil, errors.New("unix socket server disabled")
	}
	socketPath := conf.GetString("socketPath")

	s := &SocketServer{
		socketPath: socketPath,
		manager:    manager,
		logger:     logrus.WithField("component", "unixSocketServer").WithField("socketPath", socketPath),
	}
	return s, nil
}

func (s *SocketServer) Start(ctx context.Context) error {
	httpServer := &http.Server{
		BaseContext: func(l net.Listener) context.Context {
			return context.WithValue(ctx, contextKey("source"), s.socketPath)
		},
	}

	unixListener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket %s: %w", s.socketPath, err)
	}
	s.server = httpServer
	s.ctx = ctx
	go s.listen(unixListener)
	return nil
}

func (s *SocketServer) Close() error {
	shutdownCtx, shutdownCancel := context.WithTimeout(s.ctx, time.Second*15)
	defer shutdownCancel()
	return s.server.Shutdown(shutdownCtx)
}

func (s *SocketServer) listen(l net.Listener) {
	if err := s.server.Serve(l); err != http.ErrServerClosed {
		s.logger.WithError(err).Error("failed to serve on unix socket")
	}
}
