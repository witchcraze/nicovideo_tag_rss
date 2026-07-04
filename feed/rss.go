package feed

import (
	"fmt"
	"time"

	"github.com/gorilla/feeds"
	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/nico"
)

// GenerateRSS creates an RSS 2.0 XML feed from the given configuration and videos.
func GenerateRSS(cfg config.FeedConfig, videos []nico.Video) ([]byte, error) {
	now := time.Now()

	feed := &feeds.Feed{
		Title:       cfg.Title,
		Link:        &feeds.Link{Href: "https://github.com/witchcraze/nicovideo_tag_rss"},
		Description: cfg.Description,
		Created:     now,
	}

	for _, v := range videos {
		item := &feeds.Item{
			Id:          v.ID,
			Title:       v.Title,
			Link:        &feeds.Link{Href: v.Link},
			Description: fmt.Sprintf("<p><img src=\"%s\" alt=\"%s\"></p><p>%s</p>", v.Thumbnail, v.Title, v.Description),
			Author:      &feeds.Author{Name: v.Author},
			Created:     v.PubDate,
		}
		feed.Items = append(feed.Items, item)
	}

	rss, err := feed.ToRss()
	if err != nil {
		return nil, fmt.Errorf("failed to generate rss: %w", err)
	}

	return []byte(rss), nil
}
