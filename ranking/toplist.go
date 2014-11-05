package ranking

import (
	"container/list"
	"fmt"
)

const TOPLIST_LEN = 20

type Item struct {
	Vote int
	Url string
}

func Insert(l *list.List, item Item) {
	var elm *list.Element
	for e := l.Front(); e != nil; e = e.Next() {
		if item.Url==e.Value.(Item).Url {
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

	if (l.Len()<TOPLIST_LEN) {
		if (elm == nil) {
			l.PushBack(item)
		} else {
			l.InsertBefore(item, elm)
		}
	} else {
		l.Remove(l.Back())
		if (elm == nil) {
			l.PushBack(item)
		} else {
			l.InsertBefore(item, elm)
		}
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
