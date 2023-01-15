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

// Game wraps each individual game
type Game struct {
	IsProcessing bool
	LastMove     string
	Position     board.Position
	Searcher     *board.Searcher
	IsAI         bool
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
	addr := ":6969"
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
		Move    string
		Session string
		Message string
		Board   string
	}
	resp := &Resp{
		Action:  r.URL.Query().Get("action"),
		Move:    r.URL.Query().Get("move"),
		Session: r.URL.Query().Get("session"),
	}
	switch resp.Action {
	case "new":
		resp.Session = uuid.NewString()
		brd, err := board.FEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBKQBNR")
		if err != nil {
			http.Error(w, fmt.Sprintf("%s: FEN: %s", resp.Action, err), http.StatusBadRequest)
			return
		}
		game := &Game{
			Position: board.Position{Board: brd},
			Searcher: &board.Searcher{TP: map[board.Position]board.Entry{}},
			IsAI:     resp.Move != "player",
		}
		s.mux.Lock()
		s.games[resp.Session] = game
		resp.Message = "New Game Created"
		resp.Board = game.Position.Board.String()
		s.mux.Unlock()
	case "move":
		s.mux.Lock()
		game := s.games[resp.Session]
		if game == nil {
			http.Error(w, fmt.Sprintf("%s: game not found", resp.Action), http.StatusBadRequest)
			s.mux.Unlock()
			return
		}
		if game.IsProcessing {
			http.Error(w, fmt.Sprintf("%s: game is processing", resp.Action), http.StatusBadRequest)
			s.mux.Unlock()
			return
		}
		isValid := false
		for _, m := range game.Position.Moves() {
			if resp.Move != m.String() {
				continue
			}
			game.Position = game.Position.Move(m)
			isValid = true
			break
		}
		if !isValid {
			http.Error(w, fmt.Sprintf("%s %s: invalid move", resp.Action, resp.Move), http.StatusBadRequest)
			s.mux.Unlock()
			return
		}
		resp.Board = game.Position.Flip().Board.String()
		game.LastMove = resp.Move
		s.mux.Unlock()
		go func() {
			game.IsProcessing = true
			if game.IsAI {
				m := game.Searcher.Search(game.Position, 10000)
				score := game.Position.Value(m)
				if score <= -board.MateValue {
					resp.Message = "You won!"
				}
				if score >= board.MateValue {
					resp.Message = "You lost!"
				}

				game.Position = game.Position.Move(m)
			}
			game.IsProcessing = false
		}()
	case "board":
		s.mux.Lock()
		defer s.mux.Unlock()
		game := s.games[resp.Session]
		if game == nil {
			http.Error(w, fmt.Sprintf("%s: game not found", resp.Action), http.StatusBadRequest)
			return
		}
		resp.Message = "Last move: "
		resp.Board = game.Position.Board.String()
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
