package oauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type CallbackResult struct {
	Code  string
	State string
	Err   string
}

type CallbackServer struct {
	server      *http.Server
	listener    net.Listener
	redirectURI string
	results     chan CallbackResult
}

func StartCallbackServer(port int, path string) (*CallbackServer, error) {
	if path == "" {
		path = "/auth/callback"
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	server := &CallbackServer{
		listener:    listener,
		redirectURI: fmt.Sprintf("http://localhost:%d%s", actualPort, path),
		results:     make(chan CallbackResult, 1),
	}
	mux := http.NewServeMux()
	mux.HandleFunc(path, server.handleCallback)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	server.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.server.Serve(listener) }()
	return server, nil
}

func (s *CallbackServer) RedirectURI() string { return s.redirectURI }

func (s *CallbackServer) Wait(ctx context.Context) (CallbackResult, error) {
	select {
	case <-ctx.Done():
		return CallbackResult{}, ctx.Err()
	case result := <-s.results:
		return result, nil
	}
}

func (s *CallbackServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	result := CallbackResult{
		Code:  query.Get("code"),
		State: query.Get("state"),
		Err:   query.Get("error"),
	}
	if result.Err != "" {
		http.Error(w, "Sign-in failed. Return to Apex.", http.StatusBadRequest)
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>Sign-in complete. Return to Apex.</body></html>"))
	}
	select {
	case s.results <- result:
	default:
	}
}
