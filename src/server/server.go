package server

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	log "github.com/schollz/logger"
	"gopkg.in/russross/blackfriday.v1"
)

// stream contains the reader and the channel to signify its read
type stream struct {
	reader io.ReadCloser
	done   chan struct{}
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

		if r.URL.Path == "/" {
			// serve the README
			b, _ := ioutil.ReadFile("README.md")
			b = blackfriday.MarkdownCommon(b)
			w.Write(b)
			return
		}

		mutex.Lock()
		if _, ok := channels[r.URL.Path]; !ok {
			channels[r.URL.Path] = make(chan stream)
		}
		channel := channels[r.URL.Path]
		mutex.Unlock()

		queries, ok := r.URL.Query()["pubsub"]
		pubsub := (ok && queries[0] == "true")
		log.Debugf("pubsub: %+v", pubsub)

		method := r.Method
		queries, ok = r.URL.Query()["body"]
		var bodyString string
		if ok {
			bodyString = queries[0]
			if bodyString != "" {
				method = "POST"
			}
		}

		log.Debug(channel)
		if method == "GET" {
			select {
			case stream := <-channel:
				io.Copy(w, stream.reader)
				close(stream.done)
			case <-r.Context().Done():
				log.Debug("consumer canceled")
			}
		} else if method == "POST" {
			var buf []byte
			if bodyString != "" {
				buf = []byte(bodyString)
			} else {
				buf, _ = ioutil.ReadAll(r.Body)
			}

			if !pubsub {
				log.Debug("no pubsub POST")
				doneSignal := make(chan struct{})
				stream := stream{reader: ioutil.NopCloser(bytes.NewBuffer(buf)), done: doneSignal}
				select {
				case channel <- stream:
					log.Debug("connected to consumer")
				case <-r.Context().Done():
					log.Debug("producer canceled")
					doneSignal <- struct{}{}
				}
				log.Debug("waiting for done")
				<-doneSignal
			} else {
				defer func() {
					log.Debug("finished pubsub")
				}()
				log.Debug("using pubsub")
				finished := false
				for {
					if finished {
						break
					}
					doneSignal := make(chan struct{})
					stream := stream{reader: ioutil.NopCloser(bytes.NewBuffer(buf)), done: doneSignal}
					select {
					case channel <- stream:
						log.Debug("connected to consumer")
					case <-r.Context().Done():
						log.Debug("producer canceled")
					default:
						log.Debug("no one connected")
						close(doneSignal)
						finished = true
					}
					<-doneSignal
				}
			}
		}
	}

	log.Infof("running on port %s", flagPort)
	err = http.ListenAndServe(":"+flagPort, http.HandlerFunc(handler))
	if err != nil {
		log.Error(err)
	}
	return
}
