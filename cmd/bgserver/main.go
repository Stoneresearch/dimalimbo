package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stoneresearch/dimalimbo/internal/bgapi"
)

type reqBody struct {
	Prompt string `json:"prompt"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func loadEnvFiles(paths ...string) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if i := strings.IndexByte(line, '='); i > 0 {
				k := strings.TrimSpace(line[:i])
				v := strings.TrimSpace(line[i+1:])
				_ = os.Setenv(k, v)
			}
		}
		_ = f.Close()
	}
}

func main() {
	loadEnvFiles(".env.local", ".env") // prefer .env.local, then .env
	token := os.Getenv("REPLICATE_API_TOKEN")
	if token == "" {
		log.Println("warning: REPLICATE_API_TOKEN not set; requests will fail")
	}
	client := bgapi.NewClient(token, "black-forest-labs/flux-1.1-pro")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/background", func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var rb reqBody
		if json.NewDecoder(r.Body).Decode(&rb) != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if rb.Width == 0 {
			rb.Width = 1024
		}
		if rb.Height == 0 {
			rb.Height = 768
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		url, err := client.Generate(ctx, rb.Prompt, rb.Width, rb.Height)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
	})

	addr := ":8787"
	log.Printf("BG API server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
