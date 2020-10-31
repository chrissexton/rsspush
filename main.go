package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/mmcdole/gofeed"
)

const pushoverURL = "https://api.pushover.net/1/messages.json"

const envToken = "PUSHOVER_RSS_TOKEN"
const envUser = "PUSHOVER_USER"
const envData = "FEED_DATA"

const defaultData = "feeds.json"

var (
	pushoverToken = os.Getenv(envToken)
	pushoverUser  = os.Getenv(envUser)
	dataFile      = os.Getenv(envData)
)

type feedConfig struct {
	Name     string    `json:"name"`
	URL      string    `json:"url"`
	LastSeen time.Time `json:"lastSeen"`
}

func main() {
	flag.Parse()

	if dataFile == "" {
		dataFile = defaultData
	}

	if _, err := os.Stat(dataFile); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open configuration file: %s", dataFile)
		os.Exit(1)
	}
	data, err := ioutil.ReadFile(dataFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open configuration file: %s", err)
		os.Exit(1)
	}
	configs := []feedConfig{}
	if err = json.Unmarshal(data, &configs); err != nil {
		fmt.Fprintf(os.Stderr, "Could not read data file: %s", err)
		os.Exit(1)
	}

	for i, feed := range configs {
		fp := gofeed.NewParser()
		f, _ := fp.ParseURL(feed.URL)
		for _, item := range f.Items {
			if item.PublishedParsed.After(feed.LastSeen) {
				err = push(fmt.Sprintf("New post to %s: %s", feed.Name, item.Title), item.Link, item.Title)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error sending pushover: %s", err)
					continue
				}
				configs[i].LastSeen = *item.PublishedParsed
			}
		}
	}

	data, _ = json.Marshal(configs)
	if err = ioutil.WriteFile(dataFile, data, 0666); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving data file: %s", err)
		os.Exit(1)
	}
}

func push(msg, linkURL, linkTitle string) error {
	values := url.Values{}
	values.Set("token", pushoverToken)
	values.Set("user", pushoverUser)
	values.Set("message", msg)
	if linkURL != "" {
		values.Set("url", linkURL)
		values.Set("url_title", linkTitle)
	}
	resp, err := http.PostForm(pushoverURL, values)
	if err != nil {
		return err
	}
	if resp.StatusCode > 399 {
		return fmt.Errorf("pushever returned status %d", resp.StatusCode)
	}
	return nil
}
