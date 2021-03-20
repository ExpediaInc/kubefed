package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	restclient "k8s.io/client-go/rest"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"

	"testing"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/client_golang/prometheus/testutil/promlint"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)


type MetricsTestCase struct {
	Target prometheus.Collector
	Want string
	Metrics []string
}

type portForwarder struct {
	Namespace string
	PodName   string
	ctx       context.Context
	Client    kubernetes.Interface
	Config    *restclient.Config
	Out       io.Writer
	ErrOut    io.Writer
	StopChan  chan struct{}
}

var _ = Describe("Metrics ", func() {
	f := framework.NewKubeFedFramework("metrics")
	tl := framework.NewE2ELogger()

	var err error
	var port uint16
	var pfCancel context.CancelFunc
	var fwCancel context.CancelFunc
	var pfw portForwarder
	var kubeConfig *restclient.Config
	var client genericclient.Client

	BeforeEach(func() {
		if kubeConfig == nil {
			var err error
			kubeConfig = f.KubeConfig()
			client, err = genericclient.New(kubeConfig)
			if err != nil {
				tl.Fatalf("Error initializing dynamic client: %v %v", client, err)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		// TODO use 1 client?
		clientset, _ := kubernetes.NewForConfig(kubeConfig)
		pfw = portForwarder{
			Namespace: "kube-federation-system",
			PodName: "foo",
			ctx: ctx,
			Client: clientset,
			Config: kubeConfig,
			Out: os.Stdout,
			ErrOut: os.Stderr,
			StopChan: make(chan struct{}, 1),
		}
	
		fwCancel = cancel

	})

	JustBeforeEach(func() {
		err := ForwardPorts(&pfw, []string{"127.0.0.1"}, []string{"8080"}, nil )
		if err != nil {
			fmt.Println("doh")
		}
	})

	JustAfterEach(func() {
		if pfCancel != nil {
			pfCancel()
		}
	})

	AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
			fmt.Println("error %s %d", err, port)
		}
	})

	Describe("Metrics", func() {
		It("Metrics", func() {

			if !framework.TestContext.RunControllers() {
				framework.Skipf("UGGGGGHHHHHH.")
			}
		})
	})

})


func (mtc MetricsTestCase) Test() ([]promlint.Problem, error) {
	want := bytes.NewBufferString(mtc.Want)
	if err := testutil.CollectAndCompare(mtc.Target, want, mtc.Metrics...); err != nil {
		return nil, errors.Wrap(err, "output verification failed")
	}
	return nil, nil
}

type MetricsTestCases map[string]MetricsTestCase

func (mtc MetricsTestCases) Test(t *testing.T) {
	for name, tc := range mtc {
		t.Run(name, func(tt *testing.T) {
			problems, err := tc.Test()
			if err != nil {
				tt.Error(err)
			}
			for _, problem := range problems {
				log.Printf("non-standard metric '%s': %s", problem.Metric, problem.Text)
			}
		})
	}
}

func ForwardPorts(f *portForwarder, addresses []string, ports []string, stopChan <-chan struct{}) error {
	
	req := f.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(f.Namespace).
		Name(f.PodName).
		SubResource("portforward")

	ctx, cancel := context.WithCancel(f.ctx)
	transport, upgrader, err := spdy.RoundTripperFor(f.Config)
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	readyChan := make(chan struct{})
	fw, err := portforward.NewOnAddresses(dialer, addresses, ports, ctx.Done(), readyChan, f.Out, f.ErrOut)
	if err != nil {
		return err
	}
	errChan := make(chan error)
	go func() { errChan <- fw.ForwardPorts() }()
	select {
	case <-readyChan:
		return nil
	case err = <-errChan:
		cancel()
		return err
	}
}
