package qbittorrent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Masterminds/semver"
	"github.com/autobrr/go-qbittorrent/errors"
)

var rpcRequestID atomic.Int64

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int64       `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
	ID      int64           `json:"id,omitempty"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type trSession struct {
	Version                    string  `json:"version"`
	RPCVersion                 int     `json:"rpc_version"`
	RPCVersionSemver           string  `json:"rpc_version_semver"`
	DownloadDir                string  `json:"download_dir"`
	DownloadQueueEnabled       bool    `json:"download_queue_enabled"`
	DownloadQueueSize          int     `json:"download_queue_size"`
	SeedQueueEnabled           bool    `json:"seed_queue_enabled"`
	SeedQueueSize              int     `json:"seed_queue_size"`
	QueueStalledEnabled        bool    `json:"queue_stalled_enabled"`
	QueueStalledMinutes        int     `json:"queue_stalled_minutes"`
	SpeedLimitDown             int64   `json:"speed_limit_down"`
	SpeedLimitDownEnabled      bool    `json:"speed_limit_down_enabled"`
	SpeedLimitUp               int64   `json:"speed_limit_up"`
	SpeedLimitUpEnabled        bool    `json:"speed_limit_up_enabled"`
	AltSpeedDown               int64   `json:"alt_speed_down"`
	AltSpeedUp                 int64   `json:"alt_speed_up"`
	AltSpeedEnabled            bool    `json:"alt_speed_enabled"`
	PeerPort                   int     `json:"peer_port"`
	PeerLimitGlobal            int     `json:"peer_limit_global"`
	PeerLimitPerTorrent        int     `json:"peer_limit_per_torrent"`
	PEXEnabled                 bool    `json:"pex_enabled"`
	DHTEnabled                 bool    `json:"dht_enabled"`
	LPDEnabled                 bool    `json:"lpd_enabled"`
	UTPEnabled                 bool    `json:"utp_enabled"`
	PortForwardingEnabled      bool    `json:"port_forwarding_enabled"`
	RenamePartialFiles         bool    `json:"rename_partial_files"`
	StartAddedTorrents         bool    `json:"start_added_torrents"`
	TrashOriginalTorrentFiles  bool    `json:"trash_original_torrent_files"`
	SeedRatioLimit             float64 `json:"seed_ratio_limit"`
	SeedRatioLimited           bool    `json:"seed_ratio_limited"`
	IdleSeedingLimit           int     `json:"idle_seeding_limit"`
	IdleSeedingLimitEnabled    bool    `json:"idle_seeding_limit_enabled"`
	IncompleteDir              string  `json:"incomplete_dir"`
	IncompleteDirEnabled       bool    `json:"incomplete_dir_enabled"`
	ScriptTorrentAddedEnabled  bool    `json:"script_torrent_added_enabled"`
	ScriptTorrentAddedFilename string  `json:"script_torrent_added_filename"`
	BlocklistEnabled           bool    `json:"blocklist_enabled"`
	BlocklistURL               string  `json:"blocklist_url"`
}

type trStats struct {
	ActiveTorrentCount int          `json:"active_torrent_count"`
	PausedTorrentCount int          `json:"paused_torrent_count"`
	TorrentCount       int          `json:"torrent_count"`
	DownloadSpeed      int64        `json:"download_speed"`
	UploadSpeed        int64        `json:"upload_speed"`
	CumulativeStats    trStatsGroup `json:"cumulative_stats"`
	CurrentStats       trStatsGroup `json:"current_stats"`
}

type trStatsGroup struct {
	UploadedBytes   int64 `json:"uploaded_bytes"`
	DownloadedBytes int64 `json:"downloaded_bytes"`
	FilesAdded      int64 `json:"files_added"`
	SecondsActive   int64 `json:"seconds_active"`
	SessionCount    int64 `json:"session_count"`
}

type trTorrent struct {
	ActivityDate            int64           `json:"activity_date"`
	AddedDate               int64           `json:"added_date"`
	Availability            []int           `json:"availability"`
	BandwidthPriority       int             `json:"bandwidth_priority"`
	BytesCompleted          []int64         `json:"bytes_completed"`
	Comment                 string          `json:"comment"`
	CorruptEver             int64           `json:"corrupt_ever"`
	Creator                 string          `json:"creator"`
	DateCreated             int64           `json:"date_created"`
	DesiredAvailable        int64           `json:"desired_available"`
	DoneDate                int64           `json:"done_date"`
	DownloadDir             string          `json:"download_dir"`
	DownloadedEver          int64           `json:"downloaded_ever"`
	DownloadLimit           int64           `json:"download_limit"`
	DownloadLimited         bool            `json:"download_limited"`
	Error                   int             `json:"error"`
	ErrorString             string          `json:"error_string"`
	Eta                     int64           `json:"eta"`
	FileCount               int             `json:"file_count"`
	Files                   []trFile        `json:"files"`
	FileStats               []trFileStat    `json:"file_stats"`
	Group                   string          `json:"group"`
	HashString              string          `json:"hash_string"`
	HaveUnchecked           int64           `json:"have_unchecked"`
	HaveValid               int64           `json:"have_valid"`
	HonorsSessionLimits     bool            `json:"honors_session_limits"`
	ID                      int64           `json:"id"`
	IsFinished              bool            `json:"is_finished"`
	IsPrivate               bool            `json:"is_private"`
	IsStalled               bool            `json:"is_stalled"`
	Labels                  []string        `json:"labels"`
	LeftUntilDone           int64           `json:"left_until_done"`
	MagnetLink              string          `json:"magnet_link"`
	MaxConnectedPeers       int             `json:"max_connected_peers"`
	MetadataPercentComplete float64         `json:"metadata_percent_complete"`
	Name                    string          `json:"name"`
	PeerLimit               int             `json:"peer_limit"`
	Peers                   []trPeer        `json:"peers"`
	PeersConnected          int64           `json:"peers_connected"`
	PeersGettingFromUs      int64           `json:"peers_getting_from_us"`
	PeersSendingToUs        int64           `json:"peers_sending_to_us"`
	PercentComplete         float64         `json:"percent_complete"`
	PercentDone             float64         `json:"percent_done"`
	Pieces                  string          `json:"pieces"`
	PieceCount              int             `json:"piece_count"`
	PieceSize               int64           `json:"piece_size"`
	Priorities              []int           `json:"priorities"`
	QueuePosition           int64           `json:"queue_position"`
	RateDownload            int64           `json:"rate_download"`
	RateUpload              int64           `json:"rate_upload"`
	RecheckProgress         float64         `json:"recheck_progress"`
	SecondsDownloading      int64           `json:"seconds_downloading"`
	SecondsSeeding          int64           `json:"seconds_seeding"`
	SeedIdleLimit           int64           `json:"seed_idle_limit"`
	SeedIdleMode            int             `json:"seed_idle_mode"`
	SeedRatioLimit          float64         `json:"seed_ratio_limit"`
	SeedRatioMode           int             `json:"seed_ratio_mode"`
	SequentialDownload      bool            `json:"sequential_download"`
	SizeWhenDone            int64           `json:"size_when_done"`
	StartDate               int64           `json:"start_date"`
	Status                  int             `json:"status"`
	TorrentFile             string          `json:"torrent_file"`
	TotalSize               int64           `json:"total_size"`
	Trackers                []trTracker     `json:"trackers"`
	TrackerList             string          `json:"tracker_list"`
	TrackerStats            []trTrackerStat `json:"tracker_stats"`
	UploadedEver            int64           `json:"uploaded_ever"`
	UploadLimit             int64           `json:"upload_limit"`
	UploadLimited           bool            `json:"upload_limited"`
	UploadRatio             float64         `json:"upload_ratio"`
	Wanted                  []bool          `json:"wanted"`
	WebSeeds                []string        `json:"webseeds"`
	WebSeedsEx              []trWebSeed     `json:"webseeds_ex"`
	WebSeedsSendingToUs     int64           `json:"webseeds_sending_to_us"`
}

type trFile struct {
	BytesCompleted int64  `json:"bytes_completed"`
	Length         int64  `json:"length"`
	Name           string `json:"name"`
	BeginPiece     int    `json:"begin_piece"`
	EndPiece       int    `json:"end_piece"`
}

type trFileStat struct {
	BytesCompleted int64 `json:"bytes_completed"`
	Wanted         bool  `json:"wanted"`
	Priority       int   `json:"priority"`
}

type trTracker struct {
	Announce string `json:"announce"`
	ID       int    `json:"id"`
	Scrape   string `json:"scrape"`
	SiteName string `json:"sitename"`
	Tier     int    `json:"tier"`
}

type trTrackerStat struct {
	Announce              string `json:"announce"`
	AnnounceState         int    `json:"announce_state"`
	DownloadCount         int    `json:"download_count"`
	DownloaderCount       int    `json:"downloader_count"`
	HasAnnounced          bool   `json:"has_announced"`
	HasScraped            bool   `json:"has_scraped"`
	Host                  string `json:"host"`
	ID                    int    `json:"id"`
	IsBackup              bool   `json:"is_backup"`
	LastAnnouncePeerCount int    `json:"last_announce_peer_count"`
	LastAnnounceResult    string `json:"last_announce_result"`
	LastAnnounceSucceeded bool   `json:"last_announce_succeeded"`
	LastAnnounceTimedOut  bool   `json:"last_announce_timed_out"`
	LastScrapeResult      string `json:"last_scrape_result"`
	LastScrapeSucceeded   bool   `json:"last_scrape_succeeded"`
	LastScrapeTimedOut    bool   `json:"last_scrape_timed_out"`
	LeecherCount          int    `json:"leecher_count"`
	NextAnnounceTime      int64  `json:"next_announce_time"`
	SeederCount           int    `json:"seeder_count"`
	Scrape                string `json:"scrape"`
	ScrapeState           int    `json:"scrape_state"`
	SiteName              string `json:"sitename"`
	Tier                  int    `json:"tier"`
}

type trPeer struct {
	Address           string  `json:"address"`
	ClientName        string  `json:"client_name"`
	FlagStr           string  `json:"flag_str"`
	IsDownloadingFrom bool    `json:"is_downloading_from"`
	IsEncrypted       bool    `json:"is_encrypted"`
	IsIncoming        bool    `json:"is_incoming"`
	IsUploadingTo     bool    `json:"is_uploading_to"`
	IsUTP             bool    `json:"is_utp"`
	PeerID            string  `json:"peer_id"`
	Port              int     `json:"port"`
	Progress          float64 `json:"progress"`
	RateToClient      int64   `json:"rate_to_client"`
	RateToPeer        int64   `json:"rate_to_peer"`
	BytesToClient     int64   `json:"bytes_to_client"`
	BytesToPeer       int64   `json:"bytes_to_peer"`
}

type trWebSeed struct {
	URL string `json:"url"`
}

