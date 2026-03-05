package newznab

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func Example() {
	client := New("http://my-usenet-indexer", "my-api-key", 1234, false)
	_, _ = client.Capabilities()

	// Search using a tvrage id:
	categories := []int{
		CategoryTVHD,
		CategoryTVSD,
	}
	_, _ = client.SearchWithTVRage(categories, 35048, 3, 1)

	// Search using an imdb id:
	categories = []int{
		CategoryMovieHD,
		CategoryMovieBluRay,
	}
	_, _ = client.SearchWithIMDB(categories, "0364569")

	// Search using a tvmaze id:
	categories = []int{
		CategoryTVHD,
		CategoryTVSD,
	}
	_, _ = client.SearchWithTVMaze(categories, 80, 3, 1)

	// Search using a name and set of categories:
	_, _ = client.SearchWithQuery(categories, "Oldboy", "movie")

	// Get latest releases for set of categories:
	_, _ = client.SearchWithQuery(categories, "", "movie")

	// Load latest releases via RSS:
	_, _ = client.LoadRSSFeed(categories, 50)

	// Load latest releases via RSS up to a given NZB id:
	_, _ = client.LoadRSSFeedUntilNZBID(categories, 50, "nzb-guid", 15)
}

func TestUsenetCrawlerClient(t *testing.T) {
	apiKey := "gibberish"

	// Set up our mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var f []byte
		var err error

		reg := regexp.MustCompile(`\W`)
		fixedPath := reg.ReplaceAllString(r.URL.RawQuery, "_")

		if r.URL.Query()["t"][0] == "get" {
			// Fetch nzb
			nzbID := r.URL.Query()["id"][0]
			filePath := fmt.Sprintf("./fixtures/nzbs/%v.nzb", nzbID)
			f, err = os.ReadFile(filePath)
		} else {
			// Get xml
			filePath := fmt.Sprintf("./fixtures%v/%v.xml", r.URL.Path, fixedPath)
			f, err = os.ReadFile(filePath)
		}

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("File not found")) // nolint:errcheck
		} else {
			w.Write(f) // nolint:errcheck
		}
	}))

	defer ts.Close()

	t.Run("user agent is configurable", func(t *testing.T) {
		const customUserAgent = "Ovrlord-Test/2.0"

		requestUserAgents := make([]string, 0, 2)
		tsUserAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestUserAgents = append(requestUserAgents, r.Header.Get("User-Agent"))
			_, _ = w.Write([]byte("<caps></caps>"))
		}))
		defer tsUserAgent.Close()

		client := New(tsUserAgent.URL, apiKey, 1234, false)

		_, err := client.Capabilities()
		require.NoError(t, err)

		client.SetUserAgent(customUserAgent)
		_, err = client.Capabilities()
		require.NoError(t, err)

		require.Len(t, requestUserAgents, 2)
		require.Equal(t, defaultUserAgent, requestUserAgents[0])
		require.Equal(t, customUserAgent, requestUserAgents[1])
	})

	t.Run("torznab client", func(t *testing.T) {
		client := New(ts.URL, apiKey, 1234, true)

		t.Run("Simple query search", func(t *testing.T) {
			categories := []int{CategoryTVHD}
			results, err := client.SearchWithQuery(categories, "Supernatural S11E01", "tvshows")
			require.NoError(t, err)
			require.NotEmpty(t, results, "expected results")
		})
	})

	t.Run("nzb client", func(t *testing.T) {
		client := New(ts.URL, apiKey, 1234, false)
		categories := []int{CategoryTVSD}

		t.Run("invalid search", func(t *testing.T) {
			_, err := client.SearchWithTVDB(categories, 1234, 9, 2)
			require.Error(t, err, "expected an error")
		})

		t.Run("invalid api usage", func(t *testing.T) {
			_, err := client.SearchWithTVDB(categories, 5678, 9, 2)
			require.Error(t, err, "expected an error")
			require.EqualError(t, err, "newznab api error 100: Invalid API Key")
		})

		t.Run("valid category and TheTVDB id", func(t *testing.T) {
			results, err := client.SearchWithTVDB(categories, 75682, 10, 1)
			require.NoError(t, err)
			require.NotEmpty(t, results, "expected results")
		})

		t.Run("valid category and TVMaze id", func(t *testing.T) {
			results, err := client.SearchWithTVMaze(categories, 65, 10, 1)
			require.NoError(t, err)
			require.NotEmpty(t, results, "expected results")
		})

		t.Run("valid category and tvrage id", func(t *testing.T) {
			results, err := client.SearchWithTVRage(categories, 2870, 10, 1)
			require.NoError(t, err)
			require.NotEmpty(t, results, "expected results")

			t.Run("populate comments", func(t *testing.T) {
				nzb := results[1]
				require.Empty(t, nzb.Comments)
				require.NotZero(t, nzb.NumComments)
				err := client.PopulateComments(&nzb)
				require.NoError(t, err)
				require.NotEmpty(t, nzb.Comments, "expected at least one comment")
				for _, comment := range nzb.Comments {
					require.NotEmpty(t, comment, "comment should not be empty")
				}
			})

			t.Run("download url", func(t *testing.T) {
				url, err := client.NZBDownloadURL(results[0])
				require.NoError(t, err)
				require.NotEmpty(t, url, "expected a url")
			})

			t.Run("download nzb", func(t *testing.T) {
				bytes, err := client.DownloadNZB(results[0])
				require.NoError(t, err)
				require.NotEmpty(t, bytes, "expected to download something")
			})
		})

		t.Run("multiple categories and IMDB id", func(t *testing.T) {
			cats := []int{
				CategoryMovieHD,
				CategoryMovieBluRay,
			}
			results, err := client.SearchWithIMDB(cats, "0371746")
			require.NoError(t, err)
			require.NotEmpty(t, results, "expected results")

			require.Equal(t, "2040", results[0].Category[1])
			require.Equal(t, "2050", results[22].Category[1])
		})

		t.Run("single category and IMDB id", func(t *testing.T) {
			cats := []int{CategoryMovieHD}
			results, err := client.SearchWithIMDB(cats, "0364569")
			require.NoError(t, err)
			require.NotEmpty(t, results, "expected results")

			t.Run("movie specific fields", func(t *testing.T) {
				require.Equal(t, "0364569", results[0].IMDBID)
				require.Equal(t, "Oldboy", results[0].IMDBTitle)
				require.Equal(t, 2003, results[0].IMDBYear)
				require.Equal(t, float32(8.4), results[0].IMDBScore)
				require.Equal(t, "https://dognzb.cr/content/covers/movies/thumbs/364569.jpg", results[0].CoverURL)
			})
		})

		t.Run("recent items via RSS", func(t *testing.T) {
			num := 50
			categories := []int{CategoryMovieAll, CategoryTVAll}

			t.Run("recent items", func(t *testing.T) {
				results, err := client.LoadRSSFeed(categories, num)
				require.NoError(t, err)
				require.Len(t, results, num)
				require.Equal(t, "bcdbf3f1e7a1ef964527f1d40d5ec639", results[0].ID)
				require.Equal(t, "030517-VSHS0101720WDA20H264V", results[6].Title)

				t.Run("airdate with RFC1123Z format", func(t *testing.T) {
					require.Equal(t, 2017, results[7].AirDate.Year())
				})

				t.Run("usenetdate with RFC3339 format", func(t *testing.T) {
					require.Equal(t, 2017, results[7].UsenetDate.Year())
				})
			})

			t.Run("up until", func(t *testing.T) {
				results, err := client.LoadRSSFeedUntilNZBID(categories, num, "29527a54ac54bb7533abacd7dad66a6a", 0)
				require.NoError(t, err)
				require.Len(t, results, 101)

				t.Run("boundary results", func(t *testing.T) {
					require.Equal(t, "8841b21c4d2fb96f0d47ca24cae9a5b7", results[0].ID)
					require.Equal(t, "2c6c0e2ac562db69d8b3646deaf2d0cd", results[len(results)-1].ID)
				})

				t.Run("RSS up until with failures/retries", func(t *testing.T) {
					results, err := client.LoadRSSFeedUntilNZBID(categories, num, "does-not-exist", 2)
					require.NoError(t, err)
					require.Len(t, results, 100)
				})
			})
		})

		t.Run("single nzb details", func(t *testing.T) {
			d, err := client.Details("4694b91a86adc4ebd3b289687ebf4b0d")
			require.NoError(t, err)
			require.Equal(t, "Car.Craft-July.2015", d.Channel.Item.Title)
		})
	})
}
