package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"yoho-music/internal/music"
)

const maxMarketCatalog = 512 << 10

// MarketPlugin describes a download advertised by the market. Installation
// checks its identity and build against the signed package.
type MarketPlugin struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Build       uint64 `json:"build"`
	PackageURL  string `json:"package"`
	Description string `json:"description,omitempty"`
}

func (p MarketPlugin) SourceID() string { return NSGoSourcePrefix + p.ID }

func checkedMarketURL(u *url.URL, allowLocalHTTP bool) bool {
	if u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme != "http" || !allowLocalHTTP {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

var marketClient = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 || !checkedMarketURL(req.URL, via[0].URL.Scheme == "http") {
			return errors.New("invalid plugin market redirect")
		}
		return nil
	},
}

func fetchMarketFile(ctx context.Context, address string, limit int64) ([]byte, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil || !checkedMarketURL(req.URL, true) {
		return nil, nil, errors.New("invalid plugin market URL")
	}
	req.Header.Set("User-Agent", music.Agent)
	res, err := marketClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("plugin market returned %s", res.Status)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if int64(len(data)) > limit {
		return nil, nil, errors.New("plugin market file exceeds the download limit")
	}
	return data, res.Request.URL, err
}

// FetchMarket reads metadata only. Package bytes are fetched on user request.
func FetchMarket(ctx context.Context, address string) ([]MarketPlugin, error) {
	data, base, err := fetchMarketFile(ctx, address, maxMarketCatalog)
	if err != nil {
		return nil, err
	}
	var catalog struct {
		Plugins []MarketPlugin `json:"plugins"`
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("read plugin market catalog: %w", err)
	}
	if catalog.Plugins == nil {
		return nil, errors.New("missing plugin market list")
	}
	seen := make(map[string]bool, len(catalog.Plugins))
	for i := range catalog.Plugins {
		p := &catalog.Plugins[i]
		u, err := base.Parse(p.PackageURL)
		if !pluginName.MatchString(p.ID) || seen[p.ID] ||
			p.Name == "" || p.Version == "" || p.Build == 0 || p.PackageURL == "" ||
			err != nil || !checkedMarketURL(u, base.Scheme == "http") {
			return nil, fmt.Errorf("invalid plugin market entry %q", p.ID)
		}
		seen[p.ID] = true
		p.PackageURL = u.String()
	}
	return catalog.Plugins, nil
}

// DownloadMarketPlugin prepares a download for the same verification and
// installation flow used for local package imports.
func (s *NSGoStore) DownloadMarketPlugin(ctx context.Context, entry MarketPlugin) (*NSGoPackage, error) {
	data, _, err := fetchMarketFile(ctx, entry.PackageURL, maxNSGoPackage)
	if err != nil {
		return nil, err
	}
	p, err := s.prepareInstall(data)
	if err != nil {
		return nil, err
	}
	app := p.App()
	if app.ID != entry.ID || app.Version != entry.Version || app.Build != entry.Build {
		return nil, errors.New("downloaded plugin does not match its market listing")
	}
	return p, nil
}