var torrentFields = []string{
	"activity_date", "added_date", "availability", "bandwidth_priority", "bytes_completed",
	"comment", "corrupt_ever", "creator", "date_created", "desired_available", "done_date",
	"download_dir", "downloaded_ever", "download_limit", "download_limited", "error",
	"error_string", "eta", "file_count", "files", "file_stats", "group", "hash_string",
	"have_unchecked", "have_valid", "honors_session_limits", "id", "is_finished",
	"is_private", "is_stalled", "labels", "left_until_done", "magnet_link",
	"max_connected_peers", "metadata_percent_complete", "name", "peer_limit",
	"peers_connected", "peers_getting_from_us", "peers_sending_to_us", "percent_complete",
	"percent_done", "piece_count", "piece_size", "priorities", "queue_position",
	"rate_download", "rate_upload", "recheck_progress", "seconds_downloading",
	"seconds_seeding", "seed_idle_limit", "seed_idle_mode", "seed_ratio_limit",
	"seed_ratio_mode", "sequential_download", "size_when_done", "start_date", "status",
	"torrent_file", "total_size", "trackers", "tracker_list", "tracker_stats",
	"uploaded_ever", "upload_limit", "upload_limited", "upload_ratio", "wanted",
	"webseeds", "webseeds_ex", "webseeds_sending_to_us",
}

func unsupported(feature string) error {
	return errors.Wrap(ErrUnsupportedVersion, "%s is not supported by Transmission RPC", feature)
}

type MonitoredFolderMode int

const (
	MonitoredFolderModeMonitoredFolder MonitoredFolderMode = 0
	MonitoredFolderModeDefaultSavePath MonitoredFolderMode = 1
	MonitoredFolderModeCustomPath      MonitoredFolderMode = 2
)

type MonitoredFolderTarget struct {
	mode       MonitoredFolderMode
	customPath string
}

func NewMonitoredFolderTarget(mode MonitoredFolderMode) MonitoredFolderTarget {
	return MonitoredFolderTarget{mode: mode}
}

func NewMonitoredFolderCustomPath(path string) MonitoredFolderTarget {
	return MonitoredFolderTarget{mode: MonitoredFolderModeCustomPath, customPath: path}
}

func (d MonitoredFolderTarget) Mode() MonitoredFolderMode {
	return d.mode
}

func (d MonitoredFolderTarget) CustomPath() string {
	return d.customPath
}

func (d MonitoredFolderTarget) MarshalJSON() ([]byte, error) {
	switch d.mode {
	case MonitoredFolderModeMonitoredFolder, MonitoredFolderModeDefaultSavePath:
		return json.Marshal(int(d.mode))
	case MonitoredFolderModeCustomPath:
		return json.Marshal(d.customPath)
	default:
		return nil, errors.Wrap(ErrInvalidMonitoredFolderTarget, "invalid target mode: %d", d.mode)
	}
}

func (d *MonitoredFolderTarget) UnmarshalJSON(data []byte) error {
	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		switch MonitoredFolderMode(intValue) {
		case MonitoredFolderModeMonitoredFolder, MonitoredFolderModeDefaultSavePath:
			d.mode = MonitoredFolderMode(intValue)
			d.customPath = ""
			return nil
		default:
			return errors.Wrap(ErrInvalidMonitoredFolderTarget, "invalid target integer value: %d", intValue)
		}
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		d.mode = MonitoredFolderModeCustomPath
		d.customPath = stringValue
		return nil
	}

	return errors.Wrap(ErrInvalidMonitoredFolderTarget, "invalid target value: %s", string(data))
}

type MonitoredFolders map[string]MonitoredFolderTarget

type ShareLimitOptions struct {
	RatioLimit               float64
	SeedingTimeLimit         int64
	InactiveSeedingTimeLimit int64
	ShareLimitAction         string
	ShareLimitsMode          string
}

const (
	ShareLimitActionDefault            = "Default"
	ShareLimitActionStop               = "Stop"
	ShareLimitActionRemove             = "Remove"
	ShareLimitActionEnableSuperSeeding = "EnableSuperSeeding"
	ShareLimitActionRemoveWithContent  = "RemoveWithContent"

	ShareLimitsModeDefault  = "Default"
	ShareLimitsModeMatchAny = "MatchAny"
	ShareLimitsModeMatchAll = "MatchAll"
)

func (c *Client) rpcURL() string {
	raw := strings.TrimSpace(c.cfg.Host)
	if raw == "" {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if strings.HasSuffix(parsed.Path, "/rpc") {
		return parsed.String()
	}
	joined, err := url.JoinPath(parsed.String(), "/transmission/rpc")
	if err != nil {
		return strings.TrimRight(raw, "/") + "/transmission/rpc"
	}
	return joined
}

func (c *Client) rpcCall(ctx context.Context, method string, params interface{}, out interface{}) error {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      rpcRequestID.Add(1),
	})
	if err != nil {
		return err
	}

	resp, err := c.doRPCRequest(ctx, body)
	if err != nil {
		return err
	}
	defer drainAndClose(resp)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrBadCredentials
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrUnexpectedStatus, "transmission rpc %s returned status code %d", method, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNoContent || out == nil {
		return nil
	}

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return err
	}
	if rpcResp.Error != nil {
		msg := rpcResp.Error.Message
		if len(rpcResp.Error.Data) > 0 {
			msg = msg + ": " + string(rpcResp.Error.Data)
		}
		return errors.New("transmission rpc %s failed: %s", method, msg)
	}
	if out == nil {
		return nil
	}
	if len(rpcResp.Result) == 0 || string(rpcResp.Result) == "null" {
		return nil
	}
	return json.Unmarshal(rpcResp.Result, out)
}

func (c *Client) doRPCRequest(ctx context.Context, body []byte) (*http.Response, error) {
	resp, err := c.doRPCRequestOnce(ctx, body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusConflict {
		return resp, nil
	}

	sessionID := resp.Header.Get("X-Transmission-Session-Id")
	drainAndClose(resp)
	if sessionID == "" {
		return nil, errors.Wrap(ErrUnexpectedStatus, "transmission rpc returned 409 without session id")
	}
	c.sessionID = sessionID
	return c.doRPCRequestOnce(ctx, body)
}

func (c *Client) doRPCRequestOnce(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.rpcURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.sessionID != "" {
		req.Header.Set("X-Transmission-Session-Id", c.sessionID)
	}

	user, pass := c.rpcCredentials()
	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}

	return c.http.Do(req)
}

func (c *Client) rpcCredentials() (string, string) {
	if c.cfg.Username != "" || c.cfg.Password != "" {
		return c.cfg.Username, c.cfg.Password
	}
	return c.cfg.BasicUser, c.cfg.BasicPass
}

func (c *Client) Login() error {
	return c.LoginCtx(context.Background())
}

func (c *Client) LoginCtx(ctx context.Context) error {
	var result trSession
	if err := c.rpcCall(ctx, "session_get", map[string]interface{}{
		"fields": []string{"version", "rpc_version_semver", "rpc_version", "download_dir"},
	}, &result); err != nil {
		return errors.Wrap(err, "failed to connect to Transmission instance")
	}
	return nil
}

func (c *Client) GetBuildInfo() (BuildInfo, error) {
	return c.GetBuildInfoCtx(context.Background())
}

func (c *Client) GetBuildInfoCtx(ctx context.Context) (BuildInfo, error) {
	session, err := c.getSession(ctx)
	if err != nil {
		return BuildInfo{}, err
	}
	return BuildInfo{
		Libtorrent: "Transmission",
		Platform:   "unknown",
		Bitness:    0,
		Qt:         "",
		Boost:      "",
		Openssl:    "",
		Zlib:       session.Version,
	}, nil
}

func (c *Client) GetProcessInfo() (ProcessInfo, error) {
	return c.GetProcessInfoCtx(context.Background())
}

func (c *Client) GetProcessInfoCtx(ctx context.Context) (ProcessInfo, error) {
	_ = ctx
	return ProcessInfo{}, nil
}

func (c *Client) Shutdown() error {
	return c.ShutdownCtx(context.Background())
}

func (c *Client) ShutdownCtx(ctx context.Context) error {
	return c.rpcCall(ctx, "session_close", map[string]interface{}{}, nil)
}

func (c *Client) getSession(ctx context.Context, fields ...string) (trSession, error) {
	if len(fields) == 0 {
		fields = []string{
			"version", "rpc_version", "rpc_version_semver", "download_dir",
			"download_queue_enabled", "download_queue_size", "seed_queue_enabled",
			"seed_queue_size", "queue_stalled_enabled", "queue_stalled_minutes",
			"speed_limit_down", "speed_limit_down_enabled", "speed_limit_up",
			"speed_limit_up_enabled", "alt_speed_down", "alt_speed_up",
			"alt_speed_enabled", "peer_port", "peer_limit_global",
			"peer_limit_per_torrent", "pex_enabled", "dht_enabled", "lpd_enabled",
			"utp_enabled", "port_forwarding_enabled", "rename_partial_files",
			"start_added_torrents", "trash_original_torrent_files", "seed_ratio_limit",
			"seed_ratio_limited", "idle_seeding_limit", "idle_seeding_limit_enabled",
			"incomplete_dir", "incomplete_dir_enabled", "script_torrent_added_enabled",
			"script_torrent_added_filename", "blocklist_enabled", "blocklist_url",
		}
	}
	var session trSession
	err := c.rpcCall(ctx, "session_get", map[string]interface{}{"fields": fields}, &session)
	return session, err
}

func (c *Client) GetAppPreferences() (AppPreferences, error) {
	return c.GetAppPreferencesCtx(context.Background())
}

func (c *Client) GetAppPreferencesCtx(ctx context.Context) (AppPreferences, error) {
	session, err := c.getSession(ctx)
	if err != nil {
		return AppPreferences{}, err
	}
	return sessionToPreferences(session), nil
}

