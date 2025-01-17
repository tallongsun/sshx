package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/suutaku/sshx/internal/node"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	res = map[string]chan node.ConnectInfo{}
	mu  sync.RWMutex
)

func main() {
	http.Handle("/pull/", http.StripPrefix("/pull/", pullData()))
	http.Handle("/push/", http.StripPrefix("/push/", pushData()))

	port := os.Getenv("SSHX_SIGNALING_PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func pushData() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var info node.ConnectInfo
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			log.Print("json decode failed:", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		log.Println("push from ", info.Source, r.URL.Path)
		mu.Lock()
		defer mu.Unlock()
		select {
		case res[r.URL.Path] <- info:
		}
	})
}

func pullData() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ch := res[r.URL.Path]
		if ch == nil {
			mu.Lock()
			ch = make(chan node.ConnectInfo, 64)
			res[r.URL.Path] = ch
			mu.Unlock()
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		select {
		case <-ctx.Done():
			http.Error(w, ``, http.StatusRequestTimeout)
			return
		case v := <-ch:
			log.Println("pull from ", r.URL.Path, v.Source)
			w.Header().Add("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(v); err != nil {
				log.Print("json encode failed:", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}
	})
}
