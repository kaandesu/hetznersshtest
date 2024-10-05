package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/elapsed"
	"github.com/charmbracelet/wish/logging"
)

const (
	HOST = "0.0.0.0"
	PORT = "23234"
	PASS = "test"
)

//go:embed banner.txt
var banner string

type Server struct {
	ssh  *ssh.Server
	done chan os.Signal
}

func NewServer() *Server {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(HOST, PORT)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithPasswordAuth(func(ctx ssh.Context, password string) bool {
			return password == PASS
		}),
		wish.WithBannerHandler(func(ctx ssh.Context) string {
			return fmt.Sprintln(banner)
		}),
		wish.WithMiddleware(
			func(next ssh.Handler) ssh.Handler {
				return func(s ssh.Session) {
					wish.Printf(s, "Welcome you are in the system.")
					next(s)
				}
			},
			elapsed.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start the SSH server", "error", err)
	}

	return &Server{
		ssh: s,
	}
}

func (s *Server) Start() {
	var err error
	s.done = make(chan os.Signal, 1)

	log.Info("Starting the SSH server on", "host", HOST, "port", PORT)

	go func() {
		if err = s.ssh.ListenAndServe(); err != nil && !errors.Is(err, os.ErrClosed) {
			log.Error("Could not start the SSH server", "error", err)
		}
	}()

	signal.Notify(s.done, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-s.done

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = s.ssh.Shutdown(ctx); err != nil {
		log.Error("Could not stop the SSH server", "error", err)
	}
}

func main() {
	NewServer().Start()
}