func sessionToPreferences(session trSession) AppPreferences {
	return AppPreferences{
		SavePath:                     session.DownloadDir,
		TempPath:                     session.IncompleteDir,
		TempPathEnabled:              session.IncompleteDirEnabled,
		QueueingEnabled:              session.DownloadQueueEnabled || session.SeedQueueEnabled,
		MaxActiveDownloads:           session.DownloadQueueSize,
		MaxActiveUploads:             session.SeedQueueSize,
		MaxActiveTorrents:            max(session.DownloadQueueSize+session.SeedQueueSize, 0),
		DlLimit:                      int(kbToBytes(session.SpeedLimitDown, session.SpeedLimitDownEnabled)),
		UpLimit:                      int(kbToBytes(session.SpeedLimitUp, session.SpeedLimitUpEnabled)),
		AltDlLimit:                   int(session.AltSpeedDown),
		AltUpLimit:                   int(session.AltSpeedUp),
		SchedulerEnabled:             session.AltSpeedEnabled,
		ListenPort:                   session.PeerPort,
		MaxConnec:                    session.PeerLimitGlobal,
		MaxConnecPerTorrent:          session.PeerLimitPerTorrent,
		Pex:                          session.PEXEnabled,
		Dht:                          session.DHTEnabled,
		Lsd:                          session.LPDEnabled,
		Upnp:                         session.PortForwardingEnabled,
		IncompleteFilesExt:           session.RenamePartialFiles,
		StartPausedEnabled:           !session.StartAddedTorrents,
		AutoDeleteMode:               boolToInt(session.TrashOriginalTorrentFiles),
		MaxRatio:                     session.SeedRatioLimit,
		MaxRatioEnabled:              session.SeedRatioLimited,
		MaxSeedingTime:               session.IdleSeedingLimit,
		MaxSeedingTimeEnabled:        session.IdleSeedingLimitEnabled,
		AutorunOnTorrentAddedEnabled: session.ScriptTorrentAddedEnabled,
		AutorunOnTorrentAddedProgram: session.ScriptTorrentAddedFilename,
		IPFilterEnabled:              session.BlocklistEnabled,
		IPFilterPath:                 session.BlocklistURL,
		UseSubcategories:             false,
		AutoTmmEnabled:               true,
		TorrentContentLayout:         string(ContentLayoutOriginal),
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func kbToBytes(v int64, enabled bool) int64 {
	if !enabled || v <= 0 {
		return 0
	}
	return v * 1000
}

func bytesToKB(v int64) int64 {
	if v <= 0 {
		return 0
	}
	return v / 1000
}

func (c *Client) SetPreferences(prefs map[string]interface{}) error {
	return c.SetPreferencesCtx(context.Background(), prefs)
}

func (c *Client) SetPreferencesCtx(ctx context.Context, prefs map[string]interface{}) error {
	params := make(map[string]interface{})
	for key, value := range prefs {
		switch key {
		case "save_path":
			params["download_dir"] = value
		case "temp_path":
			params["incomplete_dir"] = value
		case "temp_path_enabled":
			params["incomplete_dir_enabled"] = value
		case "queueing_enabled":
			enabled := truthy(value)
			params["download_queue_enabled"] = enabled
			params["seed_queue_enabled"] = enabled
		case "max_active_downloads":
			params["download_queue_size"] = intValue(value)
		case "max_active_uploads":
			params["seed_queue_size"] = intValue(value)
		case "dl_limit":
			limit := int64Value(value)
			params["speed_limit_down"] = bytesToKB(limit)
			params["speed_limit_down_enabled"] = limit > 0
		case "up_limit":
			limit := int64Value(value)
			params["speed_limit_up"] = bytesToKB(limit)
			params["speed_limit_up_enabled"] = limit > 0
		case "alt_dl_limit":
			params["alt_speed_down"] = int64Value(value)
		case "alt_up_limit":
			params["alt_speed_up"] = int64Value(value)
		case "scheduler_enabled":
			params["alt_speed_enabled"] = truthy(value)
		case "listen_port":
			params["peer_port"] = intValue(value)
		case "max_connec":
			params["peer_limit_global"] = intValue(value)
		case "max_connec_per_torrent":
			params["peer_limit_per_torrent"] = intValue(value)
		case "pex":
			params["pex_enabled"] = truthy(value)
		case "dht":
			params["dht_enabled"] = truthy(value)
		case "lsd":
			params["lpd_enabled"] = truthy(value)
		case "upnp":
			params["port_forwarding_enabled"] = truthy(value)
		case "incomplete_files_ext":
			params["rename_partial_files"] = truthy(value)
		case "start_paused_enabled":
			params["start_added_torrents"] = !truthy(value)
		case "max_ratio":
			params["seed_ratio_limit"] = floatValue(value)
		case "max_ratio_enabled":
			params["seed_ratio_limited"] = truthy(value)
		case "max_seeding_time":
			params["idle_seeding_limit"] = intValue(value)
		case "max_seeding_time_enabled":
			params["idle_seeding_limit_enabled"] = truthy(value)
		case "autorun_on_torrent_added_enabled":
			params["script_torrent_added_enabled"] = truthy(value)
		case "autorun_on_torrent_added_program":
			params["script_torrent_added_filename"] = value
		case "ip_filter_enabled":
			params["blocklist_enabled"] = truthy(value)
		case "ip_filter_path":
			params["blocklist_url"] = value
		}
	}
	if len(params) == 0 {
		return nil
	}
	return c.rpcCall(ctx, "session_set", params, nil)
}

func truthy(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, _ := strconv.ParseBool(t)
		return b
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	default:
		return false
	}
}

func intValue(v interface{}) int {
	return int(int64Value(v))
}

func int64Value(v interface{}) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case int32:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	case json.Number:
		i, _ := t.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(t, 10, 64)
		return i
	default:
		return 0
	}
}

