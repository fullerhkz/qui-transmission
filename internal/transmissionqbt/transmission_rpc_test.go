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
			Method    string                     `json:"method"`
			Arguments map[string]json.RawMessage `json:"arguments"`
			Tag       int64                      `json:"tag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		auth := r.Header.Get("Authorization")
		call := capturedRPC{Method: req.Method, Params: req.Arguments, Auth: auth}
		calls = append(calls, call)
		result := handler(call)
		if result == nil {
			result = map[string]any{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.Tag,
			"result":  result,
		})
	}))

	return server, &calls
}

func newClassicTransmissionRPCServer(t *testing.T, handler func(capturedRPC) any) (*httptest.Server, *[]capturedRPC) {
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
			Method    string                     `json:"method"`
			Arguments map[string]json.RawMessage `json:"arguments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		auth := r.Header.Get("Authorization")
		call := capturedRPC{Method: req.Method, Params: req.Arguments, Auth: auth}
		calls = append(calls, call)
		result := handler(call)
		if result == nil {
			result = map[string]any{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"arguments": result,
			"result":    "success",
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

func requireNormalizedKeys(t *testing.T, method string, params map[string]interface{}, want map[string]interface{}) {
	t.Helper()

	normalized, ok := normalizeTransmissionRequest(method, params).(map[string]interface{})
	if !ok {
		t.Fatalf("normalized request has type %T, want map[string]interface{}", normalized)
	}
	for key, value := range want {
		got, ok := normalized[key]
		if !ok {
			t.Fatalf("normalized request missing key %q: %#v", key, normalized)
		}
		if !reflect.DeepEqual(got, value) {
			t.Fatalf("normalized[%q] = %#v, want %#v", key, got, value)
		}
	}
}

func TestNormalizeClassicTransmissionRequestNames(t *testing.T) {
	requireNormalizedKeys(t, "session_set", map[string]interface{}{
		"speed_limit_down":   100,
		"seed_ratio_limit":   2.0,
		"idle_seeding_limit": 60,
	}, map[string]interface{}{
		"speed-limit-down":    100,
		"seedRatioLimit":      2.0,
		"idle-seeding-limit":  60,
	})

	requireNormalizedKeys(t, "torrent_add", map[string]interface{}{
		"download_dir":        "/downloads",
		"peer_limit":          80,
		"bandwidth_priority":  1,
		"files_wanted":        []interface{}{0.0, 1.0},
		"sequential_download": true,
	}, map[string]interface{}{
		"download-dir":        "/downloads",
		"peer-limit":          80,
		"bandwidthPriority":   1,
		"files-wanted":        []interface{}{0.0, 1.0},
		"sequentialDownload":  true,
	})

	requireNormalizedKeys(t, "torrent_set", map[string]interface{}{
		"peer_limit":          80,
		"download_limit":      1024,
		"tracker_list":        "https://tracker.example/announce",
		"sequential_download": true,
	}, map[string]interface{}{
		"peer-limit":          80,
		"downloadLimit":       1024,
		"trackerList":         "https://tracker.example/announce",
		"sequentialDownload":  true,
	})

	requireNormalizedKeys(t, "torrent_get", map[string]interface{}{
		"fields": []string{"hash_string", "download_dir", "file_count", "peer_limit", "primary_mime_type", "tracker_list"},
	}, map[string]interface{}{
		"fields": []string{"hashString", "downloadDir", "file-count", "peer-limit", "primary-mime-type", "trackerList"},
	})

	requireNormalizedKeys(t, "session_get", map[string]interface{}{
		"fields": []string{"version", "rpc_version_semver", "download_dir", "seed_ratio_limit"},
	}, map[string]interface{}{
		"fields": []string{"version", "rpc-version-semver", "download-dir", "seedRatioLimit"},
	})
}

func TestClient_LoginUsesTransmissionSessionAndBasicAuth(t *testing.T) {
	server, calls := newTransmissionRPCServer(t, func(call capturedRPC) any {
		if call.Method != "session-get" {
			t.Fatalf("method = %q, want session-get", call.Method)
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
	requireJSONStringSlice(t, (*calls)[0].Params, "fields", []string{"version", "rpc-version-semver", "rpc-version", "download-dir"})
}

func TestClient_LoginAcceptsClassicTransmissionResponse(t *testing.T) {
	server, calls := newClassicTransmissionRPCServer(t, func(call capturedRPC) any {
		if call.Method != "session-get" {
			t.Fatalf("method = %q, want session-get", call.Method)
		}
		if !strings.HasPrefix(call.Auth, "Basic ") {
			t.Fatalf("missing basic auth header")
		}
		return map[string]any{
			"version":              "4.0.6",
			"rpc-version":          17,
			"rpc-version-semver":   "5.3.0",
			"download-dir":         "/downloads",
			"alt-speed-enabled":    false,
			"startAddedTorrents":   true,
			"seedRatioLimited":     false,
			"seedRatioLimit":       2.0,
			"blocklist-enabled":    false,
			"blocklist-url":        "",
			"rename-partial-files": true,
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
}

func TestClient_GetTorrentsMapsTransmissionFields(t *testing.T) {
	server, _ := newTransmissionRPCServer(t, func(call capturedRPC) any {
		if call.Method != "torrent-get" {
			t.Fatalf("method = %q, want torrent-get", call.Method)
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

func TestClient_GetTorrentsAcceptsClassicCamelCaseFields(t *testing.T) {
	server, _ := newClassicTransmissionRPCServer(t, func(call capturedRPC) any {
		if call.Method != "torrent-get" {
			t.Fatalf("method = %q, want torrent-get", call.Method)
		}
		return map[string]any{
			"torrents": []map[string]any{
				{
					"hashString":     "abc123",
					"name":           "example",
					"downloadDir":    "/downloads",
					"group":          "movies",
					"labels":         []string{"hd", "archive"},
					"percentDone":    0.5,
					"status":         4,
					"rateDownload":   1024,
					"rateUpload":     512,
					"totalSize":      2048,
					"sizeWhenDone":   2048,
					"leftUntilDone":  1024,
					"uploadRatio":    1.5,
					"uploadedEver":   3072,
					"downloadedEver": 1024,
					"trackerStats": []map[string]any{
						{"announce": "https://tracker.example/announce", "seederCount": 3, "leecherCount": 1},
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
		if call.Method != "torrent-set" {
			t.Fatalf("method = %q, want torrent-set", call.Method)
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
		case "torrent-get":
			return map[string]any{
				"torrents": []map[string]any{
					{"hash_string": "ABC123", "labels": []string{"old"}},
				},
			}
		case "torrent-set":
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
