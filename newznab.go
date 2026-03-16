// Package newznab provides a client for interacting with a Newznab API
// Copied from https://github.com/mrobinsn/go-newznab
package newznab

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Various constants for categories
const (
	// TV Categories
	// CategoryTVAll is for all shows
	CategoryTVAll = 5000
	// CategoryTVForeign is for foreign shows
	CategoryTVForeign = 5020
	// CategoryTVSD is for standard-definition shows
	CategoryTVSD = 5030
	// CategoryTVHD is for high-definition shows
	CategoryTVHD = 5040
	// CategoryTVUHD is for UHD shows
	CategoryTVUHD = 5045
	// CategoryTVOther is for other shows
	CategoryTVOther = 5050
	// CategoryTVSport is for sports shows
	CategoryTVSport = 5060

	// Movie categories
	// CategoryMovieAll is for all movies
	CategoryMovieAll = 2000
	// CategoryMovieForeign is for foreign movies
	CategoryMovieForeign = 2010
	// CategoryMovieOther is for other movies
	CategoryMovieOther = 2020
	// CategoryMovieSD is for standard-definition movies
	CategoryMovieSD = 2030
	// CategoryMovieHD is for high-definition movies
	CategoryMovieHD = 2040
	// CategoryMovieUHD is for UHD movies
	CategoryMovieUHD = 2045
	// CategoryMovieBluRay is for blu-ray movies
	CategoryMovieBluRay = 2050
	// CategoryMovie3D is for 3-D movies
	CategoryMovie3D = 2060
)

// Client is a type for interacting with a newznab or torznab api
type Client struct {
	apikey        string
	apiBaseURL    string
	apiUserID     int
	client        *http.Client
	userAgent     string
	ExtendedAttrs bool
}

// New returns a new instance of Client
func New(baseURL string, apikey string, userID int, insecure bool) Client {
	ret := Client{
		apikey:     apikey,
		apiBaseURL: baseURL,
		apiUserID:  userID,
		userAgent:  defaultUserAgent,
	}
	if insecure {
		ret.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}}
	} else {
		ret.client = &http.Client{}
	}
	return ret
}

// SetUserAgent sets the User-Agent header value for requests made by this client.
func (c *Client) SetUserAgent(userAgent string) {
	c.userAgent = strings.TrimSpace(userAgent)
}

// SearchWithTVRage returns NZBs for the given parameters
func (c Client) SearchWithTVRage(categories []int, tvRageID int, season int, episode int) ([]NZB, error) {
	return c.search(url.Values{
		"rid":     []string{strconv.Itoa(tvRageID)},
		"cat":     c.splitCats(categories),
		"season":  []string{strconv.Itoa(season)},
		"episode": []string{strconv.Itoa(episode)},
		"t":       []string{"tvsearch"},
	})
}

// SearchWithTVDB returns NZBs for the given parameters
func (c Client) SearchWithTVDB(categories []int, tvDBID int, season int, episode int) ([]NZB, error) {
	return c.search(url.Values{
		"tvdbid":  []string{strconv.Itoa(tvDBID)},
		"cat":     c.splitCats(categories),
		"season":  []string{strconv.Itoa(season)},
		"episode": []string{strconv.Itoa(episode)},
		"t":       []string{"tvsearch"},
	})
}

// SearchWithTVMaze returns NZBs for the given parameters
func (c Client) SearchWithTVMaze(categories []int, tvMazeID int, season int, episode int) ([]NZB, error) {
	return c.search(url.Values{
		"tvmazeid": []string{strconv.Itoa(tvMazeID)},
		"cat":      c.splitCats(categories),
		"season":   []string{strconv.Itoa(season)},
		"episode":  []string{strconv.Itoa(episode)},
		"t":        []string{"tvsearch"},
	})
}

// SearchWithIMDB returns NZBs for the given parameters
func (c Client) SearchWithIMDB(categories []int, imdbID string) ([]NZB, error) {
	return c.search(url.Values{
		"imdbid": []string{imdbID},
		"cat":    c.splitCats(categories),
		"t":      []string{"movie"},
	})
}

