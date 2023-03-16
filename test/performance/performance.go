package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	knativetest "knative.dev/pkg/test"
)

// TODO: Use consistent logging

const (
	qps                 = 1
	burst               = qps
	pollIntervalSeconds = 2
)

var cancelPipelineRunPatchBytes []byte

type perfTest struct {
	kubeClient            *kubernetes.Clientset
	rateLimitedKubeClient *kubernetes.Clientset
	tknClient             *versioned.Clientset
	rateLimitedTknClient  *versioned.Clientset
	pipelineRun           v1.PipelineRun
}

type pipelineRunData struct {
	Name           string
	StartTime      metav1.Time
	CompletionTime metav1.Time
	TaskRuns       []*taskRunData
}

type taskRunData struct {
	Name              string
	StartTime         metav1.Time
	CompletionTime    metav1.Time
	Node              string
	PodCreationTime   metav1.Time
	PodStartTime      metav1.Time
	PodCompletionTime metav1.Time
}

func init() {
	var err error
	cancelPipelineRunPatchBytes, err = json.Marshal([]jsonpatch.JsonPatchOperation{
		{
			Operation: "add",
			Path:      "/spec/status",
			Value:     v1.PipelineRunSpecStatusCancelled,
		}})
	if err != nil {
		log.Fatalf("failed to marshal PipelineRun cancel patch bytes: %v", err)
	}
}

func setUp() perfTest {
	cfg := clientConfig()
	kubeClient, cs := clientSets(cfg)

	rateLimitedCfg := clientConfig()
	rateLimitedCfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(qps, burst)
	rateLimitedKubeClient, rateLimitedCS := clientSets(rateLimitedCfg)
	return perfTest{
		kubeClient: kubeClient, rateLimitedKubeClient: rateLimitedKubeClient,
		tknClient: cs, rateLimitedTknClient: rateLimitedCS,
	}
}

func clientSets(cfg *rest.Config) (*kubernetes.Clientset, *versioned.Clientset) {
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create kubeclient from config file at %s: %s", knativetest.Flags.Kubeconfig, err)
	}

	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create pipeline clientset from config file at %s: %s", knativetest.Flags.Kubeconfig, err)
	}
	return kubeClient, cs
}

func clientConfig() *rest.Config {
	cfg, err := knativetest.BuildClientConfig(knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster)
	if err != nil {
		log.Fatalf("failed to create configuration obj from %s for cluster %s: %s", knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster, err)
	}
	return cfg
}

// CreateAndReportPipelineRuns creates n instances of the input PipelineRun, waiting a duration equal to waitInterval
// between each one. The input PipelineRun must set "GenerateName". It waits until all PipelineRuns are complete
// and writes timing data on the PipelineRuns to outfile.
// If data is received on done, cancels any running PipelineRuns and reports on them in their incomplete state.
func CreateAndReportPipelineRuns(ctx context.Context, pr v1.PipelineRun, n int, outfile string, waitInterval time.Duration, done <-chan struct{}) {
	pt := setUp()
	if pr.GenerateName == "" {
		log.Fatalf("must use GenerateName")
	}
	if pr.Namespace == "" {
		pr.Namespace = "default"
	}
	pt.pipelineRun = pr
	names := make(chan string, n)
	data := make(chan *pipelineRunData, n)

	go func() {
		createPipelineRuns(ctx, pt, n, names, waitInterval)
	}()

	go func() {
		reportOnPipelineRuns(ctx, pt, names, data, done)
	}()
	writeToFile(outfile, data)
}

// createPipelineRuns creates n PipelineRuns at a rate of one per second
// based on the PipelineRun spec in the perfTest.
// It writes the names of the generated PipelineRuns to the names channel and closes names when complete.
func createPipelineRuns(ctx context.Context, pt perfTest, n int, names chan<- string, waitInterval time.Duration) {
	for i := 0; i < n; i++ {
		pr, err := pt.rateLimitedTknClient.TektonV1().PipelineRuns(pt.pipelineRun.Namespace).Create(ctx, &pt.pipelineRun, metav1.CreateOptions{})
		if err == nil {
			logrus.Infof("pr created: %s", pr.Name)
			names <- pr.Name
		} else {
			logrus.Error(err)
		}
		time.Sleep(waitInterval)
	}
	close(names)
}

