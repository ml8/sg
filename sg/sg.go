package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"flag"

	"github.com/ml8/sg"
	"go.uber.org/zap"
)

var (
	indir   = flag.String("i", "", "input directory")
	outdir  = flag.String("o", "", "output directory")
	watch   = flag.Bool("w", false, "watch for changes (implies -s)")
	serve   = flag.Bool("s", false, "serve the output directory")
	verbose = flag.Bool("v", false, "enable verbose logging")
	port    = flag.Int("p", 8080, "port for the local server")
	drafts  = flag.Bool("drafts", false, "include draft pages in the rendered output")
	plain   = flag.Bool("plain", false, "disable colored output in watch mode")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sg [options]\n\n")
		fmt.Fprintf(os.Stderr, "A static site generator.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  sg -i site/ -o build/           Render site once\n")
		fmt.Fprintf(os.Stderr, "  sg -i site/ -o build/ -s        Render and serve locally\n")
		fmt.Fprintf(os.Stderr, "  sg -i site/ -o build/ -w        Watch for changes and serve\n")
		fmt.Fprintf(os.Stderr, "  sg -i site/ -o build/ -w -p 3000  Watch and serve on port 3000\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *indir == "" || *outdir == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *watch && *serve {
		fmt.Printf("-w implies -s\n")
	}

	var logger *zap.SugaredLogger
	if *verbose {
		l, _ := zap.NewProduction()
		logger = l.Sugar()
	}

	cfg := sg.Config{
		UseLocalRootUrl: *serve || *watch,
		InputDir:        *indir,
		OutputDir:       *outdir,
		QuiescentSecs:   10,
		Logger:          logger,
		Drafts:          *drafts || *serve || *watch,
		Port:            *port,
	}

	addr := fmt.Sprintf(":%d", *port)
	var srv *http.Server

	var lr *sg.LiveReload
	if *serve || *watch {
		lr = sg.NewLiveReload()
		mux := http.NewServeMux()
		mux.Handle("/ws", lr)
		mux.Handle("/", sg.InjectLiveReload(http.FileServer(http.Dir(cfg.OutputDir))))
		srv = &http.Server{
			Addr:    addr,
			Handler: mux,
		}
		go func() {
			fmt.Printf(" > Serving on http://localhost:%d\n", *port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Println("Server error:", err)
			}
		}()
	}

	// Handle graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if !*watch {
		r := sg.OfflineRenderer{
			Config: cfg,
		}

		if err := r.Render(); err != nil {
			fmt.Println("Errors rendering site:", err)
			os.Exit(1)
		} else {
			fmt.Println("ok")
			if lr != nil {
				lr.Reload()
			}
		}

		if *serve {
			fmt.Println("Press Ctrl+C to stop.")
			<-sigCh
			fmt.Println("\nShutting down...")
			if srv != nil {
				srv.Shutdown(context.Background())
			}
		}
	} else {
		r := sg.OnlineRenderer{
			Config: cfg,
		}
		if lr != nil {
			r.OnReload = lr.Reload
		}
		if *plain {
			r.Formatter = sg.NewPlainFormatter()
		} else {
			r.Formatter = sg.NewColorFormatter()
		}

		go func() {
			if err := r.Render(); err != nil {
				fmt.Println("Errors rendering site:", err)
				os.Exit(1)
			}
		}()

		<-sigCh
		fmt.Println("\nShutting down...")
		if srv != nil {
			srv.Shutdown(context.Background())
		}
	}
}
