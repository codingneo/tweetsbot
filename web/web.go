package main

import (
	"fmt"
	"time"
	"net/http"
	"io/ioutil"
)

/*
func main() {
  fs := http.FileServer(http.Dir("../data"))
  http.Handle("/", fs)

  fmt.Println("Listening...")
  http.ListenAndServe(":3000", nil)
}
*/

func main() {
  fs := http.FileServer(http.Dir("../static"))
  http.Handle("/", fs)


	http.HandleFunc("/data/", hello)
	fmt.Println("listening...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}


func hello(res http.ResponseWriter, req *http.Request) {
	filename := "../data/toplist-" + 
							time.Now().UTC().Format("2006-01-02") +
							".json"
	data, _ := ioutil.ReadFile(filename)

	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprintln(res, string(data))
}
