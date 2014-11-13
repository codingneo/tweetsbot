package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"container/list"
	js "encoding/json"

	"github.com/kurrik/oauth1a"
	"github.com/codingneo/twittergo"
	"github.com/kurrik/json"
	"github.com/robfig/cron"
	//"github.com/PuerkitoBio/goquery"
	"github.com/advancedlogic/GoOse"
	"github.com/codingneo/tweetsbot/ranking"
)


func LoadCredentials() (client *twittergo.Client, err error) {
	credentials, err := ioutil.ReadFile("CREDENTIALS")
	if err != nil {
		return
	}
	lines := strings.Split(string(credentials), "\n")
	config := &oauth1a.ClientConfig{
		ConsumerKey:    lines[0],
		ConsumerSecret: lines[1],
	}
	user := oauth1a.NewAuthorizedConfig(lines[2], lines[3])
	client = twittergo.NewClient(config, user, "stream.twitter.com")
	return
}

type Args struct {
	Track string
	Lang string
}

func parseArgs() *Args {
	a := &Args{}
	flag.StringVar(&a.Track, "track", "Data Science,Big Data,Machine Learning", "Keyword to look up")
	flag.StringVar(&a.Lang, "lang", "en", "Language to look up")
	flag.Parse()
	return a
}

type TopList struct {
	Articles []ranking.Item
}

func LoadExistingList(filename string) *list.List {
	result := list.New()

	data, err := ioutil.ReadFile(filename)
	if (err == nil) {
		// today's list exists, load existing list
		var input TopList
		js.Unmarshal(data, &input)
		for i := 0; i<len(input.Articles); i++ {
			result.PushBack(input.Articles[i])
		}
	}

	return result
}



type streamConn struct {
	client   *http.Client
	resp     *http.Response
	url      *url.URL
	stale    bool
	closed   bool
	mu       sync.Mutex
	// wait time before trying to reconnect, this will be
	// exponentially moved up until reaching maxWait, when
	// it will exit
	wait    int
	maxWait int
	connect func() (*http.Response, error)
}

func NewStreamConn(max int) streamConn {
	return streamConn{wait: 1, maxWait: max}
}

func (conn *streamConn) Close() {
	// Just mark the connection as stale, and let the connect() handler close after a read
	conn.mu.Lock()
	defer conn.mu.Unlock()
	conn.stale = true
	conn.closed = true
	if conn.resp != nil {
		conn.resp.Body.Close()
	}
}

func (conn *streamConn) isStale() bool {
	conn.mu.Lock()
	r := conn.stale
	conn.mu.Unlock()
	return r
}

func readStream(client *twittergo.Client, sc streamConn, path string, query url.Values, 
				resp *twittergo.APIResponse, handler func([]byte), done chan bool) {

	var reader *bufio.Reader
	reader = bufio.NewReader(resp.Body)

	for {
		//we've been closed
		if sc.isStale() {
			sc.Close()
			fmt.Println("Connection closed, shutting down ")
			break
		}

		line, err := reader.ReadBytes('\n')

		if err != nil {
			if sc.isStale() {
				fmt.Println("conn stale, continue")
				continue
			}

			time.Sleep(time.Second * time.Duration(sc.wait))
			//try reconnecting, but exponentially back off until MaxWait is reached then exit?
			resp, err := Connect(client, path, query)
			if err != nil || resp == nil {
				fmt.Println(" Could not reconnect to source? sleeping and will retry ")
				if sc.wait < sc.maxWait {
					sc.wait = sc.wait * 2
				} else {
					fmt.Println("exiting, max wait reached")
					done <- true
					return
				}
				continue
			}
			if resp.StatusCode != 200 {
				fmt.Printf("resp.StatusCode = %d", resp.StatusCode)
				if sc.wait < sc.maxWait {
					sc.wait = sc.wait * 2
				}
				continue
			}

			reader = bufio.NewReader(resp.Body)
			continue
		} else if sc.wait != 1 {
			sc.wait = 1
		}
		line = bytes.TrimSpace(line)
		fmt.Println("Received a line ")

		if len(line) == 0 {
			continue
		}
		handler(line)
	}
}

func Connect(client *twittergo.Client, path string, query url.Values) (resp *twittergo.APIResponse, err error) {
	var (
		req 	*http.Request
	)

	url := fmt.Sprintf("%v?%v", path, query.Encode())
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		err = fmt.Errorf("Could not parse request: %v\n", err)
		return
	}
	resp, err = client.SendRequest(req)
	if err != nil {
		err = fmt.Errorf("Could not send request: %v\n", err)
		return
	}

	fmt.Printf("resp.StatusCode=%d\n", resp.StatusCode)
	return
}

