package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/tomwright/mermaid-server/internal"
)

func main() {
	mermaid := flag.String("mermaid", "", "The full path to the mermaidcli executable.")
	in := flag.String("in", "", "Directory to store input files.")
	out := flag.String("out", "", "Directory to store output files.")
	puppeteer := flag.String("puppeteer", "", "Full path to optional puppeteer config.")
	addr := flag.String("addr", ":8080", "Address for the HTTP server to listen on.")
	allowAllOrigins := flag.Bool("allow-all-origins", false, "True to allow all request origins")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	var missing []string
	if *mermaid == "" {
		missing = append(missing, "mermaid")
	}
	if *in == "" {
		missing = append(missing, "in")
	}
	if *out == "" {
		missing = append(missing, "out")
	}
	if len(missing) > 0 {
		logger.Error("missing required arguments", "args", missing)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cache := internal.NewDiagramCache()
	generator := internal.NewGenerator(cache, *mermaid, *in, *out, *puppeteer, logger)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := internal.RunHTTPServer(ctx, logger, generator, *addr, *allowAllOrigins); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("http server stopped with error", "err", err)
		}
	}()

	go func() {
		defer wg.Done()
		internal.RunCleanup(ctx, logger, generator)
	}()

	wg.Wait()
}
