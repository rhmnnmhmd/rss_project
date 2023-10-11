package main

import (
	"context"
	"database/sql"
	"log"
	"rhmnnmhmd/rss_project/internal/database"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

func startScraping(db *database.Queries,
				   concurrency int,
				   timeBetweenRequest time.Duration) {
	log.Printf(
		"Starting scraping with concurrency %d and time between requests %s",
		concurrency,
		timeBetweenRequest,
	)

	ticker := time.NewTicker(timeBetweenRequest)

	for ; ; <-ticker.C {

		feeds, err := db.GetNextFeedsToFetch(
			context.Background(),
			int32(concurrency),
		)

		if err != nil {
			log.Printf("Error getting feeds to fetch: %s", err)
			continue
		}

		wg := &sync.WaitGroup{}

		for _, feed := range feeds {
			wg.Add(1)
			go scrapeFeed(db, wg, feed)
		}

		wg.Wait()
	}
}

func scrapeFeed(db *database.Queries, wg *sync.WaitGroup, feed database.Feed) {
	defer wg.Done()

	_, err := db.MarkFeedAsFetched(context.Background(), feed.ID)

	if err != nil {
		log.Printf("Error marking feed as fetched: %s", err)
		return
	}

	rssFeed, err := urlToFeed(feed.Url)

	if err != nil {
		log.Printf("Error getting feed: %s", err)
		return
	}

	for _, item := range rssFeed.Channel.Items {
		description := sql.NullString{}

		if item.Description != "" {
			description.String = item.Description
			description.Valid = true
		}

		pubAt, err := time.Parse(time.RFC1123Z, item.PubDate)

		if err != nil {
			log.Printf("Error parsing publish date: %s", err)
			continue
		}

		_, err = db.CreatePost(
			context.Background(),
			database.CreatePostParams{
				ID: uuid.New(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Title: item.Title,
				Description: description,
				PublishedAt: pubAt,
				Url: item.Link,
				FeedID: feed.ID,
			},
		)

		if err != nil {

			if strings.Contains(err.Error(), "duplicate key") {
				continue
			}

			log.Printf("Error creating post: %s", err)
			continue
		}
	}

	log.Printf(
		"Feed %s colleted, %v posts found",
		feed.Name,
		len(rssFeed.Channel.Items),
	)
}