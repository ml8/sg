package main

import (
	"net/http"

	"github.com/ml8/sg"

	"flag"
	"fmt"
	"os"
)

var (
	indir  = flag.String("i", "", "input directory")
	outdir = flag.String("o", "", "output directory")
	watch  = flag.Bool("w", false, "watch for changes")
	serve  = flag.Bool("s", false, "serve the output directory, only applicable in one-shot mode")
)

func main() {
	flag.Parse()

	if *indir == "" || *outdir == "" {
		fmt.Printf("Usage: sg -i <input directory> -o <output directory>\n")
		return
	}

	if *watch && *serve {
		fmt.Printf("-w implies -s\n")
	}

	cfg := sg.Config{
		UseLocalRootUrl: *serve || *watch,
		InputDir:        *indir,
		OutputDir:       *outdir,
		QuiescentSecs:   10,
	}

	if *serve || *watch {
		go func() {
			http.Handle("/", http.FileServer(http.Dir(cfg.OutputDir)))
			http.ListenAndServe(":8080", nil)
		}()
		fmt.Println(" > Serving on http://localhost:8080")
	}

	if !*watch {
		r := sg.OfflineRenderer{
			Config: cfg,
		}

		if err := r.Render(); err != nil {
			fmt.Println("Errors rendering site:", err)
			os.Exit(1)
		} else {
			fmt.Println("ok")
		}
	} else {
		r := sg.OnlineRenderer{
			Config: cfg,
		}

		if err := r.Render(); err != nil {
			fmt.Println("Errors rendering site:", err)
			os.Exit(1)
		}
		// On success, Render does not return.
	}

	if *serve {
		for {
		}
	}
}