// reportOnPipelineRuns reads PipelineRun names from the names channel, polls them until they are done or until data is read from done,
// and writes timing data on the PipelineRuns to the data channel. Closes the data channel after reporting on all PipelineRuns.
func reportOnPipelineRuns(ctx context.Context, pt perfTest, names <-chan string, data chan<- *pipelineRunData, done <-chan struct{}) {
	var wg sync.WaitGroup
	for name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			d, err := reportOnPipelineRun(ctx, pt, n, done)
			if err != nil {
				logrus.Error(err)
			}
			data <- d
		}(name)
	}

	go func() {
		defer close(data)
		wg.Wait()
	}()
}

// writeToFile creates or truncates the file specified by the input filename
// and writes serialized data from the channel to the file.
func writeToFile(filename string, data <-chan *pipelineRunData) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	for d := range data {
		data, err := json.Marshal(d)
		if err != nil {
			log.Fatal(err)
		}
		file.Write(data)
		file.WriteString("\n")
	}
}

// reportOnPipelineRun polls the PipelineRun until it's complete or the context is complete, and returns
// data representing all of its child TaskRuns (CustomRuns are ignored).
// If timeout is not nil, will stop polling the PipelineRun after timeout and report on the PipelineRun in its incomplete state.
func reportOnPipelineRun(ctx context.Context, pt perfTest, name string, done <-chan struct{}) (*pipelineRunData, error) {
	pr, err := pollPipelineRunUntilComplete(ctx, pt, name, done)
	if err != nil {
		logrus.Error(err)
	}

	prd := pipelineRunData{
		Name: pr.Name,
	}

	if pr.HasStarted() {
		prd.StartTime = *pr.Status.StartTime
	}
	if pr.IsDone() {
		prd.CompletionTime = *pr.Status.CompletionTime
	}

	var merr *multierror.Error
	for _, cr := range pr.Status.ChildReferences {
		if cr.Kind != "TaskRun" {
			continue
		}
		trd, err := reportOnTaskRun(ctx, pt, cr.Name)
		if err != nil {
			merr = multierror.Append(merr, err)
		}
		prd.TaskRuns = append(prd.TaskRuns, trd)
	}
	return &prd, merr.ErrorOrNil()
}

// pollPipelineRunUntilComplete returns nil when the named PipelineRun is complete.
// Returns an error if the PipelineRun could not be retrieved.
// If timeout elapses before the PipelineRun is complete, cancels the PipelineRun.
func pollPipelineRunUntilComplete(ctx context.Context, pt perfTest, name string, done <-chan struct{}) (*v1.PipelineRun, error) {
	var pr *v1.PipelineRun
	for {
		var err error
		pr, err = pt.tknClient.TektonV1().PipelineRuns(pt.pipelineRun.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return pr, err
		}
		if pr.IsDone() {
			logrus.Infof("pr %s is done", pr.Name)
			return pr, nil
		}
		sleeper := time.After(pollIntervalSeconds * time.Second)
		select {
		case <-done:
			logrus.Infof("stopped polling due to cancelation. cancelling and reporting on incomplete PipelineRun %s", pr.Name)
			err := cancelPipelineRun(ctx, pt, name)
			return pr, err
		case <-sleeper:
		}
	}
}

// reportOnTaskRun returns data related to the timing of a TaskRun.
func reportOnTaskRun(ctx context.Context, pt perfTest, name string) (*taskRunData, error) {
	tr, err := pt.tknClient.TektonV1().TaskRuns(pt.pipelineRun.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	trd := taskRunData{
		Name: name,
	}
	if tr.HasStarted() {
		trd.StartTime = *tr.Status.StartTime
	}
	if tr.Status.PodName != "" {
		pod, err := pt.kubeClient.CoreV1().Pods(pt.pipelineRun.Namespace).Get(ctx, tr.Status.PodName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		trd.Node = pod.Spec.NodeName
		trd.PodCreationTime = pod.CreationTimestamp
		if pod.Status.StartTime != nil {
			trd.PodStartTime = *pod.Status.StartTime
		}
		// TODO: Not sure of the best way to get the pod completion time,
		// since "Ready" status "false" can happen before init containers complete or after the taskrun has completed
		// but before Tekton fully shuts down the pod (e.g. stopping sidecars)
	}
	if tr.IsDone() {
		trd.CompletionTime = *tr.Status.CompletionTime
	}
	return &trd, nil
}

func cancelPipelineRun(ctx context.Context, pt perfTest, name string) error {
	_, err := pt.tknClient.TektonV1().PipelineRuns(pt.pipelineRun.Namespace).Patch(ctx, name, types.JSONPatchType, cancelPipelineRunPatchBytes, metav1.PatchOptions{})
	if errors.IsNotFound(err) {
		// The PipelineRun may have been deleted in the meantime
		return nil
	} else if err != nil {
		return fmt.Errorf("error canceling PipelineRun %s: %s", name, err)
	}
	return nil
}
