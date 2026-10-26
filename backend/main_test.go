package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postJSON(t *testing.T, handler http.Handler, path, payload string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code >= 300 {
		t.Fatalf("POST %s returned %d: %s", path, res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func getJSON(t *testing.T, handler http.Handler, path string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code >= 300 {
		t.Fatalf("GET %s returned %d: %s", path, res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestRoomLifecycle(t *testing.T) {
	store.rooms = map[string]*Room{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rooms", roomsHandler)
	mux.HandleFunc("/api/rooms/", roomHandler)

	created := postJSON(t, mux, "/api/rooms", `{"name":"房主"}`)
	code := created["code"].(string)
	hostID := created["player_id"].(string)
	joined := postJSON(t, mux, "/api/rooms/"+code+"/join", `{"name":"队友"}`)
	guestID := joined["player_id"].(string)

	postJSON(t, mux, "/api/rooms/"+code+"/ready", `{"player_id":"`+hostID+`"}`)
	postJSON(t, mux, "/api/rooms/"+code+"/ready", `{"player_id":"`+guestID+`"}`)
	started := postJSON(t, mux, "/api/rooms/"+code+"/start", `{"player_id":"`+hostID+`"}`)
	if started["phase"] != "playing" {
		t.Fatalf("expected playing phase, got %v", started["phase"])
	}
	if started["prompt"] == "" {
		t.Fatal("expected first prompt")
	}
	guestView := getJSON(t, mux, "/api/rooms/"+code+"?player_id="+guestID)
	if guestView["prompt"] != "" {
		t.Fatal("guest must not see the original prompt")
	}

	postJSON(t, mux, "/api/rooms/"+code+"/submit", `{"player_id":"`+hostID+`","transcript":"雨后的电车穿过橘色隧道","audio_data":"data:audio/webm;base64,AA=="}`)
	guessView := getJSON(t, mux, "/api/rooms/"+code+"?player_id="+guestID)
	if guessView["stage"] != "guess" || guessView["relay_text"] != "雨后的电车穿过橘色隧道" {
		t.Fatalf("expected guest guess view, got stage=%v relay=%v", guessView["stage"], guessView["relay_text"])
	}
	if guessView["relay_audio"] != "data:audio/webm;base64,AA==" {
		t.Fatalf("expected guest to receive previous audio, got %v", guessView["relay_audio"])
	}
	afterGuess := postJSON(t, mux, "/api/rooms/"+code+"/guess", `{"player_id":"`+guestID+`","guess":"雨后的电车穿过橘色隧道"}`)
	if afterGuess["round"] != float64(2) || afterGuess["stage"] != "relay" {
		t.Fatalf("expected round 2 relay state, got round=%v stage=%v", afterGuess["round"], afterGuess["stage"])
	}
	chat := postJSON(t, mux, "/api/rooms/"+code+"/chat", `{"player_id":"`+guestID+`","text":"接得漂亮"}`)
	if len(chat["chat"].([]any)) != 2 {
		t.Fatalf("expected system and player messages, got %v", chat["chat"])
	}
}

func TestFinalRevealKeepsRelayChain(t *testing.T) {
	store.rooms = map[string]*Room{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rooms", roomsHandler)
	mux.HandleFunc("/api/rooms/", roomHandler)

	created := postJSON(t, mux, "/api/rooms", `{"name":"房主"}`)
	code := created["code"].(string)
	hostID := created["player_id"].(string)
	joined := postJSON(t, mux, "/api/rooms/"+code+"/join", `{"name":"猜测者"}`)
	guestID := joined["player_id"].(string)
	store.rooms[code].MaxRounds = 1
	postJSON(t, mux, "/api/rooms/"+code+"/ready", `{"player_id":"`+hostID+`"}`)
	postJSON(t, mux, "/api/rooms/"+code+"/ready", `{"player_id":"`+guestID+`"}`)
	postJSON(t, mux, "/api/rooms/"+code+"/start", `{"player_id":"`+hostID+`"}`)
	postJSON(t, mux, "/api/rooms/"+code+"/submit", `{"player_id":"`+hostID+`","transcript":"雨后的电车穿过橘色隧道"}`)
	final := postJSON(t, mux, "/api/rooms/"+code+"/guess", `{"player_id":"`+guestID+`","guess":"雨后的电车穿过橘色隧道"}`)
	if final["phase"] != "finished" || final["stage"] != "finished" {
		t.Fatalf("expected finished reveal, got phase=%v stage=%v", final["phase"], final["stage"])
	}
	if final["final_guess_player_id"] != guestID || len(final["turns"].([]any)) != 1 {
		t.Fatalf("expected final guess metadata and one relay turn, got player=%v turns=%v", final["final_guess_player_id"], final["turns"])
	}
}
