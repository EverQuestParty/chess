package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/notnil/chess"
	"github.com/notnil/chess/uci"
)

// Game wraps each individual game
type Game struct {
	instance *chess.Game
	lastMove string
	isAI     bool
}

type httpServer struct {
	mux    sync.RWMutex
	games  map[string]*Game
	engine *uci.Engine
}

func main() {
	err := run()
	if err != nil {
		fmt.Println("failed:", err)
		os.Exit(1)
	}

}

func run() error {
	eng, err := uci.New("stockfish")
	if err != nil {
		return fmt.Errorf("uci.New: %w", err)
	}
	err = eng.Run(uci.CmdUCI, uci.CmdIsReady, uci.CmdUCINewGame)
	if err != nil {
		return fmt.Errorf("uci.Run: %w", err)
	}

	server := &httpServer{
		engine: eng,
		games:  make(map[string]*Game),
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
		Action   string
		Move     string
		AIMove   string
		LastMove string
		Session  string
		Message  string
		Board    string
	}
	resp := &Resp{
		Action:  r.URL.Query().Get("action"),
		Move:    r.URL.Query().Get("move"),
		Session: r.URL.Query().Get("session"),
	}
	s.mux.Lock()
	defer s.mux.Unlock()

	switch resp.Action {
	case "new":
		resp.Session = uuid.NewString()
		game := &Game{
			instance: chess.NewGame(chess.UseNotation(chess.UCINotation{})),
			isAI:     resp.Move != "player",
		}
		s.games[resp.Session] = game
		resp.Message = "New Game Created"
		resp.Board = game.instance.String()
		fmt.Println(game.instance.Position().Board().Draw())
	case "move":
		game := s.games[resp.Session]
		if game == nil {
			http.Error(w, fmt.Sprintf("%s: game not found", resp.Action), http.StatusBadRequest)
			return
		}
		resp.LastMove = game.lastMove
		err := game.instance.MoveStr(resp.Move)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s %s: invalid move", resp.Action, resp.Move), http.StatusBadRequest)
			return
		}
		game.lastMove = resp.Move
		resp.LastMove = resp.Move
		if game.isAI {
			cmdPos := uci.CmdPosition{Position: game.instance.Position()}
			cmdGo := uci.CmdGo{MoveTime: time.Second / 100}
			err = s.engine.Run(cmdPos, cmdGo)
			if err != nil {
				http.Error(w, fmt.Sprintf("%s %s: run: %s", resp.Action, resp.Move, err), http.StatusBadRequest)
				return
			}
			move := s.engine.SearchResults().BestMove
			err = game.instance.Move(move)
			if err != nil {
				http.Error(w, fmt.Sprintf("%s %s: move: %s", resp.Action, resp.Move, err), http.StatusBadRequest)
				return
			}
			resp.AIMove = move.String()
		}

		resp.Board = game.instance.String()
		fmt.Println(game.instance.Position().Board().Draw())
	case "auto":
		game := s.games[resp.Session]
		if game == nil {
			http.Error(w, fmt.Sprintf("%s: game not found", resp.Action), http.StatusBadRequest)
			return
		}
		cmdPos := uci.CmdPosition{Position: game.instance.Position()}
		cmdGo := uci.CmdGo{MoveTime: time.Second / 100}
		err := s.engine.Run(cmdPos, cmdGo)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s %s: run: %s", resp.Action, resp.Move, err), http.StatusBadRequest)
			return
		}
		move := s.engine.SearchResults().BestMove
		err = game.instance.Move(move)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s %s: move: %s", resp.Action, resp.Move, err), http.StatusBadRequest)
			return
		}
		resp.Move = move.String()
		game.lastMove = resp.Move
		resp.LastMove = resp.Move

		cmdPos = uci.CmdPosition{Position: game.instance.Position()}
		cmdGo = uci.CmdGo{MoveTime: time.Second / 100}
		err = s.engine.Run(cmdPos, cmdGo)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s %s: run: %s", resp.Action, resp.Move, err), http.StatusBadRequest)
			return
		}
		move = s.engine.SearchResults().BestMove
		err = game.instance.Move(move)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s %s: move: %s", resp.Action, resp.Move, err), http.StatusBadRequest)
			return
		}
		resp.AIMove = move.String()
		resp.Board = game.instance.String()
		fmt.Println(game.instance.Position().Board().Draw())
	case "board":
		game := s.games[resp.Session]
		if game == nil {
			http.Error(w, fmt.Sprintf("%s: game not found", resp.Action), http.StatusBadRequest)
			return
		}
		resp.Message = "Last move: "
		resp.LastMove = game.lastMove
		resp.Board = game.instance.String()
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
