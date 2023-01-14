package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type httpServer struct {
}

func main() {
	err := run()
	if err != nil {
		fmt.Println("failed:", err)
		os.Exit(1)
	}

}

func run() error {
	server := &httpServer{}
	r := mux.NewRouter()
	r.HandleFunc("/", server.handleGet).Methods("GET")
	addr := ":8080"
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	fmt.Println("starting listener on", addr)
	return srv.ListenAndServe()
}

func (s *httpServer) handleGet(w http.ResponseWriter, r *http.Request) {
	session := r.URL.Query().Get("session")
	type Resp struct {
		Session string
		Message string
		Board   string
	}
	resp := &Resp{
		Session: session,
	}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