func floatValue(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

func (c *Client) GetDirectoryContent(dirPath string, withMetadata bool) (any, error) {
	return c.GetDirectoryContentCtx(context.Background(), dirPath, withMetadata)
}

func (c *Client) GetDirectoryContentCtx(ctx context.Context, dirPath string, withMetadata bool) (any, error) {
	_ = ctx
	_ = dirPath
	_ = withMetadata
	return nil, unsupported("remote directory autocomplete")
}

func (c *Client) GetDefaultSavePath() (string, error) {
	return c.GetDefaultSavePathCtx(context.Background())
}

func (c *Client) GetDefaultSavePathCtx(ctx context.Context) (string, error) {
	session, err := c.getSession(ctx, "download_dir")
	if err != nil {
		return "", err
	}
	return session.DownloadDir, nil
}

func (c *Client) GetTorrents(o TorrentFilterOptions) ([]Torrent, error) {
	return c.GetTorrentsCtx(context.Background(), o)
}

func (c *Client) GetTorrentsCtx(ctx context.Context, o TorrentFilterOptions) ([]Torrent, error) {
	var result struct {
		Torrents []trTorrent `json:"torrents"`
	}
	params := map[string]interface{}{
		"fields": torrentFields,
	}
	if len(o.Hashes) > 0 {
		params["ids"] = o.Hashes
	}
	if err := c.rpcCall(ctx, "torrent_get", params, &result); err != nil {
		return nil, errors.Wrap(err, "get torrents error")
	}

	torrents := make([]Torrent, 0, len(result.Torrents))
	for _, tr := range result.Torrents {
		torrent := tr.toQBT()
		if matchesTorrentFilter(torrent, o) {
			torrents = append(torrents, torrent)
		}
	}

	return applyTorrentFilterOptions(torrents, o), nil
}

func (tr trTorrent) toQBT() Torrent {
	hash := strings.ToUpper(strings.TrimSpace(tr.HashString))
	progress := tr.PercentDone
	if progress == 0 && tr.PercentComplete > 0 {
		progress = tr.PercentComplete
	}
	total := tr.TotalSize
	if total == 0 {
		total = tr.SizeWhenDone
	}
	savePath := tr.DownloadDir
	contentPath := filepath.Join(savePath, filepath.FromSlash(tr.Name))
	if len(tr.Files) == 1 {
		contentPath = filepath.Join(savePath, filepath.FromSlash(tr.Files[0].Name))
	}

	trackers := tr.toTrackers()
	trackerURL := ""
	if len(trackers) > 0 {
		trackerURL = trackers[0].Url
	}

	completionOn := tr.DoneDate
	if completionOn == 0 && (tr.IsFinished || progress >= 1) {
		completionOn = tr.ActivityDate
	}

	return Torrent{
		AddedOn:                  tr.AddedDate,
		AmountLeft:               tr.LeftUntilDone,
		AutoManaged:              true,
		Availability:             availabilityScore(tr.Availability),
		Category:                 tr.Group,
		Comment:                  tr.Comment,
		Completed:                tr.SizeWhenDone - tr.LeftUntilDone,
		CompletionOn:             completionOn,
		CreatedBy:                tr.Creator,
		ContentPath:              contentPath,
		DlLimit:                  kbToBytes(tr.DownloadLimit, tr.DownloadLimited),
		DlSpeed:                  tr.RateDownload,
		DownloadPath:             savePath,
		Downloaded:               tr.DownloadedEver,
		DownloadedSession:        0,
		ETA:                      tr.Eta,
		ForceStart:               tr.Status == 4 && tr.QueuePosition < 0,
		Hash:                     hash,
		InfohashV1:               hash,
		MagnetURI:                tr.MagnetLink,
		MaxRatio:                 tr.SeedRatioLimit,
		MaxSeedingTime:           tr.SecondsSeeding,
		MaxInactiveSeedingTime:   tr.SeedIdleLimit,
		Name:                     tr.Name,
		NumComplete:              firstSeederCount(tr.TrackerStats),
		NumIncomplete:            firstLeecherCount(tr.TrackerStats),
		NumLeechs:                tr.PeersSendingToUs,
		NumSeeds:                 tr.PeersGettingFromUs,
		Priority:                 tr.QueuePosition,
		Private:                  tr.IsPrivate,
		Progress:                 progress,
		Ratio:                    tr.UploadRatio,
		RatioLimit:               tr.SeedRatioLimit,
		SavePath:                 savePath,
		SeedingTime:              tr.SecondsSeeding,
		SeedingTimeLimit:         tr.SeedIdleLimit,
		InactiveSeedingTimeLimit: tr.SeedIdleLimit,
		SeenComplete:             completionOn,
		SequentialDownload:       tr.SequentialDownload,
		Size:                     tr.SizeWhenDone,
		State:                    tr.toState(progress),
		Tags:                     strings.Join(tr.Labels, ", "),
		TimeActive:               tr.SecondsDownloading + tr.SecondsSeeding,
		TotalSize:                total,
		Tracker:                  trackerURL,
		TrackersCount:            int64(len(trackers)),
		UpLimit:                  kbToBytes(tr.UploadLimit, tr.UploadLimited),
		Uploaded:                 tr.UploadedEver,
		UpSpeed:                  tr.RateUpload,
		Trackers:                 trackers,
	}
}

func (tr trTorrent) toState(progress float64) TorrentState {
	if tr.Error != 0 {
		return TorrentStateError
	}
	complete := progress >= 1 || tr.IsFinished
	switch tr.Status {
	case 0:
		if complete {
			return TorrentStateStoppedUp
		}
		return TorrentStateStoppedDl
	case 1:
		return TorrentStateCheckingResumeData
	case 2:
		if complete {
			return TorrentStateCheckingUp
		}
		return TorrentStateCheckingDl
	case 3:
		return TorrentStateQueuedDl
	case 4:
		if tr.IsStalled {
			return TorrentStateStalledDl
		}
		return TorrentStateDownloading
	case 5:
		return TorrentStateQueuedUp
	case 6:
		if tr.IsStalled {
			return TorrentStateStalledUp
		}
		return TorrentStateUploading
	default:
		return TorrentStateUnknown
	}
}

func (tr trTorrent) toTrackers() []TorrentTracker {
	if len(tr.TrackerStats) > 0 {
		trackers := make([]TorrentTracker, 0, len(tr.TrackerStats))
		for _, stat := range tr.TrackerStats {
			trackers = append(trackers, TorrentTracker{
				Url:           stat.Announce,
				Status:        trackerStatus(stat),
				NumPeers:      stat.LastAnnouncePeerCount,
				NumSeeds:      stat.SeederCount,
				NumLeechers:   stat.LeecherCount,
				NumDownloaded: stat.DownloadCount,
				Message:       firstNonEmpty(stat.LastAnnounceResult, stat.LastScrapeResult),
			})
		}
		return trackers
	}
	trackers := make([]TorrentTracker, 0, len(tr.Trackers))
	for _, tracker := range tr.Trackers {
		trackers = append(trackers, TorrentTracker{Url: tracker.Announce, Status: TrackerStatusNotContacted})
	}
	return trackers
}

func trackerStatus(stat trTrackerStat) TrackerStatus {
	if stat.LastAnnounceSucceeded || stat.LastScrapeSucceeded {
		return TrackerStatusOK
	}
	if stat.AnnounceState == 1 || stat.ScrapeState == 1 {
		return TrackerStatusUpdating
	}
	if stat.LastAnnounceTimedOut || stat.LastScrapeTimedOut {
		return TrackerStatusUnreachable
	}
	if stat.LastAnnounceResult != "" || stat.LastScrapeResult != "" {
		return TrackerStatusTrackerError
	}
	return TrackerStatusNotContacted
}

func firstSeederCount(stats []trTrackerStat) int64 {
	for _, stat := range stats {
		if stat.SeederCount >= 0 {
			return int64(stat.SeederCount)
		}
	}
	return 0
}

func firstLeecherCount(stats []trTrackerStat) int64 {
	for _, stat := range stats {
		if stat.LeecherCount >= 0 {
			return int64(stat.LeecherCount)
		}
	}
	return 0
}

func availabilityScore(availability []int) float64 {
	if len(availability) == 0 {
		return 0
	}
	available := 0
	for _, value := range availability {
		if value != 0 {
			available++
		}
	}
	return float64(available) / float64(len(availability))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (c *Client) GetTorrentsActiveDownloads() ([]Torrent, error) {
	return c.GetTorrentsActiveDownloadsCtx(context.Background())
}

func (c *Client) GetTorrentsActiveDownloadsCtx(ctx context.Context) ([]Torrent, error) {
	return c.GetTorrentsCtx(ctx, TorrentFilterOptions{Filter: TorrentFilterDownloading})
}

func (c *Client) GetTorrentProperties(hash string) (TorrentProperties, error) {
	return c.GetTorrentPropertiesCtx(context.Background(), hash)
}

func (c *Client) GetTorrentPropertiesCtx(ctx context.Context, hash string) (TorrentProperties, error) {
	torrents, err := c.fetchTRTorrents(ctx, []string{hash}, torrentFields...)
	if err != nil {
		return TorrentProperties{}, err
	}
	if len(torrents) == 0 {
		return TorrentProperties{}, ErrTorrentNotFound
	}
	tr := torrents[0]
	qbt := tr.toQBT()
	return TorrentProperties{
		AdditionDate:           int(tr.AddedDate),
		Comment:                tr.Comment,
		CompletionDate:         int(tr.DoneDate),
		CreatedBy:              tr.Creator,
		CreationDate:           int(tr.DateCreated),
		DlLimit:                int(qbt.DlLimit),
		DlSpeed:                int(qbt.DlSpeed),
		DownloadPath:           qbt.SavePath,
		Eta:                    int(qbt.ETA),
		Hash:                   qbt.Hash,
		InfohashV1:             qbt.InfohashV1,
		IsPrivate:              qbt.Private,
		Name:                   qbt.Name,
		NbConnections:          int(tr.PeersConnected),
		NbConnectionsLimit:     tr.PeerLimit,
		Peers:                  int(tr.PeersSendingToUs),
		PeersTotal:             int(firstLeecherCount(tr.TrackerStats)),
		PieceSize:              int(tr.PieceSize),
		PiecesHave:             int(float64(tr.PieceCount) * qbt.Progress),
		PiecesNum:              tr.PieceCount,
		SavePath:               qbt.SavePath,
		SeedingTime:            int(qbt.SeedingTime),
		Seeds:                  int(tr.PeersGettingFromUs),
		SeedsTotal:             int(firstSeederCount(tr.TrackerStats)),
		ShareRatio:             qbt.Ratio,
		TimeElapsed:            int(qbt.TimeActive),
		TotalDownloaded:        qbt.Downloaded,
		TotalDownloadedSession: qbt.DownloadedSession,
		TotalSize:              qbt.TotalSize,
		TotalUploaded:          qbt.Uploaded,
		TotalUploadedSession:   qbt.UploadedSession,
		TotalWasted:            tr.CorruptEver,
		UpLimit:                int(qbt.UpLimit),
		UpSpeed:                int(qbt.UpSpeed),
	}, nil
}

func (c *Client) fetchTRTorrents(ctx context.Context, hashes []string, fields ...string) ([]trTorrent, error) {
	var result struct {
		Torrents []trTorrent `json:"torrents"`
	}
	params := map[string]interface{}{"fields": fields}
	if len(hashes) > 0 {
		params["ids"] = hashes
	}
	err := c.rpcCall(ctx, "torrent_get", params, &result)
	return result.Torrents, err
}

func (c *Client) GetTorrentsRaw() (string, error) {
	return c.GetTorrentsRawCtx(context.Background())
}

func (c *Client) GetTorrentsRawCtx(ctx context.Context) (string, error) {
	torrents, err := c.GetTorrentsCtx(ctx, TorrentFilterOptions{})
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(torrents)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Client) GetTorrentTrackers(hash string) ([]TorrentTracker, error) {
	return c.GetTorrentTrackersCtx(context.Background(), hash)
}

func (c *Client) GetTorrentTrackersCtx(ctx context.Context, hash string) ([]TorrentTracker, error) {
	torrents, err := c.fetchTRTorrents(ctx, []string{hash}, "hash_string", "trackers", "tracker_stats")
	if err != nil {
		return nil, err
	}
	if len(torrents) == 0 {
		return nil, ErrTorrentNotFound
	}
	return torrents[0].toTrackers(), nil
}

func (c *Client) AddTorrentFromMemory(buf []byte, options map[string]string) (*TorrentAddResponse, error) {
	return c.AddTorrentFromMemoryCtx(context.Background(), buf, options)
}

func (c *Client) AddTorrentFromMemoryCtx(ctx context.Context, buf []byte, options map[string]string) (*TorrentAddResponse, error) {
	params := addParamsFromOptions(options)
	params["metainfo"] = base64.StdEncoding.EncodeToString(buf)
	return c.addTorrent(ctx, params)
}

func (c *Client) AddTorrentsFromMemory(files [][]byte, options map[string]string) (*TorrentAddResponse, error) {
	return c.AddTorrentsFromMemoryCtx(context.Background(), files, options)
}

func (c *Client) AddTorrentsFromMemoryCtx(ctx context.Context, files [][]byte, options map[string]string) (*TorrentAddResponse, error) {
	response := &TorrentAddResponse{}
	for _, file := range files {
		added, err := c.AddTorrentFromMemoryCtx(ctx, file, options)
		if err != nil {
			response.FailureCount++
			continue
		}
		response.SuccessCount += added.SuccessCount
		response.PendingCount += added.PendingCount
		response.FailureCount += added.FailureCount
		response.AddedTorrentIds = append(response.AddedTorrentIds, added.AddedTorrentIds...)
	}
	return response, nil
}

func (c *Client) AddTorrentFromFile(filePath string, options map[string]string) (*TorrentAddResponse, error) {
	return c.AddTorrentFromFileCtx(context.Background(), filePath, options)
}

func (c *Client) AddTorrentFromFileCtx(ctx context.Context, filePath string, options map[string]string) (*TorrentAddResponse, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return c.AddTorrentFromMemoryCtx(ctx, data, options)
}

func (c *Client) AddTorrentFromUrl(rawURL string, options map[string]string) (*TorrentAddResponse, error) {
	return c.AddTorrentFromUrlCtx(context.Background(), rawURL, options)
}

func (c *Client) AddTorrentFromUrlCtx(ctx context.Context, rawURL string, options map[string]string) (*TorrentAddResponse, error) {
	params := addParamsFromOptions(options)
	params["filename"] = rawURL
	return c.addTorrent(ctx, params)
}

func (c *Client) AddTorrentsFromUrlsCtx(ctx context.Context, urls []string, options map[string]string) (*TorrentAddResponse, error) {
	response := &TorrentAddResponse{}
	for _, rawURL := range urls {
		added, err := c.AddTorrentFromUrlCtx(ctx, rawURL, options)
		if err != nil {
			response.FailureCount++
			continue
		}
		response.SuccessCount += added.SuccessCount
		response.PendingCount += added.PendingCount
		response.FailureCount += added.FailureCount
		response.AddedTorrentIds = append(response.AddedTorrentIds, added.AddedTorrentIds...)
	}
	return response, nil
}

func addParamsFromOptions(options map[string]string) map[string]interface{} {
	params := make(map[string]interface{})
	for key, value := range options {
		switch key {
		case "savepath":
			params["download_dir"] = value
		case "category":
			params["group"] = value
		case "tags":
			params["labels"] = splitCSV(value)
		case "paused", "stopped":
			if truthy(value) {
				params["paused"] = true
			}
		case "skip_checking":
			params["paused"] = truthy(value)
		case "upLimit":
			limit := int64Value(value)
			if limit > 0 {
				params["upload_limit"] = bytesToKB(limit)
			}
		case "dlLimit":
			limit := int64Value(value)
			if limit > 0 {
				params["download_limit"] = bytesToKB(limit)
			}
		case "ratioLimit":
			params["seed_ratio_limit"] = floatValue(value)
			params["seed_ratio_mode"] = 1
		case "seedingTimeLimit":
			params["seed_idle_limit"] = intValue(value)
			params["seed_idle_mode"] = 1
		case "sequentialDownload":
			params["sequential_download"] = truthy(value)
		}
	}
	return params
}

func (c *Client) addTorrent(ctx context.Context, params map[string]interface{}) (*TorrentAddResponse, error) {
	var result struct {
		TorrentAdded     *trTorrent `json:"torrent_added"`
		TorrentDuplicate *trTorrent `json:"torrent_duplicate"`
	}
	if err := c.rpcCall(ctx, "torrent_add", params, &result); err != nil {
		return nil, errors.Wrap(err, "could not add torrent")
	}

	response := &TorrentAddResponse{}
	if result.TorrentAdded != nil {
		response.SuccessCount = 1
		response.AddedTorrentIds = []string{strings.ToUpper(result.TorrentAdded.HashString)}
	} else if result.TorrentDuplicate != nil {
		response.PendingCount = 1
		response.AddedTorrentIds = []string{strings.ToUpper(result.TorrentDuplicate.HashString)}
	} else {
		response.SuccessCount = 1
	}
	return response, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '|' })
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func idsParam(hashes []string) map[string]interface{} {
	return map[string]interface{}{"ids": hashes}
}

func (c *Client) DeleteTorrents(hashes []string, deleteFiles bool) error {
	return c.DeleteTorrentsCtx(context.Background(), hashes, deleteFiles)
}

func (c *Client) DeleteTorrentsCtx(ctx context.Context, hashes []string, deleteFiles bool) error {
	return c.rpcCall(ctx, "torrent_remove", map[string]interface{}{
		"ids":               hashes,
		"delete_local_data": deleteFiles,
	}, nil)
}

func (c *Client) ReAnnounceTorrents(hashes []string) error {
	return c.ReAnnounceTorrentsCtx(context.Background(), hashes)
}

func (c *Client) ReAnnounceTorrentsCtx(ctx context.Context, hashes []string) error {
	return c.rpcCall(ctx, "torrent_reannounce", idsParam(hashes), nil)
}

func (c *Client) GetTransferInfo() (*TransferInfo, error) {
	return c.GetTransferInfoCtx(context.Background())
}

func (c *Client) GetTransferInfoCtx(ctx context.Context) (*TransferInfo, error) {
	state, err := c.serverState(ctx)
	if err != nil {
		return nil, err
	}
	return &TransferInfo{
		ConnectionStatus: ConnectionStatusConnected,
		DlInfoData:       state.DlInfoData,
		DlInfoSpeed:      state.DlInfoSpeed,
		DlRateLimit:      state.DlRateLimit,
		UpInfoData:       state.UpInfoData,
		UpInfoSpeed:      state.UpInfoSpeed,
		UpRateLimit:      state.UpRateLimit,
	}, nil
}

func (c *Client) BanPeers(peers []string) error {
	return c.BanPeersCtx(context.Background(), peers)
}

func (c *Client) BanPeersCtx(ctx context.Context, peers []string) error {
	_ = ctx
	_ = peers
	return unsupported("peer banning")
}

func (c *Client) SyncMainDataCtx(ctx context.Context, rid int64) (*MainData, error) {
	data, _, err := c.SyncMainDataCtxWithRaw(ctx, rid)
	return data, err
}

func (c *Client) SyncMainDataCtxWithRaw(ctx context.Context, rid int64) (*MainData, map[string]interface{}, error) {
	torrents, err := c.GetTorrentsCtx(ctx, TorrentFilterOptions{})
	if err != nil {
		return nil, nil, err
	}
	state, err := c.serverState(ctx)
	if err != nil {
		return nil, nil, err
	}

	torrentMap := make(map[string]Torrent, len(torrents))
	categories := make(map[string]Category)
	tagSet := make(map[string]struct{})
	trackerMap := make(map[string][]string)
	for _, torrent := range torrents {
		torrentMap[torrent.Hash] = torrent
		if torrent.Category != "" {
			categories[torrent.Category] = Category{Name: torrent.Category, SavePath: torrent.SavePath}
		}
		for _, tag := range splitCSV(torrent.Tags) {
			tagSet[tag] = struct{}{}
		}
		if torrent.Tracker != "" {
			trackerMap[torrent.Tracker] = append(trackerMap[torrent.Tracker], torrent.Hash)
		}
	}
	tags := slices.Sorted(maps.Keys(tagSet))

	data := &MainData{
		Rid:         time.Now().UnixNano(),
		FullUpdate:  true,
		Torrents:    torrentMap,
		Categories:  categories,
		Tags:        tags,
		Trackers:    trackerMap,
		ServerState: state,
	}
	raw := map[string]interface{}{
		"rid":          data.Rid,
		"full_update":  true,
		"torrents":     torrentMap,
		"categories":   categories,
		"tags":         tags,
		"trackers":     trackerMap,
		"server_state": state,
	}
	return data, raw, nil
}

func (c *Client) serverState(ctx context.Context) (ServerState, error) {
	var stats trStats
	if err := c.rpcCall(ctx, "session_stats", map[string]interface{}{}, &stats); err != nil {
		return ServerState{}, err
	}
	session, err := c.getSession(ctx, "speed_limit_down", "speed_limit_down_enabled", "speed_limit_up", "speed_limit_up_enabled")
	if err != nil {
		return ServerState{}, err
	}
	free, _ := c.GetFreeSpaceOnDiskCtx(ctx)
	return ServerState{
		AlltimeDl:        stats.CumulativeStats.DownloadedBytes,
		AlltimeUl:        stats.CumulativeStats.UploadedBytes,
		ConnectionStatus: string(ConnectionStatusConnected),
		DlInfoData:       stats.CurrentStats.DownloadedBytes,
		DlInfoSpeed:      stats.DownloadSpeed,
		DlRateLimit:      kbToBytes(session.SpeedLimitDown, session.SpeedLimitDownEnabled),
		FreeSpaceOnDisk:  free,
		Queueing:         true,
		RefreshInterval:  1500,
		UpInfoData:       stats.CurrentStats.UploadedBytes,
		UpInfoSpeed:      stats.UploadSpeed,
		UpRateLimit:      kbToBytes(session.SpeedLimitUp, session.SpeedLimitUpEnabled),
	}, nil
}

func (c *Client) Pause(hashes []string) error {
	return c.PauseCtx(context.Background(), hashes)
}

func (c *Client) Stop(hashes []string) error {
	return c.StopCtx(context.Background(), hashes)
}

func (c *Client) StopCtx(ctx context.Context, hashes []string) error {
	return c.PauseCtx(ctx, hashes)
}

func (c *Client) PauseCtx(ctx context.Context, hashes []string) error {
	return c.rpcCall(ctx, "torrent_stop", idsParam(hashes), nil)
}

func (c *Client) Resume(hashes []string) error {
	return c.ResumeCtx(context.Background(), hashes)
}

func (c *Client) Start(hashes []string) error {
	return c.StartCtx(context.Background(), hashes)
}

func (c *Client) StartCtx(ctx context.Context, hashes []string) error {
	return c.ResumeCtx(ctx, hashes)
}

func (c *Client) ResumeCtx(ctx context.Context, hashes []string) error {
	return c.rpcCall(ctx, "torrent_start", idsParam(hashes), nil)
}

func (c *Client) SetForceStart(hashes []string, value bool) error {
	return c.SetForceStartCtx(context.Background(), hashes, value)
}

func (c *Client) SetForceStartCtx(ctx context.Context, hashes []string, value bool) error {
	method := "torrent_start"
	if value {
		method = "torrent_start_now"
	}
	return c.rpcCall(ctx, method, idsParam(hashes), nil)
}

func (c *Client) Recheck(hashes []string) error {
	return c.RecheckCtx(context.Background(), hashes)
}

func (c *Client) RecheckCtx(ctx context.Context, hashes []string) error {
	return c.rpcCall(ctx, "torrent_verify", idsParam(hashes), nil)
}

func (c *Client) SetAutoManagement(hashes []string, enable bool) error {
	return c.SetAutoManagementCtx(context.Background(), hashes, enable)
}

func (c *Client) SetAutoManagementCtx(ctx context.Context, hashes []string, enable bool) error {
	_ = ctx
	_ = hashes
	_ = enable
	return nil
}

func (c *Client) SetLocation(hashes []string, location string) error {
	return c.SetLocationCtx(context.Background(), hashes, location)
}

func (c *Client) SetLocationCtx(ctx context.Context, hashes []string, location string) error {
	return c.rpcCall(ctx, "torrent_set_location", map[string]interface{}{
		"ids":      hashes,
		"location": location,
		"move":     true,
	}, nil)
}

func (c *Client) CreateCategory(category string, categoryPath string) error {
	return c.CreateCategoryCtx(context.Background(), category, categoryPath)
}

func (c *Client) CreateCategoryCtx(ctx context.Context, category string, categoryPath string) error {
	_ = categoryPath
	return c.rpcCall(ctx, "group_set", map[string]interface{}{"name": category}, nil)
}

func (c *Client) EditCategory(category string, categoryPath string) error {
	return c.EditCategoryCtx(context.Background(), category, categoryPath)
}

func (c *Client) EditCategoryCtx(ctx context.Context, category string, categoryPath string) error {
	_ = categoryPath
	return c.rpcCall(ctx, "group_set", map[string]interface{}{"name": category}, nil)
}

func (c *Client) RemoveCategories(categories []string) error {
	return c.RemoveCategoriesCtx(context.Background(), categories)
}

func (c *Client) RemoveCategoriesCtx(ctx context.Context, categories []string) error {
	torrents, err := c.GetTorrentsCtx(ctx, TorrentFilterOptions{})
	if err != nil {
		return err
	}
	categorySet := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		categorySet[category] = struct{}{}
	}
	hashes := make([]string, 0)
	for _, torrent := range torrents {
		if _, ok := categorySet[torrent.Category]; ok {
			hashes = append(hashes, torrent.Hash)
		}
	}
	if len(hashes) == 0 {
		return nil
	}
	return c.SetCategoryCtx(ctx, hashes, "")
}

