package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Player struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Host  bool   `json:"host"`
	Ready bool   `json:"ready"`
	Score int    `json:"score"`
}
type ChatMessage struct {
	Name string `json:"name"`
	Text string `json:"text"`
}
type Result struct {
	Transcript string `json:"transcript"`
	Score      int    `json:"score"`
}
type RelayTurn struct {
	PlayerID   string `json:"player_id"`
	Transcript string `json:"transcript"`
	Score      int    `json:"score"`
	AudioData  string `json:"audio_data,omitempty"`
}
type Room struct {
	Code                           string    `json:"code"`
	Players                        []*Player `json:"players"`
	Phase                          string    `json:"phase"`
	Stage                          string    `json:"stage"`
	Round, MaxRounds, CurrentIndex int
	TurnStartIndex                 int
	RelayStep                      int
	Prompt                         string        `json:"prompt"`
	RelayText                      string        `json:"relay_text"`
	RelayAudio                     string        `json:"relay_audio,omitempty"`
	FinalGuess                     string        `json:"final_guess,omitempty"`
	FinalGuessPlayerID             string        `json:"final_guess_player_id,omitempty"`
	Turns                          []RelayTurn   `json:"turns"`
	LastResult                     *Result       `json:"last_result,omitempty"`
	Chat                           []ChatMessage `json:"chat"`
	StartedAt                      time.Time
}
type Store struct {
	sync.RWMutex
	rooms map[string]*Room
}

var store = Store{rooms: map[string]*Room{}}
var prompts = []string{"雨后的电车穿过橘色隧道", "月亮把影子藏进玻璃杯", "三只蓝色纸船顺流而下", "晚风带来一颗会唱歌的种子", "森林尽头亮起一盏紫色信号灯"}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rooms", roomsHandler)
	mux.HandleFunc("/api/rooms/", roomHandler)
	fmt.Println("Magic Relay API listening on :8080")
	http.ListenAndServe(":8080", cors(mux))
}
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func writeJSON(w http.ResponseWriter, v any, s int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	json.NewEncoder(w).Encode(v)
}
func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		return errors.New("请求数据格式无效")
	}
	return nil
}
func newID(p string) string {
	n, _ := rand.Int(rand.Reader, big.NewInt(899999))
	return fmt.Sprintf("%s%d", p, n.Int64()+100000)
}
func newCode() string {
	const c = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(c))))
		b[i] = c[n.Int64()]
	}
	return string(b)
}
func roomsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 400)
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		in.Name = "玩家"
	}
	code := newCode()
	store.Lock()
	for store.rooms[code] != nil {
		code = newCode()
	}
	room := &Room{Code: code, Phase: "lobby", MaxRounds: 5, Chat: []ChatMessage{{Name: "系统", Text: "房间创建成功，等待队友加入"}}}
	p := &Player{ID: newID("p_"), Name: in.Name, Host: true}
	room.Players = []*Player{p}
	store.rooms[room.Code] = room
	store.Unlock()
	writeJSON(w, map[string]string{"code": room.Code, "player_id": p.ID}, 200)
}
func roomHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeJSON(w, map[string]string{"error": "room code required"}, 400)
		return
	}
	code := strings.ToUpper(parts[2])
	store.Lock()
	room, ok := store.rooms[code]
	if !ok {
		store.Unlock()
		writeJSON(w, map[string]string{"error": "房间不存在"}, 404)
		return
	}
	defer store.Unlock()
	action := ""
	if len(parts) > 3 {
		action = parts[3]
	}
	switch action {
	case "join":
		joinRoom(w, r, room)
	case "ready":
		setReady(w, r, room)
	case "start":
		startRoom(w, r, room)
	case "submit":
		submit(w, r, room)
	case "guess":
		guess(w, r, room)
	case "chat":
		chatRoom(w, r, room)
	default:
		if r.Method != "GET" {
			writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
			return
		}
		writeJSON(w, viewForPlayer(room, r.URL.Query().Get("player_id")), 200)
	}
}
func findPlayer(r *Room, id string) *Player {
	for _, p := range r.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}
