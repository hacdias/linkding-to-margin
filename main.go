package main

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	linkdingEndpoint := os.Getenv("LINKDING_ENDPOINT")
	linkdingApiKey := os.Getenv("LINKDING_API_KEY")
	atprotoHost := os.Getenv("ATPROTO_HOST")
	atprotoIdentifier := os.Getenv("ATPROTO_IDENTIFIER")
	atprotoPassword := os.Getenv("ATPROTO_PASSWORD")
	ignoreArchived := os.Getenv("IGNORE_ARCHIVED") == "true"
	dryRun := os.Getenv("DRY_RUN") == "true"
	processedIDsFile := os.Getenv("PROCESSED_IDS_FILE")
	if processedIDsFile == "" {
		processedIDsFile = "processed-bookmarks.csv"
	}

	xrpc, err := getXrpcClient(atprotoHost, atprotoIdentifier, atprotoPassword)
	if err != nil {
		log.Fatal(err)
	}

	bookmarks, err := fetchBookmarks(linkdingEndpoint, linkdingApiKey)
	if err != nil {
		log.Fatal(err)
	}

	processedIDs, err := loadProcessedIDs(processedIDsFile)
	if err != nil {
		log.Fatal(err)
	}

	err = importBookmarks(context.Background(), xrpc, bookmarks, dryRun, ignoreArchived, processedIDsFile, processedIDs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Bookmarks: %d\n", len(bookmarks))
}

type bookmark struct {
	ID                 int       `json:"id,omitempty"`
	URL                string    `json:"url,omitempty"`
	Title              string    `json:"title,omitempty"`
	Description        string    `json:"description,omitempty"`
	Notes              string    `json:"notes,omitempty"`
	WebsiteTitle       string    `json:"website_title,omitempty"`
	WebsiteDescription string    `json:"website_description,omitempty"`
	WebArchiveURL      string    `json:"web_archive_snapshot_url,omitempty"`
	IsArchived         bool      `json:"is_archived,omitempty"`
	Unread             bool      `json:"unread,omitempty"`
	Shared             bool      `json:"shared,omitempty"`
	TagNames           []string  `json:"tag_names,omitempty"`
	DateAdded          time.Time `json:"date_added,omitempty"`
	DateModified       time.Time `json:"date_modified,omitempty"`
}

type bookmarksResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous any    `json:"previous"`
	Results  []bookmark
}

