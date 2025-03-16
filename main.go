package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/k3a/html2text"
	"github.com/moshee/go-4chan-api/api"
	"mvdan.cc/xurls/v2"
)

var URLRegex = xurls.Strict()

var options struct {
	boardNames string
	pages      uint
	replies    uint
}

func init() {
	flag.UintVar(&options.replies, "n", 10, "cutoff of number of replies on thread")
	flag.UintVar(&options.pages, "p", 1, "number of pages/request to get/make")
	flag.StringVar(&options.boardNames, "b", "news", "comma-separated list of board names")
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
	now := time.Now()
	feed := &feeds.Feed{
		Title:       fmt.Sprintf("4chan threads from multiple boards"),
		Link:        &feeds.Link{Href: "https://boards.4channel.org/"},
		Description: fmt.Sprintf("threads from multiple boards with more than %d comments", options.replies),
		Author:      &feeds.Author{Name: "Anon"},
		Created:     now,
	}

	boards := strings.Split(options.boardNames, ",")
	var allItems []*feeds.Item
	for _, board := range boards {
		threads, err := getThreads(board, options.pages)
		if err != nil {
			return "", err
		}
		items := processThreads(threads, board)
		allItems = append(allItems, items...)
	}

	feed.Items = allItems
	atom, err := feed.ToAtom()
	if err != nil {
		return "", err
	}
	return atom, nil
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

func processThreads(threads []*api.Thread, board string) []*feeds.Item {
	var items []*feeds.Item
	for _, thread := range threads {
		if thread.Replies() < int(options.replies) {
			continue
		}
		item := processPost(thread.OP, board)
		item.Title = fmt.Sprintf("[%3d] %s", min(999, thread.Replies()), item.Title)
		items = append(items, item)
	}
	return items
}

func processPost(post *api.Post, board string) *feeds.Item {
	item := &feeds.Item{}
	item.Title = getTitle(post)
	item.Link = &feeds.Link{
		Href: fmt.Sprintf("https://boards.4channel.org/%s/thread/%d/", board, post.Id),
	}
	item.Description = anchorize(strings.ReplaceAll(post.Comment, "<wbr>", ""))
	if post.File != nil {
		item.Description += fmt.Sprintf(
			"<img alt='%s' src='%s'/>",
			post.File.Name+post.File.Ext,
			post.ImageURL(),
		)
		item.Description += fmt.Sprintf(
			"<img alt='%d.jpg' src='%s'/>",
			post.File.Id,
			post.ThumbURL(),
		)
	}
	item.Author = &feeds.Author{Name: post.Name}
	item.Created = post.Time
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
	} else {
		return b
	}
}
