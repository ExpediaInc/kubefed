package e2e

import (
	"bytes"
	restclient "k8s.io/client-go/rest"
	"log"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"

	"sigs.k8s.io/kubefed/test/e2e/framework"
	"testing"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/client_golang/prometheus/testutil/promlint"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)


type MetricsTestCase struct {
	Target prometheus.Collector
	Want string
	Metrics []string
}

var _ = Describe("Metrics ", func() {
	f := framework.NewKubeFedFramework("metrics")
	tl := framework.NewE2ELogger()

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