func (c *Client) SetCategory(hashes []string, category string) error {
	return c.SetCategoryCtx(context.Background(), hashes, category)
}

func (c *Client) SetCategoryCtx(ctx context.Context, hashes []string, category string) error {
	return c.rpcCall(ctx, "torrent_set", map[string]interface{}{
		"ids":   hashes,
		"group": category,
	}, nil)
}

func (c *Client) SetComment(hashes []string, comment string) error {
	return c.SetCommentCtx(context.Background(), hashes, comment)
}

func (c *Client) SetCommentCtx(ctx context.Context, hashes []string, comment string) error {
	return c.rpcCall(ctx, "torrent_set", map[string]interface{}{
		"ids":     hashes,
		"comment": comment,
	}, nil)
}

func (c *Client) GetCategories() (map[string]Category, error) {
	return c.GetCategoriesCtx(context.Background())
}

func (c *Client) GetCategoriesCtx(ctx context.Context) (map[string]Category, error) {
	torrents, err := c.GetTorrentsCtx(ctx, TorrentFilterOptions{})
	if err != nil {
		return nil, err
	}
	categories := make(map[string]Category)
	for _, torrent := range torrents {
		if torrent.Category != "" {
			categories[torrent.Category] = Category{Name: torrent.Category, SavePath: torrent.SavePath}
		}
	}
	return categories, nil
}

