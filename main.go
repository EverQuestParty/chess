package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/everquestparty/chess/board"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Game struct {
	Position board.Position
	Searcher *board.Searcher
}

type httpServer struct {
	mux   sync.RWMutex
	games map[string]*Game
}

func main() {
	err := run()
	if err != nil {
		fmt.Println("failed:", err)
		os.Exit(1)
	}

}

func run() error {
	server := &httpServer{
		games: make(map[string]*Game),
	}
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
	type Resp struct {
		Action  string
		Session string
		Message string
		Board   string
	}
	resp := &Resp{
		Action:  r.URL.Query().Get("action"),
		Session: r.URL.Query().Get("session"),
	}
	switch resp.Action {
	case "new":
		resp.Session = uuid.NewString()
		brd, err := board.FEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBKQBNR")
		if err != nil {
			http.Error(w, fmt.Sprintf("%s: FEN: %s", resp.Action, err), http.StatusBadRequest)
		}
		game := &Game{
			Position: board.Position{Board: brd},
			Searcher: &board.Searcher{TP: map[board.Position]board.Entry{}},
		}
		s.mux.Lock()
		s.games[resp.Session] = game
		resp.Message = "New Game Created"
		resp.Board = game.Position.Board.String()
		s.mux.Unlock()
	default:
		http.Error(w, fmt.Sprintf("invalid action: %s", resp.Action), http.StatusBadRequest)
		return
	}

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