func ParseTweet(tweet *twittergo.Tweet) ranking.Item {
	item := ranking.Item{}

	rs := tweet.RetweetedStatus()
	vote := 1
	createdAt := tweet.CreatedAt()
	id := tweet.IdStr()
	if (rs != nil) {
		fmt.Printf("retweet_count:        %d\n", rs.RetweetCount())
		fmt.Printf("favorite_count:        %d\n", rs.FavoriteCount())
		vote = int(rs.RetweetCount()+rs.FavoriteCount())

		createdAt = rs.CreatedAt()
		id = rs.IdStr()
	}
				
	if (time.Now().UTC().Sub(createdAt).Hours() < 24.0) {
		e := tweet.Entities()
		if (e != nil) {
			fmt.Printf("url:        %v\n", e.FirstUrl().ExpandedUrl())

			// Form top item
			if (e.FirstUrl().ExpandedUrl()!="") {				
				item.CreatedAt = createdAt
				item.Vote = vote
				item.TweetIds = []string{id}

				// fetch the final url
				resp, err := http.Get(e.FirstUrl().ExpandedUrl())
    		if err == nil {
        	item.Url = resp.Request.URL.String()
        	fmt.Println("final url:	", item.Url)
        	fmt.Println("content-type: ", resp.Header.Get("Content-Type"))

        	if (strings.Contains(resp.Header.Get("Content-Type"), "text/html")) {
						// If it is a webpage
						// article extraction
						g := goose.New()
						article := g.ExtractFromUrl(item.Url)

						fmt.Println("title", article.Title)
			    	fmt.Println("description", article.MetaDescription)
			    	fmt.Println("top image", article.TopImage)

			    	if (article.Title != "") {
			    		//&& (article.MetaDescription != "") {
							item.Title = article.Title
			    		item.Description = article.MetaDescription
			    		item.Image = article.TopImage
			    	}
			    } else {
			    	//Other type
			    	if (rs != nil) {
			    		item.Title = rs.Text()
			    	} else {
							item.Title = tweet.Text()    		
			    	}
			    }
			  }
			}
		}
	}

	return item
}

func filterStream(client *twittergo.Client, path string, query url.Values) (err error) {
	var (
		resp    *twittergo.APIResponse
	)

	sc := NewStreamConn(300)

	// Step 1: connect to twitter public stream endpoint
	resp, err = Connect(client, path, query)

	done := make(chan bool)
	stream := make(chan []byte, 1000)
	go func() {
		startday := time.Now().UTC().Day()
		filename := "../data/toplist-" + 
			time.Now().UTC().Format("2006-01-02") +".json"

		//
		topList := LoadExistingList(filename)
		
		//Cron job to store toplist per hour
		c := cron.New()
		c.AddFunc("0 * * * * *", 
			func() { 
				fmt.Println("cron cron cron cron ............................")

				output := make(map[string]interface{})
				output["articles"] = make([]ranking.Item, 0)

				f, err := os.OpenFile(filename, os.O_RDWR, 0666)
				if (err == nil) {
					err = os.Remove(filename)
				}
				f.Close()

				f, err = os.Create(filename)
				fmt.Printf("[Cron] filename: %v\n", filename)
				if (err == nil) {
					tlist := make([]ranking.Item, 0)
					count := 0
					for e := topList.Front(); e != nil; e = e.Next() {
						fmt.Println("[Cron] Write url into file")
						//f.WriteString(e.Value.(ranking.Item).Url)
						//f.WriteString("\n")
						tlist = append(tlist, e.Value.(ranking.Item))
						count += 1
						if (count >= 20) {
							break
						}
					}
					output["articles"] = tlist

					jsonstr, _ := js.Marshal(output)
					f.WriteString(string(jsonstr))
					f.Sync()
				} else {
					fmt.Println("[Cron] File creation error", err)	
				}
				f.Close()
			})
		c.Start()
		fmt.Println("cron job start")

		//g := goose.New()
		for data := range stream {
			if (time.Now().UTC().Day() != startday) {
				// Clear the top list
				var next *list.Element
				count := 0
				for e := topList.Front(); e != nil; e = next {
					next = e.Next()
					count += 1

					dur := time.Now().UTC().Sub(e.Value.(ranking.Item).CreatedAt)
					if (count<=20) || (dur.Hours()>=24.0) {
						topList.Remove(e)						
					}
				}
				
				startday = time.Now().UTC().Day()
				filename = "../data/toplist-" + 
					time.Now().UTC().Format("2006-01-02") +".json"
			}


			fmt.Println(string(data))
			tweet := &twittergo.Tweet{}
			err := json.Unmarshal(data, tweet)
			if (err == nil) {
				fmt.Printf("ID:                   %v\n", tweet.Id())
				fmt.Printf("User:                 %v\n", tweet.User().ScreenName())
				fmt.Printf("Tweet:                %v\n", tweet.Text())

				item := ParseTweet(tweet)	
				if (item.Title != "") {
					ranking.Insert(topList, item)

					fmt.Println("**********************************")
					for e := topList.Front(); e != nil; e = e.Next() {
						fmt.Printf("%d: %v\n",e.Value.(ranking.Item).Vote, e.Value.(ranking.Item).Url)
					}
				}
			}
		}
	}()

	readStream(client, sc, path, query, resp, 
		func(line []byte) {
			stream <- line
		}, done)


	return
}

func main() {
	var (
		err    error
		args   *Args
		client *twittergo.Client
	)

	args = parseArgs()
	if client, err = LoadCredentials(); err != nil {
		fmt.Printf("Could not parse CREDENTIALS file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(args.Track)
	query := url.Values{}
	query.Set("track", args.Track)
	query.Set("language", args.Lang)

	fmt.Println("Printing everything about data science, big data and machine learning:")
	fmt.Printf("=========================================================\n")
	if err = filterStream(client, "/1.1/statuses/filter.json", query); err != nil {
		fmt.Println("Error: %v\n", err)
	}
	fmt.Printf("\n\n")

}
