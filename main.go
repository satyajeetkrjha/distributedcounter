package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const fileName = "/tmp/messages"

// this is max amount of bytes for unix os
const maxAtomicAppendSize = 512

//go:embed index.html
var IndexHTML string

var indexTPL = template.Must(template.New("index.html").Parse(IndexHTML))

type IndexData struct {
	Count    int
	Messages []string
}

type webHandler struct {
}

func (h *webHandler) helloWorld(w http.ResponseWriter, req *http.Request) {
	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	messages := strings.Split(strings.TrimSpace(string(contents)), "\n")
	fmt.Println(messages)

	indexTPL.Execute(w, &IndexData{
		Count:    len(messages),
		Messages: messages,
	})
}

func (h *webHandler) PostMessage(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	msg := strings.TrimSpace(req.Form.Get("message"))
	if msg == "" {
		http.Error(w, "No Message", http.StatusBadRequest)
		return
	} else if strings.Contains(msg, "\n") {
		http.Error(w, "Newline Not Allowed", http.StatusBadRequest)
		return
	} else if len(msg)+1 > maxAtomicAppendSize {
		http.Error(w, "Message too long", http.StatusBadRequest)
		return
	}
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(msg + "\n"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, req, "/", http.StatusFound)

}

func main() {
	h := &webHandler{}
	http.Handle("/", http.HandlerFunc(h.helloWorld))
	http.Handle("/post-message", http.HandlerFunc(h.PostMessage))
	log.Fatal(http.ListenAndServe(":80", nil))
}
