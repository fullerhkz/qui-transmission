package qbittorrent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type capturedRPC struct {
	Method string
	Params map[string]json.RawMessage
	Auth   string
}

func newTransmissionRPCServer(t *testing.T, handler func(capturedRPC) any) (*httptest.Server, *[]capturedRPC) {
	t.Helper()

	calls := make([]capturedRPC, 0)
	seenSession := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transmission/rpc" {
			t.Fatalf("path = %q, want /transmission/rpc", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}

		if !seenSession {
			seenSession = true
			w.Header().Set("X-Transmission-Session-Id", "test-session")
			w.WriteHeader(http.StatusConflict)
			return
		}
		if got := r.Header.Get("X-Transmission-Session-Id"); got != "test-session" {
			t.Fatalf("session id = %q, want test-session", got)
		}

		var req struct {
			JSONRPC string                     `json:"jsonrpc"`
			Method  string                     `json:"method"`
			Params  map[string]json.RawMessage `json:"params"`
			ID      int64                      `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.JSONRPC != "2.0" {
			t.Fatalf("jsonrpc = %q, want 2.0", req.JSONRPC)
		}

		auth := r.Header.Get("Authorization")
		call := capturedRPC{Method: req.Method, Params: req.Params, Auth: auth}
		calls = append(calls, call)
		result := handler(call)
		if result == nil {
			result = map[string]any{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  result,
		})
	}))

	return server, &calls
}

func requireJSONValue[T comparable](t *testing.T, params map[string]json.RawMessage, key string, want T) {
	t.Helper()

	var got T
	if err := json.Unmarshal(params[key], &got); err != nil {
		t.Fatalf("decode %s: %v", key, err)
	}
	if got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}

func requireJSONStringSlice(t *testing.T, params map[string]json.RawMessage, key string, want []string) {
	t.Helper()

	var got []string
	if err := json.Unmarshal(params[key], &got); err != nil {
		t.Fatalf("decode %s: %v", key, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}

func TestClient_LoginUsesTransmissionSessionAndBasicAuth(t *testing.T) {
	server, calls := newTransmissionRPCServer(t, func(call capturedRPC) any {
		if call.Method != "session_get" {
			t.Fatalf("method = %q, want session_get", call.Method)
		}
		if !strings.HasPrefix(call.Auth, "Basic ") {
			t.Fatalf("missing basic auth header")
		}
		return map[string]any{
			"version":            "4.1.0",
			"rpc_version":        18,
			"rpc_version_semver": "6.0.0",
			"download_dir":       "/downloads",
		}
	})
	defer server.Close()

	client := NewClient(Config{
		Host:     server.URL,
		Username: "user",
		Password: "pass",
	})

	if err := client.LoginCtx(context.Background()); err != nil {
		t.Fatalf("LoginCtx() error = %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("rpc calls = %d, want 1", len(*calls))
	}
	requireJSONStringSlice(t, (*calls)[0].Params, "fields", []string{"version", "rpc_version_semver", "rpc_version", "download_dir"})
}

func TestClient_GetTorrentsMapsTransmissionFields(t *testing.T) {
	server, _ := newTransmissionRPCServer(t, func(call capturedRPC) any {
		if call.Method != "torrent_get" {
			t.Fatalf("method = %q, want torrent_get", call.Method)
		}
		return map[string]any{
			"torrents": []map[string]any{
				{
					"hash_string":     "abc123",
					"name":            "example",
					"download_dir":    "/downloads",
					"group":           "movies",
					"labels":          []string{"hd", "archive"},
					"percent_done":    0.5,
					"status":          4,
					"rate_download":   1024,
					"rate_upload":     512,
					"total_size":      2048,
					"size_when_done":  2048,
					"left_until_done": 1024,
					"upload_ratio":    1.5,
					"uploaded_ever":   3072,
					"downloaded_ever": 1024,
					"tracker_stats": []map[string]any{
						{"announce": "https://tracker.example/announce", "seeder_count": 3, "leecher_count": 1},
					},
				},
			},
		}
	})
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	torrents, err := client.GetTorrentsCtx(context.Background(), TorrentFilterOptions{})
	if err != nil {
		t.Fatalf("GetTorrentsCtx() error = %v", err)
	}
	if len(torrents) != 1 {
		t.Fatalf("torrents = %d, want 1", len(torrents))
	}

	got := torrents[0]
	if got.Hash != "ABC123" || got.Name != "example" || got.Category != "movies" || got.Tags != "hd, archive" {
		t.Fatalf("unexpected torrent mapping: %#v", got)
	}
	if got.State != TorrentStateDownloading {
		t.Fatalf("state = %q, want %q", got.State, TorrentStateDownloading)
	}
	if got.Tracker != "https://tracker.example/announce" {
		t.Fatalf("tracker = %q", got.Tracker)
	}
}

func TestClient_SetGroupLabelsAndCommentUseTorrentSet(t *testing.T) {
	var methods []capturedRPC
	server, _ := newTransmissionRPCServer(t, func(call capturedRPC) any {
		methods = append(methods, call)
		return nil
	})
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	if err := client.SetCategoryCtx(context.Background(), []string{"abc123"}, "movies"); err != nil {
		t.Fatalf("SetCategoryCtx() error = %v", err)
	}
	if err := client.SetTags(context.Background(), []string{"abc123"}, "hd,archive"); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}
	if err := client.SetCommentCtx(context.Background(), []string{"abc123"}, "note"); err != nil {
		t.Fatalf("SetCommentCtx() error = %v", err)
	}

	if len(methods) != 3 {
		t.Fatalf("methods = %d, want 3", len(methods))
	}
	for _, call := range methods {
		if call.Method != "torrent_set" {
			t.Fatalf("method = %q, want torrent_set", call.Method)
		}
		requireJSONStringSlice(t, call.Params, "ids", []string{"abc123"})
	}
	requireJSONValue(t, methods[0].Params, "group", "movies")
	requireJSONStringSlice(t, methods[1].Params, "labels", []string{"hd", "archive"})
	requireJSONValue(t, methods[2].Params, "comment", "note")
}

func TestClient_AddTagsPreservesExistingTransmissionLabels(t *testing.T) {
	var setCall capturedRPC
	server, _ := newTransmissionRPCServer(t, func(call capturedRPC) any {
		switch call.Method {
		case "torrent_get":
			return map[string]any{
				"torrents": []map[string]any{
					{"hash_string": "ABC123", "labels": []string{"old"}},
				},
			}
		case "torrent_set":
			setCall = call
			return nil
		default:
			t.Fatalf("unexpected method %q", call.Method)
		}
		return nil
	})
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	if err := client.AddTagsCtx(context.Background(), []string{"abc123"}, "new"); err != nil {
		t.Fatalf("AddTagsCtx() error = %v", err)
	}
	requireJSONStringSlice(t, setCall.Params, "labels", []string{"old", "new"})
}

func TestClient_RSSMethodsReportUnsupportedTransmissionRPC(t *testing.T) {
	client := NewClient(Config{Host: "http://example.invalid"})
	if _, err := client.GetRSSItemsCtx(context.Background(), true); err == nil {
		t.Fatal("GetRSSItemsCtx() error = nil, want unsupported")
	}
}
