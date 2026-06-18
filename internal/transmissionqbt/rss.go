package qbittorrent

import (
	"context"
	"encoding/json"
)

// RSSItems represents the hierarchical response from qBittorrent's RSS API.
// Transmission does not provide an equivalent built-in RSS API; the methods
// below return ErrUnsupportedVersion so callers can disable this feature.
type RSSItems map[string]json.RawMessage

type RSSFeed struct {
	UID             string       `json:"uid"`
	URL             string       `json:"url"`
	RefreshInterval int64        `json:"refreshInterval,omitempty"`
	Title           string       `json:"title,omitempty"`
	LastBuildDate   string       `json:"lastBuildDate,omitempty"`
	IsLoading       bool         `json:"isLoading,omitempty"`
	HasError        bool         `json:"hasError,omitempty"`
	Articles        []RSSArticle `json:"articles,omitempty"`
}

type RSSArticle struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	Title       string `json:"title"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	TorrentURL  string `json:"torrentURL,omitempty"`
	Link        string `json:"link,omitempty"`
	IsRead      bool   `json:"isRead"`
}

type RSSAutoDownloadRule struct {
	Enabled                   bool                  `json:"enabled"`
	Priority                  int                   `json:"priority"`
	UseRegex                  bool                  `json:"useRegex"`
	MustContain               string                `json:"mustContain"`
	MustNotContain            string                `json:"mustNotContain"`
	EpisodeFilter             string                `json:"episodeFilter,omitempty"`
	AffectedFeeds             []string              `json:"affectedFeeds"`
	LastMatch                 string                `json:"lastMatch,omitempty"`
	IgnoreDays                int                   `json:"ignoreDays"`
	SmartFilter               bool                  `json:"smartFilter"`
	PreviouslyMatchedEpisodes []string              `json:"previouslyMatchedEpisodes,omitempty"`
	TorrentParams             *RSSRuleTorrentParams `json:"torrentParams,omitempty"`
	AddPaused                 *bool                 `json:"addPaused,omitempty"`
	SavePath                  string                `json:"savePath,omitempty"`
	AssignedCategory          string                `json:"assignedCategory,omitempty"`
	TorrentContentLayout      string                `json:"torrentContentLayout,omitempty"`
}

type RSSRuleTorrentParams struct {
	Category                 string   `json:"category,omitempty"`
	Tags                     []string `json:"tags,omitempty"`
	SavePath                 string   `json:"save_path,omitempty"`
	DownloadPath             string   `json:"download_path,omitempty"`
	ContentLayout            string   `json:"content_layout,omitempty"`
	OperatingMode            string   `json:"operating_mode,omitempty"`
	SkipChecking             bool     `json:"skip_checking,omitempty"`
	UploadLimit              int      `json:"upload_limit,omitempty"`
	DownloadLimit            int      `json:"download_limit,omitempty"`
	SeedingTimeLimit         int      `json:"seeding_time_limit,omitempty"`
	InactiveSeedingTimeLimit int      `json:"inactive_seeding_time_limit,omitempty"`
	ShareLimitAction         string   `json:"share_limit_action,omitempty"`
	RatioLimit               float64  `json:"ratio_limit,omitempty"`
	Stopped                  *bool    `json:"stopped,omitempty"`
	StopCondition            string   `json:"stop_condition,omitempty"`
	UseAutoTMM               *bool    `json:"use_auto_tmm,omitempty"`
	UseDownloadPath          *bool    `json:"use_download_path,omitempty"`
	AddToQueueTop            *bool    `json:"add_to_top_of_queue,omitempty"`
}

type RSSRules map[string]RSSAutoDownloadRule

type RSSMatchingArticles map[string][]string

func (items RSSItems) ParseFeeds() ([]RSSFeed, error) {
	var feeds []RSSFeed
	for _, raw := range items {
		var feed RSSFeed
		if err := json.Unmarshal(raw, &feed); err == nil && feed.URL != "" {
			feeds = append(feeds, feed)
			continue
		}
		var nested RSSItems
		if err := json.Unmarshal(raw, &nested); err == nil {
			nestedFeeds, _ := nested.ParseFeeds()
			feeds = append(feeds, nestedFeeds...)
		}
	}
	return feeds, nil
}

func IsFeed(raw json.RawMessage) bool {
	var feed RSSFeed
	if err := json.Unmarshal(raw, &feed); err == nil {
		return feed.URL != ""
	}
	return false
}

func (c *Client) GetRSSItems(withData bool) (RSSItems, error) {
	return c.GetRSSItemsCtx(context.Background(), withData)
}

func (c *Client) GetRSSItemsCtx(ctx context.Context, withData bool) (RSSItems, error) {
	_ = ctx
	_ = withData
	return nil, unsupported("Transmission RSS")
}

func (c *Client) AddRSSFolder(path string) error {
	return c.AddRSSFolderCtx(context.Background(), path)
}

func (c *Client) AddRSSFolderCtx(ctx context.Context, path string) error {
	_ = ctx
	_ = path
	return unsupported("Transmission RSS")
}

func (c *Client) AddRSSFeed(url, path string) error {
	return c.AddRSSFeedCtx(context.Background(), url, path)
}

func (c *Client) AddRSSFeedCtx(ctx context.Context, url, path string) error {
	_ = ctx
	_ = url
	_ = path
	return unsupported("Transmission RSS")
}

func (c *Client) SetRSSFeedURL(path, url string) error {
	return c.SetRSSFeedURLCtx(context.Background(), path, url)
}

func (c *Client) SetRSSFeedURLCtx(ctx context.Context, path, url string) error {
	_ = ctx
	_ = path
	_ = url
	return unsupported("Transmission RSS")
}

func (c *Client) RemoveRSSItem(path string) error {
	return c.RemoveRSSItemCtx(context.Background(), path)
}

func (c *Client) RemoveRSSItemCtx(ctx context.Context, path string) error {
	_ = ctx
	_ = path
	return unsupported("Transmission RSS")
}

func (c *Client) MoveRSSItem(itemPath, destPath string) error {
	return c.MoveRSSItemCtx(context.Background(), itemPath, destPath)
}

func (c *Client) MoveRSSItemCtx(ctx context.Context, itemPath, destPath string) error {
	_ = ctx
	_ = itemPath
	_ = destPath
	return unsupported("Transmission RSS")
}

func (c *Client) RefreshRSSItem(itemPath string) error {
	return c.RefreshRSSItemCtx(context.Background(), itemPath)
}

func (c *Client) RefreshRSSItemCtx(ctx context.Context, itemPath string) error {
	_ = ctx
	_ = itemPath
	return unsupported("Transmission RSS")
}

func (c *Client) MarkRSSItemAsRead(itemPath string, articleID string) error {
	return c.MarkRSSItemAsReadCtx(context.Background(), itemPath, articleID)
}

func (c *Client) MarkRSSItemAsReadCtx(ctx context.Context, itemPath string, articleID string) error {
	_ = ctx
	_ = itemPath
	_ = articleID
	return unsupported("Transmission RSS")
}

func (c *Client) GetRSSRules() (RSSRules, error) {
	return c.GetRSSRulesCtx(context.Background())
}

func (c *Client) GetRSSRulesCtx(ctx context.Context) (RSSRules, error) {
	_ = ctx
	return nil, unsupported("Transmission RSS")
}

func (c *Client) SetRSSRule(ruleName string, rule RSSAutoDownloadRule) error {
	return c.SetRSSRuleCtx(context.Background(), ruleName, rule)
}

func (c *Client) SetRSSRuleCtx(ctx context.Context, ruleName string, rule RSSAutoDownloadRule) error {
	_ = ctx
	_ = ruleName
	_ = rule
	return unsupported("Transmission RSS")
}

func (c *Client) RenameRSSRule(ruleName, newRuleName string) error {
	return c.RenameRSSRuleCtx(context.Background(), ruleName, newRuleName)
}

func (c *Client) RenameRSSRuleCtx(ctx context.Context, ruleName, newRuleName string) error {
	_ = ctx
	_ = ruleName
	_ = newRuleName
	return unsupported("Transmission RSS")
}

func (c *Client) RemoveRSSRule(ruleName string) error {
	return c.RemoveRSSRuleCtx(context.Background(), ruleName)
}

func (c *Client) RemoveRSSRuleCtx(ctx context.Context, ruleName string) error {
	_ = ctx
	_ = ruleName
	return unsupported("Transmission RSS")
}

func (c *Client) GetRSSMatchingArticles(ruleName string) (RSSMatchingArticles, error) {
	return c.GetRSSMatchingArticlesCtx(context.Background(), ruleName)
}

func (c *Client) GetRSSMatchingArticlesCtx(ctx context.Context, ruleName string) (RSSMatchingArticles, error) {
	_ = ctx
	_ = ruleName
	return nil, unsupported("Transmission RSS")
}
