package rss

import (
	"fmt"
	"time"
	"os"
	"io/ioutil"
	"github.com/gorilla/feeds"
	"github.com/codingneo/tweetsbot/ranking"
)

const RSS_FILE = "../static/rss.xml" 

func UpdateRSSFeed(toplist []ranking.Item) {
	now := time.Now()
	feed := &feeds.Feed{
    Title:       "Data Science Daily",
    Link:        &feeds.Link{Href: "http://datasciencedaily.org"},
    Description: "The top list of most mentioned stuff about Data Science, Big Data and Machine Learning on Twitter",
    Author:      &feeds.Author{"Yiqun Hu", "yiqun.hu@gmail.com"},
    Created:     now,
	}

	feed.Items = make([]*feeds.Item, len(toplist))

	for idx, item := range toplist {
		fmt.Println(item)
		feed.Items[idx] =  
			&feeds.Item{
				Title: item.Title,
				Link:	&feeds.Link{Href: item.Url},
				Description: item.Description,
				Created: now,
			}
	}

	rss, err := feed.ToRss()
	err = os.Remove(RSS_FILE)
	if (err != nil) {
		fmt.Println("[RSS] remove feed error: ", RSS_FILE)
	}

	err = ioutil.WriteFile(RSS_FILE, []byte(rss), os.ModePerm)
	if (err != nil) {
		fmt.Println("[RSS] write feed error: ", RSS_FILE)
	}
}