func joinRoom(w http.ResponseWriter, r *http.Request, room *Room) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 400)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeJSON(w, map[string]string{"error": "昵称不能为空"}, 400)
		return
	}
	if len(room.Players) >= 8 {
		writeJSON(w, map[string]string{"error": "房间已满"}, 409)
		return
	}
	p := &Player{ID: newID("p_"), Name: in.Name}
	room.Players = append(room.Players, p)
	writeJSON(w, map[string]string{"player_id": p.ID}, 200)
}
func setReady(w http.ResponseWriter, r *http.Request, room *Room) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
		return
	}
	var in struct {
		PlayerID string `json:"player_id"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 400)
		return
	}
	if p := findPlayer(room, in.PlayerID); p != nil {
		p.Ready = true
		writeJSON(w, viewForPlayer(room, in.PlayerID), 200)
		return
	}
	writeJSON(w, map[string]string{"error": "玩家不存在"}, 404)
}
func startRoom(w http.ResponseWriter, r *http.Request, room *Room) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
		return
	}
	var in struct {
		PlayerID string `json:"player_id"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 400)
		return
	}
	p := findPlayer(room, in.PlayerID)
	if p == nil || !p.Host {
		writeJSON(w, map[string]string{"error": "只有房主可以开始"}, 403)
		return
	}
	if len(room.Players) < 2 {
		writeJSON(w, map[string]string{"error": "至少需要 2 名玩家"}, 400)
		return
	}
	for _, player := range room.Players {
		if !player.Ready {
			writeJSON(w, map[string]string{"error": "还有玩家未准备"}, 400)
			return
		}
	}
	room.Phase = "playing"
	room.Stage = "relay"
	room.CurrentIndex = 0
	room.TurnStartIndex = 0
	room.RelayStep = 0
	room.Round = 1
	room.Prompt = prompts[0]
	room.RelayText = room.Prompt
	room.Turns = nil
	room.FinalGuess = ""
	room.StartedAt = time.Now()
	writeJSON(w, viewForPlayer(room, in.PlayerID), 200)
}
func submit(w http.ResponseWriter, r *http.Request, room *Room) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
		return
	}
	var in struct {
		PlayerID   string `json:"player_id"`
		Transcript string `json:"transcript"`
		AudioData  string `json:"audio_data"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 400)
		return
	}
	if len(in.AudioData) > 3_000_000 {
		writeJSON(w, map[string]string{"error": "录音文件过大，请控制在 2MB 以内"}, 400)
		return
	}
	if room.Phase != "playing" || room.Stage != "relay" || room.Players[room.CurrentIndex].ID != in.PlayerID {
		writeJSON(w, map[string]string{"error": "还没轮到你"}, 400)
		return
	}
	score := scoreFor(in.Transcript, room.RelayText)
	p := room.Players[room.CurrentIndex]
	p.Score += score
	room.Turns = append(room.Turns, RelayTurn{PlayerID: p.ID, Transcript: in.Transcript, Score: score, AudioData: in.AudioData})
	room.RelayText = strings.TrimSpace(in.Transcript)
	room.RelayAudio = in.AudioData
	room.LastResult = &Result{Transcript: in.Transcript, Score: score}
	if room.RelayStep >= len(room.Players)-2 {
		room.Stage = "guess"
		room.CurrentIndex = (room.TurnStartIndex + len(room.Players) - 1) % len(room.Players)
	} else {
		room.RelayStep++
		room.CurrentIndex = (room.TurnStartIndex + room.RelayStep) % len(room.Players)
	}
	writeJSON(w, viewForPlayer(room, in.PlayerID), 200)
}
func guess(w http.ResponseWriter, r *http.Request, room *Room) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
		return
	}
	var in struct {
		PlayerID string `json:"player_id"`
		Guess    string `json:"guess"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 400)
		return
	}
	if room.Phase != "playing" || room.Stage != "guess" || room.Players[room.CurrentIndex].ID != in.PlayerID {
		writeJSON(w, map[string]string{"error": "还没轮到你猜测"}, 400)
		return
	}
	in.Guess = strings.TrimSpace(in.Guess)
	if in.Guess == "" {
		writeJSON(w, map[string]string{"error": "猜测内容不能为空"}, 400)
		return
	}
	score := scoreFor(in.Guess, room.Prompt)
	room.Players[room.CurrentIndex].Score += score
	room.FinalGuess = in.Guess
	room.FinalGuessPlayerID = in.PlayerID
	room.LastResult = &Result{Transcript: in.Guess, Score: score}
	if room.Round >= room.MaxRounds {
		room.Phase = "finished"
		room.Stage = "finished"
		writeJSON(w, viewForPlayer(room, in.PlayerID), 200)
		return
	}
	room.Round++
	room.Stage = "relay"
	room.TurnStartIndex = (room.Round - 1) % len(room.Players)
	room.RelayStep = 0
	room.CurrentIndex = room.TurnStartIndex
	room.Prompt = prompts[(room.Round-1)%len(prompts)]
	room.RelayText = room.Prompt
	room.RelayAudio = ""
	room.FinalGuess = ""
	room.FinalGuessPlayerID = ""
	room.Turns = nil
	writeJSON(w, viewForPlayer(room, in.PlayerID), 200)
}
func scoreFor(text, target string) int {
	if strings.TrimSpace(text) == "" {
		return 78
	}
	seen := map[rune]bool{}
	matched := 0
	for _, c := range text {
		if !seen[c] && strings.ContainsRune(target, c) {
			seen[c] = true
			matched++
		}
	}
	den := len([]rune(target))
	score := matched * 100 / den
	if score < 45 {
		score = 45
	}
	if score > 99 {
		score = 99
	}
	return score
}
func chatRoom(w http.ResponseWriter, r *http.Request, room *Room) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]string{"error": "method not allowed"}, 405)
		return
	}
	var in struct {
		PlayerID string `json:"player_id"`
		Text     string `json:"text"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 400)
		return
	}
	p := findPlayer(room, in.PlayerID)
	if p == nil || strings.TrimSpace(in.Text) == "" {
		writeJSON(w, map[string]string{"error": "消息无效"}, 400)
		return
	}
	room.Chat = append(room.Chat, ChatMessage{Name: p.Name, Text: strings.TrimSpace(in.Text)})
	if len(room.Chat) > 30 {
		room.Chat = room.Chat[len(room.Chat)-30:]
	}
	writeJSON(w, viewForPlayer(room, in.PlayerID), 200)
}

type RoomView struct {
	Code                 string        `json:"code"`
	Players              []*Player     `json:"players"`
	Phase                string        `json:"phase"`
	Stage                string        `json:"stage"`
	Round                int           `json:"round"`
	MaxRounds            int           `json:"max_rounds"`
	CurrentPlayerID      string        `json:"current_player_id"`
	CurrentPlayerName    string        `json:"current_player_name"`
	Prompt               string        `json:"prompt"`
	RelayText            string        `json:"relay_text"`
	RelayAudio           string        `json:"relay_audio,omitempty"`
	FinalGuess           string        `json:"final_guess,omitempty"`
	FinalGuessPlayerID   string        `json:"final_guess_player_id,omitempty"`
	FinalGuessPlayerName string        `json:"final_guess_player_name,omitempty"`
	Turns                []RelayTurn   `json:"turns"`
	LastResult           *Result       `json:"last_result,omitempty"`
	Chat                 []ChatMessage `json:"chat"`
	TeamProgress         int           `json:"team_progress"`
	RemainingSeconds     int           `json:"remaining_seconds"`
}

func view(r *Room) RoomView {
	cid, cn := "", ""
	if len(r.Players) > 0 && r.Phase == "playing" {
		cid = r.Players[r.CurrentIndex].ID
		cn = r.Players[r.CurrentIndex].Name
	}
	total := 0
	for _, p := range r.Players {
		total += p.Score
	}
	progress := 0
	if len(r.Players) > 0 {
		progress = total * 100 / (len(r.Players) * r.MaxRounds * 100)
	}
	remaining := 24
	if !r.StartedAt.IsZero() {
		remaining = 24 - int(time.Since(r.StartedAt).Seconds())%24
	}
	finalGuessName := ""
	if finalPlayer := findPlayer(r, r.FinalGuessPlayerID); finalPlayer != nil {
		finalGuessName = finalPlayer.Name
	}
	return RoomView{Code: r.Code, Players: r.Players, Phase: r.Phase, Stage: r.Stage, Round: r.Round, MaxRounds: r.MaxRounds, CurrentPlayerID: cid, CurrentPlayerName: cn, Prompt: r.Prompt, RelayText: r.RelayText, RelayAudio: r.RelayAudio, FinalGuess: r.FinalGuess, FinalGuessPlayerID: r.FinalGuessPlayerID, FinalGuessPlayerName: finalGuessName, Turns: r.Turns, LastResult: r.LastResult, Chat: r.Chat, TeamProgress: progress, RemainingSeconds: remaining}
}

func viewForPlayer(r *Room, playerID string) RoomView {
	v := view(r)
	if r.Phase == "lobby" || r.Phase == "finished" {
		return v
	}
	currentID := r.Players[r.CurrentIndex].ID
	if playerID != currentID {
		v.Prompt = ""
		v.RelayText = ""
		v.RelayAudio = ""
	}
	if r.Stage == "relay" && r.RelayStep > 0 && playerID == currentID {
		v.Prompt = ""
	}
	if r.Stage == "guess" && playerID == currentID {
		v.Prompt = ""
	}
	if r.Stage != "finished" {
		v.Turns = nil
		v.FinalGuess = ""
	}
	return v
}
