package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/nats-io/nats.go"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Input struct {
	Msg *nats.Msg
}

var (
	nc     *nats.Conn
	inchan = make(chan Input, 1)
)

func main() {
	var (
		maxWorkers = flag.Int("w", getCoreCount(), "worker count")
		topic      = "resize"

		err error
	)

	flag.Parse()

	var (
		natsHost     = flag.Arg(0)
		inputFolder  = flag.Arg(1)
		outputFolder = flag.Arg(2)
	)

	// connect to external nats
	nc, err = nats.Connect(natsHost)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	os.MkdirAll(outputFolder, 0755)
	nc.QueueSubscribe(topic, "resize", resize)

	for i := 0; i < *maxWorkers; i++ {
		go func() {
			for {
				in := <-inchan
				filename := string(in.Msg.Data)
				err := ffmpeg.Input(filepath.Join(inputFolder, filename)).
					Filter("scale", ffmpeg.Args{"480:-1"}).
					Output(filepath.Join(outputFolder, filename)).
					OverWriteOutput().
					Run()
				if err != nil {
					fmt.Printf("error: %s: %s\n", filename, err.Error())
				}
				nc.Publish(in.Msg.Reply, []byte("done"))
			}
		}()
	}

	select {}
}

func getCoreCount() int {
	b, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		panic(err)
	}
	reg := regexp.MustCompile(`processor`)
	return len(reg.FindAll(b, -1))
}

func resize(msg *nats.Msg) {
	fmt.Printf("got message\n")
	inchan <- Input{
		msg,
	}
}
