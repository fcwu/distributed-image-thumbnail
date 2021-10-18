package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: resize-image <input folder> <output folder>\n")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	inputFolder := flag.Arg(0)
	outputFolder := flag.Arg(1)

	os.MkdirAll(outputFolder, 0755)
	files, err := os.ReadDir(inputFolder)
	if err != nil {
		panic(err)
	}

	type Output struct {
		Name  string
		Error error
	}

	inchan := make(chan string, 5)
	outchan := make(chan Output, 5)

	go func() {
		for _, file := range files {
			inchan <- file.Name()
		}
	}()

	for i := 0; i < getCoreCount(); i++ {
		go func() {
			for {
				filename := <-inchan
				err := ffmpeg.Input(filepath.Join(inputFolder, filename)).
					Filter("scale", ffmpeg.Args{"480:-1"}).
					Output(filepath.Join(outputFolder, filename)).
					OverWriteOutput().
					Run()
				if err != nil {
					fmt.Printf("error: %s: %s\n", filename, err.Error())
				}

				outchan <- Output{
					Name:  filename,
					Error: err,
				}
			}
		}()
	}

	for i := 0; i < len(files); i++ {
		_ = <-outchan
	}
}

func getCoreCount() int {
	b, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		panic(err)
	}
	reg := regexp.MustCompile(`processor`)
	return len(reg.FindAll(b, -1))
}
