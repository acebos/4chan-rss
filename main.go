package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/k3a/html2text"
	"github.com/mmcdole/gofeed"
	"github.com/moshee/go-4chan-api/api"
	"mvdan.cc/xurls/v2"
)

var URLRegex = xurls.Strict()

var options struct {
	boardNames   string
	pages        uint
	replies      uint
	filterString string
}

func init() {
	flag.UintVar(&options.replies, "n", 10, "Minimum number of replies required to include a thread")
	flag.UintVar(&options.pages, "p", 1, "Number of pages to fetch per board")
	flag.StringVar(&options.boardNames, "b", "news", "Comma-separated list of board names")
	flag.StringVar(&options.filterString, "f", "", "String to filter out from thread titles (e.g. 'general')")
}

func main() {
	if rss, err := run(); err != nil {
		flag.Usage()
		log.Fatal(err)
	} else {
		err := os.WriteFile("rss.xml", []byte(rss), 0644)
		if err != nil {
			log.Fatal("Failed to write RSS to file:", err)
		}
		fmt.Println("RSS feed saved to rss.xml")
	}
}

func run() (string, error) {
	flag.Parse()
	if options.pages == 0 {
		return "", fmt.Errorf("page count (-p) must be greater than 0")
	}

	now := time.Now()
	feed := &gofeed.Feed{
		Title:       fmt.Sprintf("4chan threads from multiple boards"),
		Link:        "https://boards.4channel.org/", // Link as a string
		Description: fmt.Sprintf("Threads from multiple boards with more than %d replies", options.replies),
		Author: &gofeed.Person{
			Name: "Anon",
		},
		Updated: now.Format(time.RFC3339), // Convert time to string in RFC3339 format
	}

	boards := strings.Split(options.boardNames, ",")
	var allItems []*gofeed.Item
	for _, board := range boards {
		threads, err := getThreads(board, options.pages)
		if err != nil {
			return "", err
		}
		items := processThreads(threads, board)
		allItems = append(allItems, items...)
	}

	// Sorting items by Published date
	sort.Slice(allItems, func(i, j int) bool {
		time1, _ := time.Parse(time.RFC3339, allItems[i].Published)
		time2, _ := time.Parse(time.RFC3339, allItems[j].Published)
		return time1.After(time2)
	})

	feed.Items = allItems

	// Manually serialize feed to XML (RSS)
	rssXML, err := toRSSXML(feed)
	if err != nil {
		return "", err
	}

	return rssXML, nil
}

func toRSSXML(feed *gofeed.Feed) (string, error) {
	type RSS struct {
		XMLName xml.Name    `xml:"rss"`
		Version string      `xml:"version,attr"`
		Channel *RSSChannel `xml:"channel"`
	}

	// Create a new channel with the necessary fields
	rssChannel := &RSSChannel{
		Title:       feed.Title,
		Link:        feed.Link,
		Description: feed.Description,
		Author:      feed.Author.Name,
		Updated:     feed.Updated,
	}

	// Add all items to the channel
	for _, item := range feed.Items {
		rssChannel.Items = append(rssChannel.Items, RSSItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Author:      item.Author.Name,
			Published:   item.Published,
		})
	}

	rss := &RSS{
		Version: "2.0",
		Channel: rssChannel,
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return "", err
	}
	return string(output), nil
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Author      string    `xml:"managingEditor"`
	Updated     string    `xml:"lastBuildDate"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Author      string `xml:"author"`
	Published   string `xml:"pubDate"`
}

func getThreads(board string, pages uint) (threads []*api.Thread, err error) {
	for i := 0; i < int(pages); i++ {
		newthreads, err := api.GetIndex(board, i)
		if err != nil {
			return nil, err
		}
		threads = append(threads, newthreads...)
	}
	return
}

func processThreads(threads []*api.Thread, board string) []*gofeed.Item {
	var items []*gofeed.Item
	for _, thread := range threads {
		if thread.Replies() < int(options.replies) {
			continue
		}
		title := getTitle(thread.OP)
		if options.filterString != "" && strings.Contains(strings.ToLower(title), strings.ToLower(options.filterString)) {
			continue
		}
		item := processPost(thread.OP, board)
		item.Title = fmt.Sprintf("[%3d] %s", min(999, thread.Replies()), item.Title)
		items = append(items, item)
	}
	return items
}

func processPost(post *api.Post, board string) *gofeed.Item {
	item := &gofeed.Item{}
	item.Title = getTitle(post)
	item.Link = fmt.Sprintf("https://boards.4channel.org/%s/thread/%d/", board, post.Id)

	item.Description = anchorize(strings.ReplaceAll(post.Comment, "<wbr>", ""))
	if post.File != nil {
		item.Description += fmt.Sprintf("<p>Original filename: %s%s</p>", post.File.Name, post.File.Ext)

		item.Description += fmt.Sprintf(
			"<a href='%s'><img alt='%s' src='%s'/></a>",
			post.ImageURL(),
			post.File.Name+post.File.Ext,
			post.ThumbURL(),
		)
	}

	item.Author = &gofeed.Person{Name: post.Name}
	item.Published = post.Time.Format(time.RFC3339) // `Published` should be a string in RFC3339 format
	return item
}

func anchorize(comment string) string {
	return URLRegex.ReplaceAllString(comment, "<a href='$0'>$0</a>")
}

func getTitle(post *api.Post) string {
	title := post.Subject
	if title == "" {
		title = html2text.HTML2Text(post.Comment)
		title = substring(title, 80)
		title = strings.TrimSpace(title)
	}
	if title == "" && post.File != nil {
		title = substring(post.File.Name+post.File.Ext, 80)
	}
	if title == "" {
		title = "no title"
	}
	return title
}

func substring(s string, end int) string {
	unline := strings.ReplaceAll(s, "\n", " ")
	return unline[:min(len(s), end-1)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
