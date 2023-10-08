package main

import (
	_ "embed"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const fileName = "/tmp/messages"

//go:embed index.html
var IndexHTML string

type webHandler struct {
}

func (h *webHandler) helloWorld(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, IndexHTML)
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
