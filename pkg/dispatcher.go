package pkg

import (
	"context"
	"fmt"
	"github.com/machinebox/progress"
	"github.com/rylio/ytdl"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

type worker struct {
	*State
	uri url.URL
}

type dispatcher struct {
	newTask    chan url.URL
	connect    chan chan State
	disconnect chan chan State
	update     chan State
	pool       []*worker
	lock       sync.RWMutex
}

func NewDispatcher() *dispatcher {
	return &dispatcher{
		newTask:    make(chan url.URL),
		connect:    make(chan chan State),
		disconnect: make(chan chan State),
		update:     make(chan State, 0),
		pool:       make([]*worker, 0),
	}
}

func (d *dispatcher) Start() {
	clients := make([]chan State, 0)

	for {
		select {
		case uri := <-d.newTask:
			w := &worker{&State{
				Url:    uri.String(),
				Status: "Starting",
			}, uri}

			go func() {
				if err := w.fetchMeta(); err != nil {
					log.Println(err)
					log.Printf("Unable to add %s", uri.String())
					return
				}

				d.lock.RLock()
				for _, ww := range d.pool {
					if ww.Vid.ID == w.Vid.ID {
						log.Printf("Video %s is already downloaded", w.Vid.ID)
						d.lock.RUnlock()
						return
					}
				}
				d.lock.RUnlock()

				d.lock.Lock()
				d.pool = append(d.pool, w)
				d.lock.Unlock()

				log.Printf("Added: %s", uri.String())
				go w.Start(d.update)
			}()

		case upd := <-d.connect:
			clients = append(clients, upd)
			log.Printf("Connect: %d", len(clients))
		case upd := <-d.disconnect:
			for k, v := range clients {
				if v == upd {
					clients = append(clients[:k], clients[k+1:]...)
					continue
				}
			}
			log.Printf("Disconnect: %d", len(clients))
		case state := <-d.update:
			for _, q := range clients {
				q <- state
			}
		}
	}
}

func (w *worker) fetchMeta() error {
	w.Status = "Receiving info"
	vid, err := ytdl.GetVideoInfo(w.uri.String())
	if err != nil {
		return err
	}
	w.Vid = vid
	return nil
}

func (w *worker) Start(q chan State) {
	q <- *w.State

	downUrl, err := w.Vid.GetDownloadURL(w.Vid.Formats[0])
	if err != nil {
		log.Println(err)
		w.Error = err.Error()
		q <- *w.State
		return
	}

	resp, err := http.Head(downUrl.String())
	if err != nil {
		log.Println(err)
		w.Error = err.Error()
		q <- *w.State
		return
	}

	if resp.StatusCode != http.StatusOK {
		w.Error = fmt.Sprintf("Status code: %d", resp.StatusCode)
		q <- *w.State
		return
	}

	// the Header "Content-Length" will let us know
	// the total file size to download
	size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	w.DownloadSize = int64(size)

	mp4File := path.Join("out", w.Vid.Title+".mp4")
	mp3File := path.Join("out", w.Vid.Title+".mp3")

	file, _ := os.Create(mp4File)
	defer func() {
		if err := file.Close(); err != nil {
			log.Println(err)
		}

		if err := os.Remove(mp4File); err != nil {
			log.Println(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	r := progress.NewWriter(file)
	go func() {
		ctx := context.Background()
		progressChan := progress.NewTicker(ctx, r, w.DownloadSize, 100*time.Millisecond)
		for p := range progressChan {
			w.Remaining = p.Remaining().Round(time.Second)
			w.Percent = p.Percent()

			q <- *w.State
		}
		wg.Done()
	}()

	w.Status = "Downloading"
	q <- *w.State

	err = w.Vid.Download(w.Vid.Formats[0], r)
	if err != nil {
		log.Println(err)
		w.Error = err.Error()
		return
	}

	wg.Wait()
	w.Status = "Processing"
	w.Percent = 0
	q <- *w.State

	err = trans.Initialize(mp4File, mp3File)
	if err != nil {
		log.Println(err)
		w.Error = err.Error()
		q <- *w.State
	}

	done := trans.Run(false)
	pr := trans.Output()
	go func() {
		for msg := range pr {
			w.Error = msg.FramesProcessed
			q <- *w.State
		}
	}()

	err = <-done
	if err != nil {
		log.Println(err)
		w.Error = err.Error()
		q <- *w.State
	}

	w.DownloadUrl = "/result/" + w.Vid.Title + ".mp3"
	w.Status = "Done"
	q <- *w.State
}
