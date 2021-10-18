package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	type Output struct {
		Name  string
		Error error
	}

	var (
		maxWorkers = flag.Int("w", 5, "worker count")
		topic      = "resize"
		inchan     = make(chan string, 5)
		outchan    = make(chan Output, 5)
	)

	flag.Parse()

	var (
		natsHost    = flag.Arg(0)
		inputFolder = flag.Arg(1)
	)

	// connect to external nats
	nc, err := nats.Connect(natsHost)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	files, err := os.ReadDir(inputFolder)
	if err != nil {
		panic(err)
	}

	go func() {
		for _, file := range files {
			inchan <- file.Name()
		}
	}()

	for i := 0; i < *maxWorkers; i++ {
		go func() {
			for {
				filename := <-inchan

				if _, err := nc.Request(topic, []byte(filename), 5*time.Second); err == nil {
					// fmt.Printf("%s: %s\n", filename, string(msg.Data))
					outchan <- Output{
						Name:  filename,
						Error: nil,
					}
				} else {
					outchan <- Output{
						Name:  filename,
						Error: err,
					}
				}
			}
		}()
	}

	for i := 0; i < len(files); i++ {
		o := <-outchan
		fmt.Printf("%s: %v\n", o.Name, o.Error)
	}
}