// SearchWithTMDB returns NZBs for the given parameters
func (c Client) SearchWithTMDB(categories []int, tmdbID string, searchType string) ([]NZB, error) {
	return c.search(url.Values{
		"tmdbid": []string{tmdbID},
		"cat":    c.splitCats(categories),
		"t":      []string{searchType},
	})
}

// SearchWithQuery returns NZBs for the given parameters
func (c Client) SearchWithQuery(categories []int, query string, searchType string) ([]NZB, error) {
	return c.search(url.Values{
		"q":   []string{query},
		"cat": c.splitCats(categories),
		"t":   []string{searchType},
	})
}

// FetchRecent returns the most recent NZBs for the given categories
func (c Client) FetchRecent(categories []int, searchType string) (NZBS []NZB, total int, offset int, err error) {
	return c.fetch(url.Values{
		"cat": c.splitCats(categories),
		"t":   []string{searchType},
	})
}

// LoadRSSFeed returns up to <num> of the most recent NZBs of the given categories.
func (c Client) LoadRSSFeed(categories []int, num int) ([]NZB, error) {
	return c.rss(url.Values{
		"num": []string{strconv.Itoa(num)},
		"t":   c.splitCats(categories),
		"dl":  []string{"1"},
	})
}

// Capabilities returns the capabilities of this tracker
func (c Client) Capabilities() (Capabilities, error) {
	return c.caps(url.Values{
		"t": []string{"caps"},
	})
}

// LoadRSSFeedUntilNZBID fetches NZBs until a given NZB id is reached.
func (c Client) LoadRSSFeedUntilNZBID(categories []int, num int, id string, maxRequests int) ([]NZB, error) {
	count := 0
	var nzbs []NZB
	for {
		partition, err := c.rss(url.Values{
			"num":    []string{strconv.Itoa(num)},
			"t":      c.splitCats(categories),
			"dl":     []string{"1"},
			"offset": []string{strconv.Itoa(num * count)},
		})
		count++
		if err != nil {
			return nil, err
		}
		for k, nzb := range partition {
			if nzb.ID == id {
				return append(nzbs, partition[:k]...), nil
			}
		}
		nzbs = append(nzbs, partition...)
		if maxRequests != 0 && count == maxRequests {
			break
		}
	}
	return nzbs, nil
}

// Details get the details of a particular nzb
func (c Client) Details(guid string) (Details, error) {
	return c.details(url.Values{
		"t":    []string{"details"},
		"guid": []string{guid},
	})
}

func (c Client) splitCats(cats []int) []string {
	categories := make([]string, 0, len(cats))
	for _, v := range cats {
		categories = append(categories, strconv.Itoa(v))
	}
	return []string{strings.Join(categories, ",")}
}

func (c Client) rss(vals url.Values) ([]NZB, error) {
	vals.Set("r", c.apikey)
	vals.Set("i", strconv.Itoa(c.apiUserID))
	nzbs, _, _, err := c.process(vals, rssPath)
	return nzbs, err
}

func (c Client) search(vals url.Values) ([]NZB, error) {
	vals.Set("apikey", c.apikey)
	nzbs, _, _, err := c.process(vals, apiPath)
	return nzbs, err
}

func (c Client) fetch(vals url.Values) ([]NZB, int, int, error) {
	vals.Set("apikey", c.apikey)
	return c.process(vals, apiPath)
}

func (c Client) caps(vals url.Values) (Capabilities, error) {
	vals.Set("apikey", c.apikey)
	resp, err := c.getURL(c.buildURL(vals, apiPath))
	if err != nil {
		return Capabilities{}, fmt.Errorf("failed to get capabilities: %w", err)
	}
	var cResp Capabilities
	if err = xml.Unmarshal(resp, &cResp); err != nil {
		return cResp, fmt.Errorf("failed to unmarshal xml: %w", err)
	}
	return cResp, nil
}

