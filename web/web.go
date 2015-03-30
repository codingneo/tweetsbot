package main

import (
	"fmt"
	"time"
	"path/filepath"
	"net/http"
	"html/template"
	"io/ioutil"
)

type Archive struct {
	Name string
	FileName string
}

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
	http.HandleFunc("/archives/", archives)
	http.HandleFunc("/archive", viewarchive)
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

func viewarchive(res http.ResponseWriter, req *http.Request) {
	filename := "../data/" + req.URL.Query().Get("filename")
        data, _ := ioutil.ReadFile(filename)

        res.Header().Set("Content-Type", "application/json")
        res.Header().Set("Access-Control-Allow-Origin", "*")
        fmt.Fprintln(res, string(data))
}


func archives(res http.ResponseWriter, req *http.Request) {
	dirname := "../data/"
	files, _ := ioutil.ReadDir(dirname)

	archives := []Archive{}	

	for _,file := range files {
		ext := filepath.Ext(file.Name())
		name := file.Name()
		archives = append(archives, Archive{name[8:len(name)-len(ext)], name})
	}

	fmt.Println(len(files))

	t, _ := template.ParseFiles("../static/archives.html")
	t.Execute(res, archives)	
}
