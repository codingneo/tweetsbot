package ranking

import (
	"fmt"
	"time"
	"strings"
	"container/list"
)

//const TOPLIST_LEN = 20

type Item struct {
	CreatedAt	time.Time
	Vote int
	Url string
	Title string
	Description string
	Image string
	TweetIds []string
}

func (item Item) Contains(i Item) bool {
	result := true
	for _, id1 := range i.TweetIds {
		matched := false
		for _, id2 := range item.TweetIds {
			if (id1 == id2) {
				matched = true
				break;
			}
		}
		if (!matched) {
			result = false
			break;
		}
	}

	return result
}

func CombineIds(i1 Item, i2 Item) []string {
	result := make([]string, len(i1.TweetIds))
	result = i1.TweetIds

	for _, id := range i2.TweetIds {
		matched := false
		for _, id2 := range i1.TweetIds {
			if (id == id2) {
				matched = true
				break
			}
		}

		if (!matched) {
			result = append(result, id)
		}
	}

	return result
}

func Insert(l *list.List, item Item) {
	fmt.Printf("List len=%d\n", l.Len())
	var elm *list.Element
	for e := l.Front(); e != nil; e = e.Next() {
		if //(strings.Contains(item.Title, e.Value.(Item).Title)) ||
			 //(strings.Contains(e.Value.(Item).Title, item.Title)) || 
			 (item.Url==e.Value.(Item).Url) {
			//if item.Vote<=e.Value.(Item).Vote {
			//	item.Vote = e.Value.(Item).Vote
			//}

			if (e.Value.(Item).Contains(item)) {
				item.Vote = e.Value.(Item).Vote+1	
			} else {
				item.Vote += e.Value.(Item).Vote
				item.TweetIds = CombineIds(e.Value.(Item), item)
			}

			
			if (item.CreatedAt.After(e.Value.(Item).CreatedAt)) {
				item.CreatedAt = e.Value.(Item).CreatedAt				
			}
			elm = e
			break			
		}
	}
	if (elm!=nil) {
		l.Remove(elm)
	}

	elm = nil
	for e := l.Front(); e != nil; e = e.Next() {
		if item.Vote>e.Value.(Item).Vote {
			elm = e
			break
		}
	}

	/*
	if (l.Len()<TOPLIST_LEN) {
		if (elm == nil) {
			l.PushBack(item)
		} else {
			l.InsertBefore(item, elm)
		}
	} else {
		if (elm != nil) {
			l.InsertBefore(item, elm)
			l.Remove(l.Back())
		}
	}
	*/
	if (elm == nil) {
		l.PushBack(item)
	} else {
		l.InsertBefore(item, elm)
	}
}

func main() {
	l := list.New()
	Insert(l, Item{Vote: 3, Url: "http://twitter.com"})
	Insert(l, Item{Vote: 8, Url: "http://google.com"})

	for e := l.Front(); e != nil; e = e.Next() {
		fmt.Println(e.Value.(Item).Url)
	}
}
