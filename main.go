package main

import (
	"bytes"
	"flag"
	"image/png"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
)

var display int
var addr string

func init() {
	flag.StringVar(&addr, "addr", "localhost:8080", "server address")
	flag.IntVar(&display, "display", 0, "number of the display to stream")
}

func main() {
	flag.Parse()
	interrupt := make(chan os.Signal, 1)
	//signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: addr, Path: "/source"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	n := screenshot.NumActiveDisplays()
	if display >= n {
		log.Fatalf("Display devices greater than that available. Enter value between [0-%d]", n)
	}

	sendPngBytes(c, display, interrupt)
}

func sendPngBytes(conn *websocket.Conn, displayCount int, interrupt chan os.Signal) {
	bounds := screenshot.GetDisplayBounds(display)
	buf := new(bytes.Buffer)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:
			img, err := screenshot.CaptureRect(bounds)
			if err != nil {
				panic(err)
			}
			png.Encode(buf, img)
			bytesToSend := buf.Bytes()
			log.Printf("Start sending  stream. Size: %d Kb\n", len(bytesToSend)/1000)
			err = conn.WriteMessage(websocket.BinaryMessage, bytesToSend)
			log.Println("Done sending  stream.")
			if err != nil {
				log.Println("write:", err)
				return
			}
			buf.Reset()
		case <-interrupt:
			log.Println("Terminating stream..")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-time.After(time.Second):
			}
			return
		}
	}
}
