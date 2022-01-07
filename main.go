package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	logger = getLogger()
	config = readConfigJsonFile()
	relayConfig = readRelayConfigJsonFile(config.RelayConfigFile)
	f, err := os.Open(config.HostFile)
	if err != nil {
		fmt.Println(err.Error())
	}
	buf := bufio.NewReader(f)
	for {
		data, _, eof := buf.ReadLine()
		if eof == io.EOF {
			break
		}
		fmt.Printf("scaning %s:%d",string(data),25)
		go ScanHost(string(data),25,"")  // Goroutine
	}
}
