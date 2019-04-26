package pkg

import (
	"encoding/json"
	"github.com/rylio/ytdl"
	"net/http"
	"time"
)

type State struct {
	Url          string          `json:"url"`
	DownloadUrl  string          `json:"downloadUrl"`
	Status       string          `json:"status"`
	Error        string          `json:"error"`
	Vid          *ytdl.VideoInfo `json:"vid"`
	DownloadSize int64           `json:"downloadSize"`
	Remaining    time.Duration   `json:"remaining"`
	Percent      float64         `json:"percent"`
}

func (d *dispatcher) GetProgress(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/json")

	d.lock.RLock()
	res, err := json.Marshal(d.pool)
	d.lock.RUnlock()

	if err != nil {
		_, _ = writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	_, _ = writer.Write(res)
}
