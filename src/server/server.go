package server

import (
	"io"
	"net/http"
	"sync"

	log "github.com/schollz/logger"
)

// stream contains the reader and the channel to signify its read
type stream struct {
	reader io.ReadCloser
	done   chan struct{}
	header http.Header
}

// Serve will start the server
func Serve(flagPort string) (err error) {
	channels := make(map[string]chan stream)
	mutex := &sync.Mutex{}

	handler := func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("opened %s %s", r.Method, r.URL.Path)
		defer func() {
			log.Debugf("finished %s\n", r.URL.Path)
		}()

		mutex.Lock()
		if _, ok := channels[r.URL.Path]; !ok {
			channels[r.URL.Path] = make(chan stream)
		}
		channel := channels[r.URL.Path]
		mutex.Unlock()

		log.Debug(channel)
		if r.Method == "GET" {
			select {
			case stream := <-channel:
				flusher, ok := w.(http.Flusher)
				if !ok {
					panic("expected http.ResponseWriter to be an http.Flusher")
				}
				w.Header().Set("Connection", "Keep-Alive")
				w.Header().Set("Transfer-Encoding", "chunked")
				buffer := make([]byte, 1024)
				for {
					n, err := stream.reader.Read(buffer)
					if err != nil {
						log.Debugf("err: %v", err)
						break
					}
					w.Write(buffer[:n])
					flusher.Flush()
				}
				close(stream.done)
			case <-r.Context().Done():
				log.Debug("consumer canceled")
			}
		} else if r.Method == "POST" {
			doneSignal := make(chan struct{})
			stream := stream{reader: r.Body, done: doneSignal, header: r.Header}
			select {
			case channel <- stream:
				log.Debug("connected to consumer")
			case <-r.Context().Done():
				log.Debug("producer canceled")
				doneSignal <- struct{}{}
			}
			log.Debug("waiting for done")
			<-doneSignal
		}
	}

	log.Infof("running on port %s", flagPort)
	err = http.ListenAndServe(":"+flagPort, http.HandlerFunc(handler))
	if err != nil {
		log.Error(err)
	}
	return
}
