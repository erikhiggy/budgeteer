// Command budgeteer runs the budgeting app on localhost.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"budgeteer/internal/api"
	"budgeteer/internal/db"
	"budgeteer/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:7420", "address to listen on (keep it on loopback)")
	dbPath := flag.String("db", "", "SQLite database file (default ~/Library/Application Support/Budgeteer/budget.db)")
	flag.Parse()

	if err := run(*addr, *dbPath); err != nil {
		log.Fatal(err)
	}
}

func run(addr, dbPath string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if dbPath == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		dbPath = filepath.Join(dir, "Budgeteer", "budget.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return err
	}

	conn, err := db.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Listen before logging so the printed URL is known to work.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           api.New(conn, web.Dist()),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("database: %s", dbPath)
	log.Printf("listening on http://%s", ln.Addr())

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
