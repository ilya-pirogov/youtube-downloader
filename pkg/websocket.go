package pkg

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/xfrr/goffmpeg/transcoder"
	"log"
	"net/http"
	"time"
)

var (
	trans      = new(transcoder.Transcoder)
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	upgrader   = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func (d *dispatcher) ServeWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	upd := make(chan State)
	d.connect <- upd
	defer func() {
		close(upd)
		d.disconnect <- upd
	}()

	go writer(ws, upd)
	reader(ws)
}

func writer(ws *websocket.Conn, upd <-chan State) {
	pingTicker := time.NewTicker(pingPeriod)

	for {
		select {
		case state := <-upd:
			p, err := json.Marshal(state)
			if err != nil {
				log.Println(err)
				continue
			}
			err = ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, p); err != nil {
				log.Println(err)
				return
			}
		case <-pingTicker.C:
			err := ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Println(err)
				return
			}

			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println(err)
				return
			}
		}

	}
}

func reader(ws *websocket.Conn) {
	defer func() {
		if err := ws.Close(); err != nil {
			log.Println(err)
		}
	}()

	ws.SetReadLimit(512)
	if err := ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}

	ws.SetPongHandler(func(string) error {
		if err := ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Println(err)
		}
		return nil
	})

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}
