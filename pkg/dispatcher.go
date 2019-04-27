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
	uri          url.URL
	baseFileName string
	mp4File      string
	mp3File      string
}

type dispatcher struct {
	newTask    chan *worker
	connect    chan chan State
	disconnect chan chan State
	update     chan State
	converter  chan *worker
	pool       []*worker
	lock       sync.RWMutex
}

func NewDispatcher() *dispatcher {
	return &dispatcher{
		newTask:    make(chan *worker),
		connect:    make(chan chan State),
		disconnect: make(chan chan State),
		update:     make(chan State, 0),
		pool:       make([]*worker, 0),
		converter:  make(chan *worker, 16),
	}
}

func (d *dispatcher) Downloading() {
	clients := make([]chan State, 0)

	for {
		select {
		case w := <-d.newTask:
			d.lock.Lock()
			d.pool = append(d.pool, w)
			d.lock.Unlock()

			log.Printf("Added: %s (%s)", w.Url, w.Vid.Title)
			go func() {
				file, _ := os.Create(w.mp4File)

				if err := w.Download(d.update, file); err != nil {
					log.Println(err)
					w.Error = err.Error()
					d.update <- *w.State
					return
				}

				w.Status = "Scheduled"
				w.Percent = -1
				d.update <- *w.State

				if err := file.Close(); err != nil {
					log.Println(err)
				}

				d.converter <- w
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

	w.baseFileName = sanitizeFilename(w.Vid.Title)

	w.mp4File = path.Join("out", w.baseFileName+".mp4")
	w.mp3File = path.Join("out", w.baseFileName+".mp3")
	return nil
}

func (w *worker) Download(q chan State, fp *os.File) error {
	var (
		downUrl *url.URL
		resp    *http.Response
		err     error
		wg      sync.WaitGroup
	)

	q <- *w.State

	if downUrl, err = w.Vid.GetDownloadURL(w.Vid.Formats[0]); err != nil {
		return err
	}

	if resp, err = http.Head(downUrl.String()); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	w.DownloadSize = int64(size)

	r := progress.NewWriter(fp)

	wg.Add(1)
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

	if err = w.Vid.Download(w.Vid.Formats[0], r); err != nil {
		return err
	}

	wg.Wait()
	return nil
}