func (c Client) details(vals url.Values) (Details, error) {
	vals.Set("apikey", c.apikey)
	resp, err := c.getURL(c.buildURL(vals, apiPath))
	if err != nil {
		return Details{}, fmt.Errorf("failed to get details: %w", err)
	}
	var dResp Details
	if err = xml.Unmarshal(resp, &dResp); err != nil {
		return dResp, fmt.Errorf("failed to unmarshal xml: %w", err)
	}
	return dResp, nil
}

func (c Client) process(vals url.Values, path string) ([]NZB, int, int, error) {
	var nzbs []NZB
	var offset int
	var total int
	resp, err := c.getURL(c.buildURL(vals, path))
	if err != nil {
		return nzbs, offset, total, err
	}
	var feed SearchResponse
	err = xml.Unmarshal(resp, &feed)
	if err != nil {
		return nil, total, offset, fmt.Errorf("failed to unmarshal xml feed: %w", err)
	}
	if feed.ErrorCode != 0 {
		return nil, total, offset, fmt.Errorf("newznab api error %d: %s", feed.ErrorCode, feed.ErrorDesc)
	}
	offset = feed.Channel.Response.Offset
	total = feed.Channel.Response.Total
	for _, gotNZB := range feed.Channel.NZBs {
		nzb := NZB{
			Title:          gotNZB.Title,
			Description:    gotNZB.Description,
			PubDate:        gotNZB.Date.Add(0),
			DownloadURL:    gotNZB.Enclosure.URL,
			SourceEndpoint: c.apiBaseURL,
			SourceAPIKey:   c.apikey,
			UnmatchedAttrs: make(map[string]string),
		}
		for _, attr := range gotNZB.Attributes {
			switch attr.Name {
			case "tvairdate":
				if parsedAirDate, err := parseDate(attr.Value); err != nil {
					slog.Debug("newznab:Client:Search: failed to parse tvairdate", "err", err, "tvairdate", attr.Value)
				} else {
					nzb.AirDate = parsedAirDate
				}
			case "guid":
				nzb.ID = attr.Value
			case "size":
				parsedInt, _ := strconv.ParseInt(attr.Value, 10, 64)
				nzb.Size = parsedInt
			case "grabs":
				parsedInt, _ := strconv.ParseInt(attr.Value, 10, 32)
				nzb.NumGrabs = int(parsedInt)
			case "comments":
				parsedInt, _ := strconv.ParseInt(attr.Value, 10, 32)
				nzb.NumComments = int(parsedInt)
			case "seeders":
				parsedInt, _ := strconv.ParseInt(attr.Value, 10, 32)
				nzb.Seeders = int(parsedInt)
				nzb.IsTorrent = true
			case "peers":
				parsedInt, _ := strconv.ParseInt(attr.Value, 10, 32)
				nzb.Peers = int(parsedInt)
				nzb.IsTorrent = true
			case "infohash":
				nzb.InfoHash = attr.Value
				nzb.IsTorrent = true
			case "category":
				nzb.Category = append(nzb.Category, attr.Value)
			case "genre":
				nzb.Genre = attr.Value
			case "tvdbid":
				nzb.TVDBID = attr.Value
			case "rageid":
				nzb.TVRageID = attr.Value
			case "tvmazeid":
				nzb.TVMazeID = attr.Value
			case "info":
				nzb.Info = attr.Value
			case "season":
				nzb.Season = attr.Value
			case "episode":
				nzb.Episode = attr.Value
			case "tvtitle":
				nzb.TVTitle = attr.Value
			case "rating":
				parsedInt, _ := strconv.ParseInt(attr.Value, 10, 32)
				nzb.Rating = int(parsedInt)
			case "imdb":
				nzb.IMDBID = attr.Value
			case "imdbtitle":
				nzb.IMDBTitle = attr.Value
			case "imdbyear":
				parsedInt, _ := strconv.ParseInt(attr.Value, 10, 32)
				nzb.IMDBYear = int(parsedInt)
			case "imdbscore":
				parsedFloat, _ := strconv.ParseFloat(attr.Value, 32)
				nzb.IMDBScore = float32(parsedFloat)
			case "tmdbid":
				parsedInt, _ := strconv.ParseInt(attr.Value, 10, 32)
				nzb.TMDBID = int(parsedInt)
			case "tmdbyear":
				parsedInt, _ := strconv.ParseInt(attr.Value, 10, 32)
				nzb.TMDBYear = int(parsedInt)
			case "coverurl":
				nzb.CoverURL = attr.Value
			case "usenetdate":
				if parsedUsetnetDate, err := parseDate(attr.Value); err != nil {
					slog.Debug("failed to parse usenetdate", "err", err, "usenetdate", attr.Value)
				} else {
					nzb.UsenetDate = parsedUsetnetDate
				}
			case "resolution":
				nzb.Resolution = attr.Value
			default:
				slog.Debug("encountered unknown attribute", "name", attr.Name, "value", attr.Value)
				nzb.UnmatchedAttrs[attr.Name] = attr.Value
			}
		}
		if nzb.Size == 0 {
			nzb.Size = gotNZB.Size
		}
		nzbs = append(nzbs, nzb)
	}
	return nzbs, total, offset, nil
}

