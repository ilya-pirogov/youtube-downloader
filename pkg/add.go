package pkg

import (
	"net/http"
	"net/url"
)

func (d *dispatcher) AddToDownload(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := request.ParseForm()
	if err != nil {
		_, _ = writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	uri, err := url.Parse(request.Form.Get("url"))
	if err != nil {
		_, _ = writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	d.newTask <- *uri
	_, _ = writer.Write([]byte(`<html><head><script>window.location = "/";</script></head></html>`))
}

