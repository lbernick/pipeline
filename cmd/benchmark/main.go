package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/test/performance"
	"sigs.k8s.io/yaml"
)

var (
	yamlfilename  string
	n             int
	outfile       string
	reportTimeout time.Duration
)

func main() {
	flag.StringVar(&yamlfilename, "f", "", "path to file containing PipelineRun yaml")
	flag.StringVar(&outfile, "o", "output.txt", "path where output should be written")
	flag.IntVar(&n, "n", 5, "number of PipelineRuns to create")
	flag.DurationVar(&reportTimeout, "t", 10*time.Minute, "timeout for reporting on PipelineRuns")
	flag.Parse()

	waitInterval := time.Second // Time to wait in between creating PipelineRuns
	var pr v1.PipelineRun
	MustLoadYAML(yamlfilename, &pr)
	ctx := context.Background()

	cancel := make(chan struct{}, 1)
	go func() {
		cancelAfterTimeoutOrInterrupt(cancel, reportTimeout)
	}()

	performance.CreateAndReportPipelineRuns(ctx, pr, n, outfile, waitInterval, cancel)
}

// MustLoadYAML loads yaml or panics if the parse fails.
func MustLoadYAML(file string, obj interface{}) {
	err := loadYAML(file, obj)
	if err != nil {
		log.Fatalf("load yaml file failed, file path: %s, err: %s", file, err)
	}
}

// loadYAML loads yaml from file to obj.
func loadYAML(file string, obj interface{}) (err error) {
	var data []byte
	if data, err = os.ReadFile(file); err != nil {
		return
	}
	err = yaml.Unmarshal(data, obj)
	return
}

func cancelAfterTimeoutOrInterrupt(c chan<- struct{}, timeout time.Duration) {
	defer close(c)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
	case <-time.After(timeout):
	}
}
