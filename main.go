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
)

const fileName = "/tmp/messages"

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
	f, err := os.OpenFile(counterFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%d\n", count+1); err != nil {
		return err
	}
	return nil

}

func (h *webHandler) helloWorld(w http.ResponseWriter, req *http.Request) {
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

func main() {
	h := &webHandler{}
	http.Handle("/", http.HandlerFunc(h.helloWorld))
	http.Handle("/post-message", http.HandlerFunc(h.PostMessage))
	log.Fatal(http.ListenAndServe(":80", nil))
}
