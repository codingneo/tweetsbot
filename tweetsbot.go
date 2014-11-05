package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/kurrik/oauth1a"
	"github.com/codingneo/twittergo"
	"github.com/kurrik/json"
	//"io/ioutil"
	"net/http"
	"net/url"
	"os"
	//"strings"
	"sync"
	"time"
)


/*
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
	client = twittergo.NewClient(config, user)
	return
}
*/
func LoadCredentials() (client *twittergo.Client, err error) {
	config := &oauth1a.ClientConfig{
		ConsumerKey:    "IvRak6KY38k6LLCY4qGOgjfGC",
		ConsumerSecret: "z454pOM3VqApBjAaZPtdW0hDNOayqy2AX28hD8OZGWDCQ2fVvW",
	}
	user := oauth1a.NewAuthorizedConfig(
		"2679119462-FezANsujMRbfYJHfTIMKyBI4JiVei9NDwUesNkV", 
		"wTpLvDg4r9u77CpL7nGce0djla9p27HgQ5wyp4dT5DQHk")
	client = twittergo.NewClient(config, user, "stream.twitter.com")
	return
}


type Args struct {
	Track string
	Lang string
}

func parseArgs() *Args {
	a := &Args{}
	flag.StringVar(&a.Track, "track", "Data Science,Big Data", "Keyword to look up")
	flag.StringVar(&a.Lang, "lang", "en", "Language to look up")
	flag.Parse()
	return a
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

func filterStream(client *twittergo.Client, path string, query url.Values) (err error) {
	var (
		resp    *twittergo.APIResponse
	)

	sc := NewStreamConn(300)

	resp, err = Connect(client, path, query)

	done := make(chan bool)
	stream := make(chan []byte, 1000)
	go func() {
		for data := range stream {
			fmt.Println(string(data))
			tweet := &twittergo.Tweet{}
			err := json.Unmarshal(data, tweet)
			if (err == nil) {
					fmt.Printf("ID:                   %v\n", tweet.Id())
					fmt.Printf("User:                 %v\n", tweet.User().ScreenName())
					fmt.Printf("Tweet:                %v\n", tweet.Text())
					rs := tweet.RetweetStatus()
					if (rs != nil) {
						fmt.Printf("retweet_count:        %d\n", rs.RetweetCount())
						fmt.Printf("favorite_count:        %d\n", rs.FavoriteCount())
					}
					e := tweet.Entities()
					if (e != nil) {
						fmt.Printf("url:        %v\n", e.FirstUrl().ExpandedUrl())
					}
			}
		}		
	}()

	readStream(client, sc, path, query, resp, func(line []byte) {
		stream <- line}, done)


	return
}

func main() {
	var (
		err    error
		args   *Args
		client *twittergo.Client
	)

	f, err := os.Create("/tmp/dat")
	defer f.Close()
	n, err := f.WriteString("writes\n")
	fmt.Println("Write %d bytes", n)


	args = parseArgs()
	if client, err = LoadCredentials(); err != nil {
		fmt.Printf("Could not parse CREDENTIALS file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(args.Track)
	query := url.Values{}
	query.Set("track", args.Track)
	query.Set("lang", args.Lang)

	fmt.Println("Printing everything about data science:")
	fmt.Printf("=========================================================\n")
	if err = filterStream(client, "/1.1/statuses/filter.json", query); err != nil {
		fmt.Println("Error: %v\n", err)
	}
	fmt.Printf("\n\n")

}
