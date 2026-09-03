package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cdreier/kubeql/internal/kube"
	"github.com/cdreier/kubeql/internal/server"
	"github.com/urfave/cli/v2"
)

// Set by GoReleaser/Makefile: -ldflags "-X main.version=...".
var version = "dev"

func main() {
	app := &cli.App{
		Name:    "kubeql",
		Usage:   "Read-only GraphQL API over your local kubeconfig",
		Version: version,
		Commands: []*cli.Command{
			serveCommand(),
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func serveCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start the GraphQL server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Value: ":8080",
				Usage: "HTTP listen address",
			},
			&cli.StringFlag{
				Name: "kubeconfig",
				// Leave empty for client-go defaults, or pass an absolute path / ~/...
				// Note: ~ is expanded in code (not by Go's os.Stat itself).
				Usage:   "Path to kubeconfig (default: KUBECONFIG env, then $HOME/.kube/config)",
				Value:   "",
				EnvVars: []string{"KUBECONFIG"},
			},
			&cli.StringFlag{
				Name:  "context",
				Usage: "Kubeconfig context to use",
			},
			&cli.BoolFlag{
				Name:  "app",
				Usage: "Open a Chrome/Chromium app window and exit when it closes (stable localhost port unless --addr is set)",
			},
		},
		Action: func(c *cli.Context) error {
			// Parse kubeconfig (contexts) before creating any cluster client.
			reg, err := kube.OpenRegistry(c.String("kubeconfig"), c.String("context"))
			if err != nil {
				return fmt.Errorf("kubeconfig: %w", err)
			}
			svc, err := kube.NewServiceFromRegistry(reg)
			if err != nil {
				return fmt.Errorf("kubernetes client: %w", err)
			}
			router := server.NewRouter(svc)

			appMode := c.Bool("app")
			addr := c.String("addr")
			if appMode && !c.IsSet("addr") {
				addr = appListenAddr
			}

			ln, err := net.Listen("tcp", addr)
			if err != nil && appMode && !c.IsSet("addr") {
				log.Printf("app port %s busy (%v), falling back to a random port — favorites may not persist", addr, err)
				ln, err = net.Listen("tcp", "127.0.0.1:0")
			}
			if err != nil {
				return fmt.Errorf("listen %s: %w", addr, err)
			}
			url := publicURL(ln.Addr().String())

			if reg.HasActiveContext() {
				log.Printf("kubeql default context %q (%d contexts); clients are created per context on demand",
					reg.ActiveContext(), len(reg.Contexts()))
			} else {
				log.Printf("kubeql no default context (%d contexts); nest under contexts { ... } or pass context: on root fields",
					len(reg.Contexts()))
			}
			log.Printf("kubeql listening on %s (ui / · playground /playground · gql /query)", url)

			srv := &http.Server{Handler: router}
			serveErr := make(chan error, 1)
			go func() {
				err := srv.Serve(ln)
				if err != nil && err != http.ErrServerClosed {
					serveErr <- err
				}
				close(serveErr)
			}()

			if !appMode {
				return waitServe(serveErr)
			}
			return runAppMode(srv, url, serveErr)
		},
	}
}

func publicURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func waitServe(serveErr <-chan error) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		return nil
	}
}

// Stable loopback port so the Chrome origin (and localStorage favorites) stay put.
const appListenAddr = "127.0.0.1:17687"

func runAppMode(srv *http.Server, url string, serveErr <-chan error) error {
	cmd, err := startAppWindow(url)
	if err != nil {
		_ = shutdownHTTP(srv)
		return err
	}
	log.Printf("app window pid %d; closing it stops kubeql", cmd.Process.Pid)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()

	select {
	case err := <-serveErr:
		_ = cmd.Process.Kill()
		<-wait
		return err
	case err := <-wait:
		_ = shutdownHTTP(srv)
		if err != nil && cmd.ProcessState != nil && !cmd.ProcessState.Success() {
			return fmt.Errorf("chrome exited: %w", err)
		}
		return nil
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-wait
		return shutdownHTTP(srv)
	}
}

func shutdownHTTP(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