func fetchBookmarks(endpoint, key string) ([]bookmark, error) {
	httpClient := &http.Client{
		Timeout: 2 * time.Minute,
	}
	endpoint = fmt.Sprintf("%s/api/bookmarks?", strings.TrimSuffix(endpoint, "/"))

	var bookmarks []bookmark
	for p := 0; ; p++ {
		q := url.Values{}
		q.Set("limit", "100")
		q.Set("offset", strconv.Itoa((p)*100))

		req, err := http.NewRequest(http.MethodGet, endpoint+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", "Token "+key)

		res, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		var data *bookmarksResponse
		err = json.NewDecoder(res.Body).Decode(&data)
		if err != nil {
			return nil, err
		}

		bookmarks = append(bookmarks, data.Results...)
		if len(data.Results) == 0 {
			break
		}
	}

	return bookmarks, nil
}

func getXrpcClient(host, identifier, password string) (*xrpc.Client, error) {
	client := &xrpc.Client{
		Host: host,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sess, err := atproto.ServerCreateSession(ctx, client, &atproto.ServerCreateSession_Input{
		Identifier: identifier,
		Password:   password,
	})
	if err != nil {
		return nil, err
	}

	client.Auth = &xrpc.AuthInfo{
		AccessJwt:  sess.AccessJwt,
		RefreshJwt: sess.RefreshJwt,
		Handle:     sess.Handle,
		Did:        sess.Did,
	}

	return client, nil
}

func importBookmarks(ctx context.Context, client *xrpc.Client, bookmarks []bookmark, dryRun, ignoreArchived bool, processedIDsFile string, processedIDs map[int]bool) error {
	var writer *csv.Writer
	var file *os.File
	var err error

	// Open processed IDs file for writing (unless dry run)
	if !dryRun {
		file, err = os.OpenFile(processedIDsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = csv.NewWriter(file)
		defer writer.Flush()
	}

	for _, bookmark := range bookmarks {
		// Skip already processed bookmarks
		if processedIDs[bookmark.ID] {
			log.Printf("Skipping bookmark %d (already processed)", bookmark.ID)
			continue
		}

		if bookmark.IsArchived && ignoreArchived {
			continue
		}

		var (
			uri string
			err error
		)

		if bookmark.Notes != "" {
			uri, err = createAnnotation(ctx, client, &bookmark, dryRun)
		} else {
			uri, err = createBookmark(ctx, client, &bookmark, dryRun)
		}
		if err != nil {
			return err
		}

		fmt.Printf("Processed %d: %s --> %s\n", bookmark.ID, bookmark.URL, uri)

		// Write processed ID to file
		if !dryRun {
			err = writer.Write([]string{fmt.Sprintf("%d", bookmark.ID), uri})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func createBookmark(ctx context.Context, client *xrpc.Client, b *bookmark, dryRun bool) (string, error) {
	record := map[string]any{
		"$type":      "at.margin.bookmark",
		"title":      b.Title,
		"source":     b.URL,
		"createdAt":  b.DateAdded.Format(time.RFC3339),
		"sourceHash": hashURL(b.URL),
	}

	if len(b.TagNames) > 0 {
		record["tags"] = b.TagNames
	}

	if dryRun {
		printJson(record)
		return "", nil
	}

	result, err := agnostic.RepoCreateRecord(ctx, client, &agnostic.RepoCreateRecord_Input{
		Collection: "at.margin.bookmark",
		Repo:       client.Auth.Did,
		Record:     record,
	})
	if err != nil {
		return "", err
	}

	return result.Uri, nil
}

func createAnnotation(ctx context.Context, client *xrpc.Client, b *bookmark, dryRun bool) (string, error) {
	record := map[string]any{
		"$type": "at.margin.annotation",
		"body": map[string]string{
			"format": "text/markdown",
			"value":  b.Notes,
		},
		"target": map[string]any{
			"title":      b.Title,
			"source":     b.URL,
			"selector":   nil,
			"sourceHash": hashURL(b.URL),
		},
		"createdAt":  b.DateAdded.Format(time.RFC3339),
		"motivation": "commenting",
	}

	if len(b.TagNames) > 0 {
		record["tags"] = b.TagNames
	}

	if dryRun {
		printJson(record)
		return "", nil
	}

	result, err := agnostic.RepoCreateRecord(ctx, client, &agnostic.RepoCreateRecord_Input{
		Collection: "at.margin.annotation",
		Repo:       client.Auth.Did,
		Record:     record,
	})
	if err != nil {
		return "", err
	}

	return result.Uri, nil
}

func loadProcessedIDs(filename string) (map[int]bool, error) {
	processedIDs := make(map[int]bool)

	file, err := os.Open(filename)
	if err != nil {
		// File doesn't exist yet, which is fine
		if os.IsNotExist(err) {
			return processedIDs, nil
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Printf("Warning: error reading CSV: %v", err)
			continue
		}

		if len(record) < 1 {
			continue
		}

		id, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			log.Printf("Warning: invalid ID in processed file: %s", record[0])
			continue
		}
		processedIDs[id] = true
	}

	return processedIDs, nil
}

// Directly from margin.at's code
func hashURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return hashString(rawURL)
	}

	host := strings.ToLower(parsed.Host)
	host = strings.TrimPrefix(host, "www.")

	normalized := host + parsed.Path
	if parsed.RawQuery != "" {
		normalized += "?" + parsed.RawQuery
	}
	normalized = strings.TrimSuffix(normalized, "/")

	return hashString(normalized)
}

// Directly from margin.at's code
func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func printJson(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("Error marshalling to JSON: %v", err)
		return
	}
	fmt.Println("[DRY RUN]", string(b))
}
