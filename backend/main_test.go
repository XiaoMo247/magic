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

	postJSON(t, mux, "/api/rooms/"+code+"/submit", `{"player_id":"`+hostID+`","transcript":"雨后的电车穿过橘色隧道"}`)
	chat := postJSON(t, mux, "/api/rooms/"+code+"/chat", `{"player_id":"`+guestID+`","text":"接得漂亮"}`)
	if len(chat["chat"].([]any)) != 2 {
		t.Fatalf("expected system and player messages, got %v", chat["chat"])
	}
}
