package gosse

import (
	"fmt"
	"net/http"
)

type GoSSE struct {
	clients  map[chan string]bool
	add      chan chan string
	remove   chan chan string
	messages chan string
}

func NewSSE() *GoSSE {
	sse := &GoSSE{
		make(map[chan string]bool),
		make(chan (chan string)),
		make(chan (chan string)),
		make(chan string, 1),
	}

	go func() {
		for {
			select {
			case c := <-sse.add:
				sse.clients[c] = true
			case c := <-sse.remove:
				delete(sse.clients, c)
				close(c)
			case msg := <-sse.messages:
				for c := range sse.clients {
					c <- msg
				}
			}
		}
	}()

	return sse
}

func (sse *GoSSE) Publish(msg string) {
	sse.messages <- msg
}

func (sse *GoSSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	message := make(chan string, 2)

	sse.add <- message

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	notify := w.(http.CloseNotifier).CloseNotify()

	for {
		select {
		case <-notify:
			sse.remove <- message
			break
		case msg := <-message:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			f.Flush()
		}
	}
}