func (c *Client) GetFilesInformation(hash string) (*TorrentFiles, error) {
	return c.GetFilesInformationCtx(context.Background(), hash)
}

func (c *Client) GetFilesInformationCtx(ctx context.Context, hash string) (*TorrentFiles, error) {
	torrents, err := c.fetchTRTorrents(ctx, []string{hash}, "hash_string", "files", "file_stats", "priorities", "wanted")
	if err != nil {
		return nil, err
	}
	if len(torrents) == 0 {
		return nil, ErrTorrentNotFound
	}
	files := transmissionFilesToQBT(torrents[0])
	return &files, nil
}

func transmissionFilesToQBT(torrent trTorrent) TorrentFiles {
	files := make(TorrentFiles, 0, len(torrent.Files))
	for i, file := range torrent.Files {
		stat := trFileStat{Wanted: true}
		if i < len(torrent.FileStats) {
			stat = torrent.FileStats[i]
		}
		priority := 4
		if !stat.Wanted {
			priority = 0
		} else if stat.Priority < 0 {
			priority = 1
		} else if stat.Priority > 0 {
			priority = 7
		}
		progress := float32(0)
		if file.Length > 0 {
			progress = float32(float64(file.BytesCompleted) / float64(file.Length))
		}
		files = append(files, TorrentFile{
			Index:      i,
			IsSeed:     file.BytesCompleted >= file.Length && file.Length > 0,
			Name:       file.Name,
			PieceRange: []int{file.BeginPiece, file.EndPiece},
			Priority:   priority,
			Progress:   progress,
			Size:       file.Length,
		})
	}
	return files
}

func (c *Client) SetFilePriority(hash string, IDs string, priority int) error {
	return c.SetFilePriorityCtx(context.Background(), hash, IDs, priority)
}

func (c *Client) SetFilePriorityCtx(ctx context.Context, hash string, IDs string, priority int) error {
	ids, err := parseFileIDs(IDs)
	if err != nil {
		return ErrInvalidPriority
	}
	params := map[string]interface{}{"ids": []string{hash}}
	switch {
	case priority <= 0:
		params["files_unwanted"] = ids
	case priority <= 1:
		params["files_wanted"] = ids
		params["priority_low"] = ids
	case priority >= 6:
		params["files_wanted"] = ids
		params["priority_high"] = ids
	default:
		params["files_wanted"] = ids
		params["priority_normal"] = ids
	}
	return c.rpcCall(ctx, "torrent_set", params, nil)
}

