package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type AddRequest struct {
	Url string `json:"url"`
}

func (d *dispatcher) AddToDownload(writer http.ResponseWriter, request *http.Request) {
	var (
		body []byte
		req  AddRequest
		err  error
		uri  *url.URL
	)
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if body, err = ioutil.ReadAll(request.Body); err != nil {
		_, _ = writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		_, _ = writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	if uri, err = url.Parse(req.Url); err != nil {
		_, _ = writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	w := &worker{&State{
		Url:    uri.String(),
		Status: "Starting",
	}, *uri, "", "", ""}

	if err := w.fetchMeta(); err != nil {
		log.Printf("Unable to add %s: %s", uri.String(), err)
		_, _ = writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	d.lock.RLock()
	for _, ww := range d.pool {
		if ww.Vid.ID == w.Vid.ID {
			d.lock.RUnlock()

			err := fmt.Errorf("video %s is already downloaded", w.Vid.ID)
			log.Println(err)

			_, _ = writer.Write([]byte(err.Error()))
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	d.lock.RUnlock()

	d.newTask <- w

	writer.Header().Add("Content-Type", "text/json")
	_, _ = writer.Write([]byte("{\"success\": true}"))

}
