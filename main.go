package main

import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

// const fileName = "/tmp/messages"
const fileName = "/tmp/smallfile"

var counterFileName = "/tmp/counter"

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
	sync.RWMutex
}

func (h *webHandler) writeMessage(msg string) error {
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(msg + "\n"); err != nil {
		return err
	}
	return nil

}

func (h *webHandler) ReadCounterValue() (count int, err error) {
	counterContents, err := ioutil.ReadFile(counterFileName)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	count, _ = strconv.Atoi(strings.TrimSpace(string(counterContents)))
	return count, nil
}

func (h *webHandler) incrementCounter() error {
	count, err := h.ReadCounterValue()
	if err != nil {
		return err
	}
	contents := fmt.Sprintf("%d\n", count+1)
	// writing contents to a temp file for situation where you run out of disk space
	if err := ioutil.WriteFile(counterFileName+".tmp", []byte(contents), 0666); err != nil {
		return err
	}
	// renaming is an atomic operation so it will either rename it or just fail and not do anything
	return os.Rename(counterFileName+".tmp", counterFileName)

}

func (h *webHandler) helloWorld(w http.ResponseWriter, req *http.Request) {
	h.RLock()
	defer h.RUnlock()
	contents, err := ioutil.ReadFile(fileName)
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	count, err := h.ReadCounterValue()
	messages := strings.Split(strings.TrimSpace(string(contents)), "\n")
	fmt.Println(messages)
	fmt.Println(count)

	indexTPL.Execute(w, &IndexData{
		Count:    count,
		Messages: messages,
	})
}

func (h *webHandler) PostMessage(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	msg := strings.TrimSpace(req.Form.Get("message"))
	fmt.Println(len(msg))
	fmt.Println("msg is ", msg)
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
	h.Lock()
	defer h.Unlock()

	if err := h.writeMessage(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// increment counter
	if err := h.incrementCounter(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusFound)

}

// Making concurrent requests
// ab -c 100 -n 100000 'http://localhost:80/post-message?message=satya'
func main() {
	h := &webHandler{}
	http.Handle("/", http.HandlerFunc(h.helloWorld))
	http.Handle("/post-message", http.HandlerFunc(h.PostMessage))
	log.Fatal(http.ListenAndServe(":80", nil))
}