func parseFileIDs(raw string) ([]int, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '|' || r == ',' || r == ' '
	})
	ids := make([]int, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (c *Client) ExportTorrent(hash string) ([]byte, error) {
	return c.ExportTorrentCtx(context.Background(), hash)
}

func (c *Client) ExportTorrentCtx(ctx context.Context, hash string) ([]byte, error) {
	torrents, err := c.fetchTRTorrents(ctx, []string{hash}, "hash_string", "torrent_file")
	if err != nil {
		return nil, err
	}
	if len(torrents) == 0 || torrents[0].TorrentFile == "" {
		return nil, ErrTorrentNotFound
	}
	return os.ReadFile(torrents[0].TorrentFile)
}

func (c *Client) RenameFile(hash, oldPath, newPath string) error {
	return c.RenameFileCtx(context.Background(), hash, oldPath, newPath)
}

func (c *Client) RenameFileCtx(ctx context.Context, hash, oldPath, newPath string) error {
	return c.rpcCall(ctx, "torrent_rename_path", map[string]interface{}{
		"ids":  []string{hash},
		"path": oldPath,
		"name": path.Base(newPath),
	}, nil)
}

func (c *Client) RenameFolder(hash, oldPath, newPath string) error {
	return c.RenameFolderCtx(context.Background(), hash, oldPath, newPath)
}

func (c *Client) RenameFolderCtx(ctx context.Context, hash, oldPath, newPath string) error {
	return c.RenameFileCtx(ctx, hash, oldPath, newPath)
}

func (c *Client) SetTorrentName(hash string, name string) error {
	return c.SetTorrentNameCtx(context.Background(), hash, name)
}

func (c *Client) SetTorrentNameCtx(ctx context.Context, hash string, name string) error {
	torrents, err := c.fetchTRTorrents(ctx, []string{hash}, "hash_string", "name")
	if err != nil {
		return err
	}
	if len(torrents) == 0 {
		return ErrTorrentNotFound
	}
	return c.rpcCall(ctx, "torrent_rename_path", map[string]interface{}{
		"ids":  []string{hash},
		"path": torrents[0].Name,
		"name": name,
	}, nil)
}

func (c *Client) GetTags() ([]string, error) {
	return c.GetTagsCtx(context.Background())
}

func (c *Client) GetTagsCtx(ctx context.Context) ([]string, error) {
	torrents, err := c.GetTorrentsCtx(ctx, TorrentFilterOptions{})
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	for _, torrent := range torrents {
		for _, tag := range splitCSV(torrent.Tags) {
			set[tag] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(set)), nil
}

func (c *Client) CreateTags(tags []string) error {
	return c.CreateTagsCtx(context.Background(), tags)
}

func (c *Client) CreateTagsCtx(ctx context.Context, tags []string) error {
	_ = ctx
	_ = tags
	return nil
}

func (c *Client) AddTags(hashes []string, tags string) error {
	return c.AddTagsCtx(context.Background(), hashes, tags)
}

func (c *Client) AddTagsCtx(ctx context.Context, hashes []string, tags string) error {
	current, err := c.labelMap(ctx, hashes)
	if err != nil {
		return err
	}
	toAdd := splitCSV(tags)
	for _, hash := range hashes {
		labels := append(current[strings.ToUpper(hash)], toAdd...)
		if err := c.setLabels(ctx, []string{hash}, labels); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) SetTags(ctx context.Context, hashes []string, tags string) error {
	return c.setLabels(ctx, hashes, splitCSV(tags))
}

func (c *Client) DeleteTags(tags []string) error {
	return c.DeleteTagsCtx(context.Background(), tags)
}

func (c *Client) DeleteTagsCtx(ctx context.Context, tags []string) error {
	torrents, err := c.GetTorrentsCtx(ctx, TorrentFilterOptions{})
	if err != nil {
		return err
	}
	hashes := make([]string, 0, len(torrents))
	for _, torrent := range torrents {
		hashes = append(hashes, torrent.Hash)
	}
	return c.RemoveTagsCtx(ctx, hashes, strings.Join(tags, ","))
}

func (c *Client) RemoveTags(hashes []string, tags string) error {
	return c.RemoveTagsCtx(context.Background(), hashes, tags)
}

func (c *Client) RemoveTagsCtx(ctx context.Context, hashes []string, tags string) error {
	current, err := c.labelMap(ctx, hashes)
	if err != nil {
		return err
	}
	remove := make(map[string]struct{})
	for _, tag := range splitCSV(tags) {
		remove[tag] = struct{}{}
	}
	for _, hash := range hashes {
		existing := current[strings.ToUpper(hash)]
		labels := make([]string, 0, len(existing))
		for _, label := range existing {
			if _, ok := remove[label]; !ok {
				labels = append(labels, label)
			}
		}
		if err := c.setLabels(ctx, []string{hash}, labels); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) labelMap(ctx context.Context, hashes []string) (map[string][]string, error) {
	torrents, err := c.fetchTRTorrents(ctx, hashes, "hash_string", "labels")
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string, len(torrents))
	for _, torrent := range torrents {
		result[strings.ToUpper(torrent.HashString)] = append([]string(nil), torrent.Labels...)
	}
	return result, nil
}

func (c *Client) setLabels(ctx context.Context, hashes []string, labels []string) error {
	return c.rpcCall(ctx, "torrent_set", map[string]interface{}{
		"ids":    hashes,
		"labels": removeDuplicateStrings(labels),
	}, nil)
}

func (c *Client) RemoveTrackers(hash string, urls string) error {
	return c.RemoveTrackersCtx(context.Background(), hash, urls)
}

func (c *Client) RemoveTrackersCtx(ctx context.Context, hash string, urls string) error {
	return c.updateTrackerList(ctx, hash, func(existing []string) []string {
		remove := make(map[string]struct{})
		for _, u := range strings.Fields(urls) {
			remove[u] = struct{}{}
		}
		out := make([]string, 0, len(existing))
		for _, u := range existing {
			if _, ok := remove[u]; !ok {
				out = append(out, u)
			}
		}
		return out
	})
}

func (c *Client) EditTracker(hash string, old, new string) error {
	return c.EditTrackerCtx(context.Background(), hash, old, new)
}

func (c *Client) EditTrackerCtx(ctx context.Context, hash string, old, new string) error {
	return c.updateTrackerList(ctx, hash, func(existing []string) []string {
		for i, u := range existing {
			if u == old {
				existing[i] = new
			}
		}
		return existing
	})
}

func (c *Client) AddTrackers(hash string, urls string) error {
	return c.AddTrackersCtx(context.Background(), hash, urls)
}

func (c *Client) AddTrackersCtx(ctx context.Context, hash string, urls string) error {
	return c.updateTrackerList(ctx, hash, func(existing []string) []string {
		for _, u := range strings.Fields(urls) {
			if strings.TrimSpace(u) != "" {
				existing = append(existing, strings.TrimSpace(u))
			}
		}
		return removeDuplicateStrings(existing)
	})
}

func (c *Client) updateTrackerList(ctx context.Context, hash string, update func([]string) []string) error {
	torrents, err := c.fetchTRTorrents(ctx, []string{hash}, "hash_string", "tracker_list", "trackers")
	if err != nil {
		return err
	}
	if len(torrents) == 0 {
		return ErrTorrentNotFound
	}
	existing := trackerURLs(torrents[0])
	next := update(existing)
	return c.rpcCall(ctx, "torrent_set", map[string]interface{}{
		"ids":          []string{hash},
		"tracker_list": strings.Join(next, "\n"),
	}, nil)
}

func trackerURLs(torrent trTorrent) []string {
	if strings.TrimSpace(torrent.TrackerList) != "" {
		return strings.Fields(torrent.TrackerList)
	}
	urls := make([]string, 0, len(torrent.Trackers))
	for _, tracker := range torrent.Trackers {
		if tracker.Announce != "" {
			urls = append(urls, tracker.Announce)
		}
	}
	return urls
}

func (c *Client) SetPreferencesQueueingEnabled(enabled bool) error {
	return c.SetPreferences(map[string]interface{}{"queueing_enabled": enabled})
}

func (c *Client) SetPreferencesMaxActiveDownloads(max int) error {
	return c.SetPreferences(map[string]interface{}{"max_active_downloads": max})
}

func (c *Client) SetPreferencesMaxActiveTorrents(maxActive int) error {
	half := maxActive / 2
	return c.SetPreferences(map[string]interface{}{"max_active_downloads": half, "max_active_uploads": maxActive - half})
}

func (c *Client) SetPreferencesMaxActiveUploads(maxUploads int) error {
	return c.SetPreferences(map[string]interface{}{"max_active_uploads": maxUploads})
}

func (c *Client) SetPreferencesSubcategoriesEnabled(enabled bool) error {
	_ = enabled
	return nil
}

func (c *Client) GetMonitoredFolders() (MonitoredFolders, error) {
	return c.GetMonitoredFoldersCtx(context.Background())
}

func (c *Client) GetMonitoredFoldersCtx(ctx context.Context) (MonitoredFolders, error) {
	_ = ctx
	return MonitoredFolders{}, nil
}

func (c *Client) SetMonitoredFolders(scanDirs MonitoredFolders) error {
	return c.SetMonitoredFoldersCtx(context.Background(), scanDirs)
}

func (c *Client) SetMonitoredFoldersCtx(ctx context.Context, scanDirs MonitoredFolders) error {
	_ = ctx
	_ = scanDirs
	return unsupported("watched folders")
}

func (c *Client) SetRSSAutoDownloadingEnabled(enabled bool) error {
	return c.SetRSSAutoDownloadingEnabledCtx(context.Background(), enabled)
}

func (c *Client) SetRSSAutoDownloadingEnabledCtx(ctx context.Context, enabled bool) error {
	_ = ctx
	_ = enabled
	return unsupported("Transmission RSS")
}

func (c *Client) SetRSSProcessingEnabled(enabled bool) error {
	return c.SetRSSProcessingEnabledCtx(context.Background(), enabled)
}

func (c *Client) SetRSSProcessingEnabledCtx(ctx context.Context, enabled bool) error {
	_ = ctx
	_ = enabled
	return unsupported("Transmission RSS")
}

func (c *Client) SetMaxPriority(hashes []string) error {
	return c.SetMaxPriorityCtx(context.Background(), hashes)
}

func (c *Client) SetMaxPriorityCtx(ctx context.Context, hashes []string) error {
	return c.rpcCall(ctx, "queue_move_top", idsParam(hashes), nil)
}

func (c *Client) SetMinPriority(hashes []string) error {
	return c.SetMinPriorityCtx(context.Background(), hashes)
}

func (c *Client) SetMinPriorityCtx(ctx context.Context, hashes []string) error {
	return c.rpcCall(ctx, "queue_move_bottom", idsParam(hashes), nil)
}

func (c *Client) DecreasePriority(hashes []string) error {
	return c.DecreasePriorityCtx(context.Background(), hashes)
}

func (c *Client) DecreasePriorityCtx(ctx context.Context, hashes []string) error {
	return c.rpcCall(ctx, "queue_move_down", idsParam(hashes), nil)
}

func (c *Client) IncreasePriority(hashes []string) error {
	return c.IncreasePriorityCtx(context.Background(), hashes)
}

func (c *Client) IncreasePriorityCtx(ctx context.Context, hashes []string) error {
	return c.rpcCall(ctx, "queue_move_up", idsParam(hashes), nil)
}

func (c *Client) ToggleFirstLastPiecePrio(hashes []string) error {
	return c.ToggleFirstLastPiecePrioCtx(context.Background(), hashes)
}

func (c *Client) ToggleFirstLastPiecePrioCtx(ctx context.Context, hashes []string) error {
	_ = ctx
	_ = hashes
	return unsupported("first/last piece priority")
}

func (c *Client) ToggleAlternativeSpeedLimits() error {
	return c.ToggleAlternativeSpeedLimitsCtx(context.Background())
}

func (c *Client) ToggleAlternativeSpeedLimitsCtx(ctx context.Context) error {
	enabled, err := c.GetAlternativeSpeedLimitsModeCtx(ctx)
	if err != nil {
		return err
	}
	return c.rpcCall(ctx, "session_set", map[string]interface{}{"alt_speed_enabled": !enabled}, nil)
}

func (c *Client) GetAlternativeSpeedLimitsMode() (bool, error) {
	return c.GetAlternativeSpeedLimitsModeCtx(context.Background())
}

func (c *Client) GetAlternativeSpeedLimitsModeCtx(ctx context.Context) (bool, error) {
	session, err := c.getSession(ctx, "alt_speed_enabled")
	return session.AltSpeedEnabled, err
}

func (c *Client) SetGlobalDownloadLimit(limit int64) error {
	return c.SetGlobalDownloadLimitCtx(context.Background(), limit)
}

func (c *Client) SetGlobalDownloadLimitCtx(ctx context.Context, limit int64) error {
	return c.rpcCall(ctx, "session_set", map[string]interface{}{
		"speed_limit_down":         bytesToKB(limit),
		"speed_limit_down_enabled": limit > 0,
	}, nil)
}

func (c *Client) GetGlobalDownloadLimit() (int64, error) {
	return c.GetGlobalDownloadLimitCtx(context.Background())
}

func (c *Client) GetGlobalDownloadLimitCtx(ctx context.Context) (int64, error) {
	session, err := c.getSession(ctx, "speed_limit_down", "speed_limit_down_enabled")
	return kbToBytes(session.SpeedLimitDown, session.SpeedLimitDownEnabled), err
}

func (c *Client) SetGlobalUploadLimit(limit int64) error {
	return c.SetGlobalUploadLimitCtx(context.Background(), limit)
}

func (c *Client) SetGlobalUploadLimitCtx(ctx context.Context, limit int64) error {
	return c.rpcCall(ctx, "session_set", map[string]interface{}{
		"speed_limit_up":         bytesToKB(limit),
		"speed_limit_up_enabled": limit > 0,
	}, nil)
}

func (c *Client) GetGlobalUploadLimit() (int64, error) {
	return c.GetGlobalUploadLimitCtx(context.Background())
}

func (c *Client) GetGlobalUploadLimitCtx(ctx context.Context) (int64, error) {
	session, err := c.getSession(ctx, "speed_limit_up", "speed_limit_up_enabled")
	return kbToBytes(session.SpeedLimitUp, session.SpeedLimitUpEnabled), err
}

func (c *Client) GetTorrentUploadLimit(hashes []string) (map[string]int64, error) {
	return c.GetTorrentUploadLimitCtx(context.Background(), hashes)
}

func (c *Client) GetTorrentUploadLimitCtx(ctx context.Context, hashes []string) (map[string]int64, error) {
	torrents, err := c.fetchTRTorrents(ctx, hashes, "hash_string", "upload_limit", "upload_limited")
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(torrents))
	for _, torrent := range torrents {
		result[strings.ToUpper(torrent.HashString)] = kbToBytes(torrent.UploadLimit, torrent.UploadLimited)
	}
	return result, nil
}

func (c *Client) GetTorrentDownloadLimit(hashes []string) (map[string]int64, error) {
	return c.GetTorrentDownloadLimitCtx(context.Background(), hashes)
}

func (c *Client) GetTorrentDownloadLimitCtx(ctx context.Context, hashes []string) (map[string]int64, error) {
	torrents, err := c.fetchTRTorrents(ctx, hashes, "hash_string", "download_limit", "download_limited")
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(torrents))
	for _, torrent := range torrents {
		result[strings.ToUpper(torrent.HashString)] = kbToBytes(torrent.DownloadLimit, torrent.DownloadLimited)
	}
	return result, nil
}

func (c *Client) SetTorrentDownloadLimit(hashes []string, limit int64) error {
	return c.SetTorrentDownloadLimitCtx(context.Background(), hashes, limit)
}

func (c *Client) SetTorrentDownloadLimitCtx(ctx context.Context, hashes []string, limit int64) error {
	return c.rpcCall(ctx, "torrent_set", map[string]interface{}{
		"ids":              hashes,
		"download_limit":   bytesToKB(limit),
		"download_limited": limit > 0,
	}, nil)
}

func (c *Client) ToggleTorrentSequentialDownload(hashes []string) error {
	return c.ToggleTorrentSequentialDownloadCtx(context.Background(), hashes)
}

func (c *Client) ToggleTorrentSequentialDownloadCtx(ctx context.Context, hashes []string) error {
	torrents, err := c.fetchTRTorrents(ctx, hashes, "hash_string", "sequential_download")
	if err != nil {
		return err
	}
	for _, torrent := range torrents {
		if err := c.rpcCall(ctx, "torrent_set", map[string]interface{}{
			"ids":                 []string{torrent.HashString},
			"sequential_download": !torrent.SequentialDownload,
		}, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) SetTorrentSuperSeeding(hashes []string, on bool) error {
	return c.SetTorrentSuperSeedingCtx(context.Background(), hashes, on)
}

func (c *Client) SetTorrentSuperSeedingCtx(ctx context.Context, hashes []string, on bool) error {
	_ = ctx
	_ = hashes
	_ = on
	return unsupported("super seeding")
}

func (c *Client) SetTorrentShareLimit(hashes []string, opts ShareLimitOptions) error {
	return c.SetTorrentShareLimitCtx(context.Background(), hashes, opts)
}

func (c *Client) SetTorrentShareLimitCtx(ctx context.Context, hashes []string, opts ShareLimitOptions) error {
	params := map[string]interface{}{"ids": hashes}
	if opts.RatioLimit >= 0 {
		params["seed_ratio_limit"] = opts.RatioLimit
		params["seed_ratio_mode"] = 1
	}
	if opts.SeedingTimeLimit >= 0 {
		params["seed_idle_limit"] = opts.SeedingTimeLimit
		params["seed_idle_mode"] = 1
	}
	return c.rpcCall(ctx, "torrent_set", params, nil)
}

func (c *Client) SetTorrentUploadLimit(hashes []string, limit int64) error {
	return c.SetTorrentUploadLimitCtx(context.Background(), hashes, limit)
}

func (c *Client) SetTorrentUploadLimitCtx(ctx context.Context, hashes []string, limit int64) error {
	return c.rpcCall(ctx, "torrent_set", map[string]interface{}{
		"ids":            hashes,
		"upload_limit":   bytesToKB(limit),
		"upload_limited": limit > 0,
	}, nil)
}

func (c *Client) GetAppVersion() (string, error) {
	return c.GetAppVersionCtx(context.Background())
}

func (c *Client) GetAppVersionCtx(ctx context.Context) (string, error) {
	session, err := c.getSession(ctx, "version")
	return session.Version, err
}

func (c *Client) GetAppCookies() ([]Cookie, error) {
	return c.GetAppCookiesCtx(context.Background())
}

func (c *Client) GetAppCookiesCtx(ctx context.Context) ([]Cookie, error) {
	_ = ctx
	return nil, unsupported("app cookies")
}

func (c *Client) SetAppCookies(cookies []Cookie) error {
	return c.SetAppCookiesCtx(context.Background(), cookies)
}

func (c *Client) SetAppCookiesCtx(ctx context.Context, cookies []Cookie) error {
	_ = ctx
	_ = cookies
	return unsupported("app cookies")
}

func (c *Client) GetTorrentPieceStates(hash string) ([]PieceState, error) {
	return c.GetTorrentPieceStatesCtx(context.Background(), hash)
}

func (c *Client) GetTorrentPieceStatesCtx(ctx context.Context, hash string) ([]PieceState, error) {
	torrents, err := c.fetchTRTorrents(ctx, []string{hash}, "hash_string", "pieces", "piece_count")
	if err != nil {
		return nil, err
	}
	if len(torrents) == 0 {
		return nil, ErrTorrentNotFound
	}
	pieces, err := base64.StdEncoding.DecodeString(torrents[0].Pieces)
	if err != nil {
		return nil, err
	}
	states := make([]PieceState, torrents[0].PieceCount)
	for i := 0; i < torrents[0].PieceCount; i++ {
		byteIndex := i / 8
		bitIndex := 7 - (i % 8)
		if byteIndex < len(pieces) && pieces[byteIndex]&(1<<bitIndex) != 0 {
			states[i] = PieceStateAlreadyDownloaded
		}
	}
	return states, nil
}

func (c *Client) GetTorrentPieceHashes(hash string) ([]string, error) {
	return c.GetTorrentPieceHashesCtx(context.Background(), hash)
}

func (c *Client) GetTorrentPieceHashesCtx(ctx context.Context, hash string) ([]string, error) {
	_ = ctx
	_ = hash
	return nil, unsupported("piece hashes")
}

func (c *Client) AddPeersForTorrents(hashes, peers []string) error {
	return c.AddPeersForTorrentsCtx(context.Background(), hashes, peers)
}

func (c *Client) AddPeersForTorrentsCtx(ctx context.Context, hashes, peers []string) error {
	_ = ctx
	_ = hashes
	_ = peers
	return unsupported("manual peer add")
}

func (c *Client) GetWebAPIVersion() (string, error) {
	return c.GetWebAPIVersionCtx(context.Background())
}

func (c *Client) GetWebAPIVersionCtx(ctx context.Context) (string, error) {
	session, err := c.getSession(ctx, "rpc_version_semver", "rpc_version")
	if err != nil {
		return "", err
	}
	if session.RPCVersionSemver != "" {
		return session.RPCVersionSemver, nil
	}
	if session.RPCVersion > 0 {
		return fmt.Sprintf("%d.0.0", session.RPCVersion), nil
	}
	return "6.0.0", nil
}

func (c *Client) GetLogs() ([]Log, error) {
	return c.GetLogsCtx(context.Background())
}

func (c *Client) GetLogsCtx(ctx context.Context) ([]Log, error) {
	_ = ctx
	return nil, unsupported("Transmission log retrieval")
}

func (c *Client) GetPeerLogs() ([]PeerLog, error) {
	return c.GetPeerLogsCtx(context.Background())
}

func (c *Client) GetPeerLogsCtx(ctx context.Context) ([]PeerLog, error) {
	_ = ctx
	return nil, unsupported("Transmission peer log retrieval")
}

func (c *Client) GetFreeSpaceOnDisk() (int64, error) {
	return c.GetFreeSpaceOnDiskCtx(context.Background())
}

func (c *Client) GetFreeSpaceOnDiskCtx(ctx context.Context) (int64, error) {
	session, err := c.getSession(ctx, "download_dir")
	if err != nil {
		return 0, err
	}
	var result struct {
		Path      string `json:"path"`
		SizeBytes int64  `json:"size_bytes"`
		TotalSize int64  `json:"total_size"`
	}
	if err := c.rpcCall(ctx, "free_space", map[string]interface{}{"path": session.DownloadDir}, &result); err != nil {
		return 0, nil
	}
	return result.SizeBytes, nil
}

func (c *Client) RequiresMinVersion(minVersion *semver.Version) (bool, error) {
	versionString, err := c.GetWebAPIVersion()
	if err != nil {
		return false, err
	}
	version, err := semver.NewVersion(versionString)
	if err != nil {
		return false, err
	}
	if version.LessThan(minVersion) {
		return false, ErrUnsupportedVersion
	}
	return true, nil
}

const (
	ReannounceMaxAttempts = 50
	ReannounceInterval    = 7
)

type ReannounceOptions struct {
	Interval        int
	MaxAttempts     int
	DeleteOnFailure bool
}

func (c *Client) ReannounceTorrentWithRetry(ctx context.Context, hash string, opts *ReannounceOptions) error {
	interval := ReannounceInterval
	maxAttempts := ReannounceMaxAttempts
	deleteOnFailure := false
	if opts != nil {
		if opts.Interval > 0 {
			interval = opts.Interval
		}
		if opts.MaxAttempts > 0 {
			maxAttempts = opts.MaxAttempts
		}
		deleteOnFailure = opts.DeleteOnFailure
	}
	for i := 0; i < maxAttempts; i++ {
		if err := c.ReAnnounceTorrentsCtx(ctx, []string{hash}); err != nil {
			return err
		}
		time.Sleep(time.Duration(interval) * time.Second)
		trackers, err := c.GetTorrentTrackersCtx(ctx, hash)
		if err == nil && isTrackerStatusOK(trackers) {
			return nil
		}
	}
	if deleteOnFailure {
		_ = c.DeleteTorrentsCtx(ctx, []string{hash}, false)
		return ErrReannounceTookTooLong
	}
	return nil
}

func isTrackerStatusOK(trackers []TorrentTracker) bool {
	for _, tracker := range trackers {
		if tracker.Status == TrackerStatusOK && !isUnregistered(tracker.Message) {
			return true
		}
	}
	return false
}

func isUnregistered(msg string) bool {
	msg = strings.ToLower(msg)
	for _, word := range []string{"unregistered", "not registered", "not found", "not exist"} {
		if strings.Contains(msg, word) {
			return true
		}
	}
	return false
}

func (c *Client) GetTorrentsWebSeeds(hash string) ([]WebSeed, error) {
	return c.GetTorrentsWebSeedsCtx(context.Background(), hash)
}

func (c *Client) GetTorrentsWebSeedsCtx(ctx context.Context, hash string) ([]WebSeed, error) {
	torrents, err := c.fetchTRTorrents(ctx, []string{hash}, "hash_string", "webseeds", "webseeds_ex")
	if err != nil {
		return nil, err
	}
	if len(torrents) == 0 {
		return nil, ErrTorrentNotFound
	}
	var seeds []WebSeed
	for _, seed := range torrents[0].WebSeedsEx {
		seeds = append(seeds, WebSeed{URL: seed.URL})
	}
	for _, seed := range torrents[0].WebSeeds {
		seeds = append(seeds, WebSeed{URL: seed})
	}
	return seeds, nil
}

func (c *Client) GetTorrentPeers(hash string, rid int64) (*TorrentPeersResponse, error) {
	return c.GetTorrentPeersCtx(context.Background(), hash, rid)
}

func (c *Client) GetTorrentPeersCtx(ctx context.Context, hash string, rid int64) (*TorrentPeersResponse, error) {
	torrents, err := c.fetchTRTorrents(ctx, []string{hash}, "hash_string", "peers")
	if err != nil {
		return nil, err
	}
	if len(torrents) == 0 {
		return nil, ErrTorrentNotFound
	}
	peers := make(map[string]TorrentPeer, len(torrents[0].Peers))
	for _, peer := range torrents[0].Peers {
		key := fmt.Sprintf("%s:%d", peer.Address, peer.Port)
		flags := peer.FlagStr
		if flags == "" {
			flags = transmissionPeerFlags(peer)
		}
		peers[key] = TorrentPeer{
			IP:           peer.Address,
			Connection:   "BT",
			Flags:        flags,
			Client:       peer.ClientName,
			PeerIDClient: peer.PeerID,
			Port:         peer.Port,
			Progress:     peer.Progress,
			DownSpeed:    peer.RateToClient,
			UpSpeed:      peer.RateToPeer,
			Downloaded:   peer.BytesToClient,
			Uploaded:     peer.BytesToPeer,
			Relevance:    1,
		}
	}
	return &TorrentPeersResponse{
		Peers:      peers,
		Rid:        time.Now().UnixNano(),
		FullUpdate: true,
		ShowFlags:  true,
	}, nil
}

func transmissionPeerFlags(peer trPeer) string {
	var flags strings.Builder
	if peer.IsEncrypted {
		flags.WriteByte('E')
	}
	if peer.IsIncoming {
		flags.WriteByte('I')
	}
	if peer.IsUTP {
		flags.WriteByte('U')
	}
	if peer.IsDownloadingFrom {
		flags.WriteByte('D')
	}
	if peer.IsUploadingTo {
		flags.WriteByte('U')
	}
	return flags.String()
}

func (c *Client) CreateTorrent(params TorrentCreationParams) (*TorrentCreationTaskResponse, error) {
	return c.CreateTorrentCtx(context.Background(), params)
}

func (c *Client) CreateTorrentCtx(ctx context.Context, params TorrentCreationParams) (*TorrentCreationTaskResponse, error) {
	_ = ctx
	_ = params
	return nil, unsupported("torrent creation")
}

func (c *Client) GetTorrentCreationStatus(taskID string) ([]TorrentCreationTask, error) {
	return c.GetTorrentCreationStatusCtx(context.Background(), taskID)
}

func (c *Client) GetTorrentCreationStatusCtx(ctx context.Context, taskID string) ([]TorrentCreationTask, error) {
	_ = ctx
	_ = taskID
	return nil, unsupported("torrent creation")
}

func (c *Client) GetTorrentFile(taskID string) ([]byte, error) {
	return c.GetTorrentFileCtx(context.Background(), taskID)
}

func (c *Client) GetTorrentFileCtx(ctx context.Context, taskID string) ([]byte, error) {
	_ = ctx
	_ = taskID
	return nil, unsupported("torrent creation")
}

func (c *Client) DeleteTorrentCreationTask(taskID string) error {
	return c.DeleteTorrentCreationTaskCtx(context.Background(), taskID)
}

func (c *Client) DeleteTorrentCreationTaskCtx(ctx context.Context, taskID string) error {
	_ = ctx
	_ = taskID
	return unsupported("torrent creation")
}

func (c *Client) postTransmissionMultipart(ctx context.Context, fields map[string]string, fileField string, fileName string, reader io.Reader) (*http.Response, error) {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}
	if reader != nil {
		part, err := writer.CreateFormFile(fileField, fileName)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(part, reader); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.rpcURL(), &requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	user, pass := c.rpcCredentials()
	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}
	return c.http.Do(req)
}