// PopulateComments fills in the Comments for the given NZB
func (c Client) PopulateComments(nzb *NZB) error {
	data, err := c.getURL(c.buildURL(url.Values{
		"t":      []string{"comments"},
		"id":     []string{nzb.ID},
		"apikey": []string{c.apikey},
	}, apiPath))
	if err != nil {
		return err
	}
	var resp commentResponse
	err = xml.Unmarshal(data, &resp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal comments xml data: %w", err)
	}

	for _, rawComment := range resp.Channel.Comments {
		comment := Comment{
			Title:   rawComment.Title,
			Content: rawComment.Description,
		}
		if parsedPubDate, err := time.Parse(time.RFC1123Z, rawComment.PubDate); err != nil {
			slog.Debug("failed to parse comment date", "err", err, "pubdate", rawComment.PubDate)
		} else {
			comment.PubDate = parsedPubDate
		}
		nzb.Comments = append(nzb.Comments, comment)
	}
	return nil
}

// NZBDownloadURL returns a URL to download the NZB from
func (c Client) NZBDownloadURL(nzb NZB) (string, error) {
	return c.buildURL(url.Values{
		"t":      []string{"get"},
		"id":     []string{nzb.ID},
		"apikey": []string{c.apikey},
	}, apiPath)
}

// DownloadNZB returns the bytes of the actual NZB file for the given NZB
func (c Client) DownloadNZB(nzb NZB) ([]byte, error) {
	return c.getURL(c.NZBDownloadURL(nzb))
}

func (c Client) getURL(url string, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := c.userAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %s: %w", url, err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			slog.Debug("failed to close response body", "err", closeErr, "url", url)
		}
	}()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for %s: %w", url, err)
	}

	// Check for HTTP error status codes
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("http error %d: %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	return data, nil
}

func (c Client) buildURL(vals url.Values, path string) (string, error) {
	parsedURL, err := url.Parse(c.apiBaseURL + path)
	if err != nil {
		return "", fmt.Errorf("failed to parse base API url: %w", err)
	}

	if c.ExtendedAttrs {
		vals.Set("extended", "1")
	}

	parsedURL.RawQuery = vals.Encode()
	return parsedURL.String(), nil
}

func parseDate(date string) (time.Time, error) {
	formats := []string{time.RFC3339, time.RFC1123Z}
	var parsedTime time.Time
	var err error
	for _, format := range formats {
		if parsedTime, err = time.Parse(format, date); err == nil {
			return parsedTime, nil
		}
	}
	return parsedTime, fmt.Errorf("failed to parse date %s as one of %s", date, strings.Join(formats, ", "))
}

const (
	defaultUserAgent = "newznab-client/1.0"
	apiPath          = "/api"
	rssPath          = "/rss"
)

type commentResponse struct {
	Channel struct {
		Comments []rssComment `xml:"item"`
	} `xml:"channel"`
}

type rssComment struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}
