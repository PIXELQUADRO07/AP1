package websocket

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func CredentialStream(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	logPath := "../system/runtime/portal_credentials.log"

	// Open the file
	file, err := os.Open(logPath)
	var lastOffset int64 = 0
	if err == nil {
		info, err := file.Stat()
		if err == nil {
			lastOffset = info.Size()
		}
		file.Close()
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		file, err := os.Open(logPath)
		if err != nil {
			continue
		}

		info, err := file.Stat()
		if err != nil {
			file.Close()
			continue
		}

		if info.Size() > lastOffset {
			_, err = file.Seek(lastOffset, io.SeekStart)
			if err != nil {
				file.Close()
				continue
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
					file.Close()
					return
				}
			}
			lastOffset = info.Size()
		}
		file.Close()
	}
}
