package rss

import (
	"encoding/xml"
	"net/http"
	"context"
	"html"
	"fmt"
)

type RSSFeed struct {
	Channel struct {
		Title 		string `xml:"title"`
		Link 		string `xml:"link"`
		Description	string `xml:"description"`
		Item		[]RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title		string `xml:"title"`
	Link		string `xml:"link"`
	Description	string `xml:"description"`
	PubDate		string `xml:"pubDate"`
}

func FetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "gator")
	
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to complete the request: %w", err)
	}
	defer res.Body.Close()

	decoder := xml.NewDecoder(res.Body)
	var rssFeed RSSFeed
	if err = decoder.Decode(&rssFeed); err != nil {
		return nil, fmt.Errorf("error decoding the response body: %w", err)
	}

	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
	rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)
	for i := 0; i < len(rssFeed.Channel.Item); i++ {
		rssFeed.Channel.Item[i].Title = html.UnescapeString(rssFeed.Channel.Item[i].Title)
		rssFeed.Channel.Item[i].Description = html.UnescapeString(rssFeed.Channel.Item[i].Description)
	}
	

	return &rssFeed, nil	
}
