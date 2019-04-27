package pkg

import (
	"log"
	"os"
)

func (d *dispatcher) Converting() {
	for w := range d.converter {
		w.Status = "Processing"
		w.Percent = 0
		d.update <- *w.State

		if err := trans.Initialize(w.mp4File, w.mp3File); err != nil {
			log.Println(err)
			w.Error = err.Error()
			d.update <- *w.State
			continue
		}

		done := trans.Run(false)

		if err := <-done; err != nil {
			log.Println(err)
			w.Error = err.Error()
			d.update <- *w.State
			continue
		}

		w.DownloadUrl = "/result/" + w.baseFileName + ".mp3"
		w.Status = "Done"
		d.update <- *w.State

		if err := os.Remove(w.mp4File); err != nil {
			log.Println(err)
		}
	}
}
