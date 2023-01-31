package unix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	logrusLogger "github.com/chi-middleware/logrus-logger"
	"github.com/dereulenspiegel/raucgithub"
	"github.com/dereulenspiegel/raucgithub/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	r := chi.NewRouter()
	httpServer := &http.Server{
		BaseContext: func(l net.Listener) context.Context {
			return context.WithValue(ctx, contextKey("source"), s.socketPath)
		},
		Handler: r,
	}
	r.Use(middleware.CleanPath)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))
	r.Use(logrusLogger.Logger("router", s.logger))

	r.Route("/update", func(r chi.Router) {
		r.Get("/check", s.checkUpdate)
		r.Get("/status", s.status)
	})

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

type updateResponse struct {
	Version     string    `json:"version"`
	ReleaseDate time.Time `json:"releaseDate"`
	Name        string    `json:"name"`
	Notes       string    `json:"notes"`
	Prerelease  bool      `json:"prerelease"`
}

func responseFromUpdate(update *repository.Update) updateResponse {
	return updateResponse{
		Version:     update.Version.String(),
		ReleaseDate: update.ReleaseDate,
		Name:        update.Name,
		Notes:       update.Notes,
		Prerelease:  update.Prerelease,
	}
}

func (s *SocketServer) checkUpdate(w http.ResponseWriter, r *http.Request) {
	status, err := s.manager.Status(r.Context())
	if err != nil {
		panic(err)
	}
	if status != raucgithub.StatusIdle {
		writeError(errors.New("update service is not idle"), http.StatusConflict, w)
		return
	}

	update, err := s.manager.CheckForUpdate(r.Context())
	if err == raucgithub.ErrNoSuitableUpdate {
		writeError(errors.New("no update"), http.StatusNotFound, w)
		http.Error(w, "no update", http.StatusNotFound)
		return
	} else if err != nil {
		panic(err)
	}

	resp := responseFromUpdate(update)

	writeResponse(resp, http.StatusOK, w)
}

type statusResponse struct {
	Status     string          `json:"status"`
	Progress   *int32          `json:"progress,omitempty"`
	NextUpdate *updateResponse `json:"nextUpdate,omitempty"`
}

func (s *SocketServer) status(w http.ResponseWriter, r *http.Request) {
	status, err := s.manager.Status(r.Context())
	if err != nil {
		panic(err)
	}
	resp := statusResponse{
		Status: status.String(),
	}
	if status == raucgithub.StatusInstalling {
		progress, err := s.manager.Progress(r.Context())
		if err != nil {
			panic(err)
		}
		resp.Progress = &progress
	} else {
		update, err := s.manager.CheckForUpdate(r.Context())
		if err == nil {
			updateResp := responseFromUpdate(update)
			resp.NextUpdate = &updateResp
		} else if err != nil && err != raucgithub.ErrNoSuitableUpdate {
			panic(err)
		}
	}

	writeResponse(resp, http.StatusOK, w)
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError[T error](err T, status int, w http.ResponseWriter) {
	errRsp := errorResponse{
		Error: err.Error(),
	}
	writeResponse(errRsp, status, w)
}

func writeResponse[T any](resp T, status int, w http.ResponseWriter) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}
