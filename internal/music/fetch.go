package music

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Agent is who the app says it is when fetching what it plays and shows.
const Agent = "YohoMusic/0.1"

// fetchClient gets covers: they're small, and a slow one shouldn't hold a
// slot for long.
var fetchClient = &http.Client{Timeout: 30 * time.Second}

// maxFetch is the most a cover may take.
const maxFetch = 64 << 20

// ImageHeaders, when set, answers the headers to fetch the picture at u with,
// such as the cookie of a source that keeps its covers behind its session.
// It may answer nil.
var ImageHeaders func(u *url.URL) map[string]string

// SizedImage returns the URL to fetch the picture at u with to show it px
// pixels across: u with its size filled in, where it says it goes with
// {size}, else u itself.
func SizedImage(u string, px int) string {
	if px <= 0 {
		px = placeholderSize
	}
	if !strings.Contains(u, "size") {
		return u
	}
	return sizePlaceholder.ReplaceAllLiteralString(u, strconv.Itoa(px))
}

// placeholderSize is the size a picture whose URL says where its size goes
// is fetched at where nothing asks for one: the most the app shows one at.
const placeholderSize = 600

// sizePlaceholder is where a picture's URL says its size goes, in the URL
// as a source gives it or escaped, as a URL parsed and written again has it.
var sizePlaceholder = regexp.MustCompile(`\{size\}|%7[Bb]size%7[Dd]`)

// Fetch returns what's at url, of the web or a file://, such as a cover.
func Fetch(u string) ([]byte, error) {
	return FetchImage(context.Background(), u, ImageHeaders)
}

// FetchImage fetches artwork with headers supplied by its own source.
func FetchImage(ctx context.Context, u string, lookup func(*url.URL) map[string]string) ([]byte, error) {
	return FetchMediaImage(ctx, u, lookup, nil)
}

// FetchMediaImage applies the source policy to both the initial URL and redirects.
func FetchMediaImage(ctx context.Context, u string, lookup func(*url.URL) map[string]string, check func(*url.URL) error) ([]byte, error) {
	if path, ok := PathOf(u); ok && check == nil {
		return os.ReadFile(path)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", Agent)
	var headers map[string]string
	if lookup != nil {
		headers = lookup(req.URL)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := RequestMedia(fetchClient, req, headers, check)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(res.Status)
	}
	return io.ReadAll(io.LimitReader(res.Body, maxFetch))
}

// RequestMedia fetches media while checking the source's URL policy on every hop.
func RequestMedia(client *http.Client, req *http.Request, headers map[string]string, check func(*url.URL) error) (*http.Response, error) {
	if check == nil {
		return HeadersStay(client, req.URL.Hostname(), headers).Do(req)
	}
	if err := check(req.URL); err != nil {
		return nil, err
	}
	c := *client
	c.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		if err := check(next.URL); err != nil {
			return err
		}
		for _, previous := range via {
			if next.URL.Scheme != previous.URL.Scheme || !strings.EqualFold(next.URL.Host, previous.URL.Host) {
				for k := range headers {
					next.Header.Del(k)
				}
				next.Header.Del("Referer")
				break
			}
		}
		return nil
	}
	return c.Do(req)
}

// MediaPolicy returns the URL policy of a source, when it has one.
func MediaPolicy(source Source) func(*url.URL) error {
	if policy, ok := source.(interface{ CheckMediaURL(*url.URL) error }); ok {
		return policy.CheckMediaURL
	}
	return nil
}

// HeadersStay returns client, or, with headers to send, one like it that
// asks a redirect to a host other than host, or one below it, without them,
// so that what a source asks with, such as its token, reaches no one else.
// It's the rule Go keeps for Cookie and Authorization, for every header.
func HeadersStay(client *http.Client, host string, headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return client
	}
	host = strings.ToLower(host)
	c := *client
	c.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		if to := strings.ToLower(next.URL.Hostname()); to != host && !strings.HasSuffix(to, "."+host) {
			for k := range headers {
				next.Header.Del(k)
			}
		}
		return nil
	}
	return &c
}

// FileURL is the file:// URL of a path.
func FileURL(path string) string {
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path // C:/Music
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

// PathOf returns the path a file:// URL names, and whether u is one.
func PathOf(u string) (string, bool) {
	if !strings.HasPrefix(u, "file:") {
		return "", false
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Path == "" {
		return "", false
	}
	path := parsed.Path
	if runtime.GOOS == "windows" {
		path = strings.TrimPrefix(path, "/")
	}
	return filepath.FromSlash(path), true
}
