package plugin

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/argoproj/argo-rollouts/metricproviders/plugin/rpc"
	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	goPlugin "github.com/hashicorp/go-plugin"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
)

var testHandshake = goPlugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ARGO_ROLLOUTS_RPC_PLUGIN",
	MagicCookieValue: "metrics",
}

// This is just an example of how to test a plugin.
func TestRunSuccessfully(t *testing.T) {
	//Skip test because this is just an example of how to test a plugin.
	t.Skip("Skipping test because it requires a running prometheus server")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logCtx := *log.WithFields(log.Fields{"plugin-test": "prometheus"})

	rpcPluginImp := &RpcPlugin{
		LogCtx: logCtx,
	}

	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]goPlugin.Plugin{
		"RpcMetricsPlugin": &rpc.RpcMetricsPlugin{Impl: rpcPluginImp},
	}

	ch := make(chan *goPlugin.ReattachConfig, 1)
	closeCh := make(chan struct{})
	go goPlugin.Serve(&goPlugin.ServeConfig{
		HandshakeConfig: testHandshake,
		Plugins:         pluginMap,
		Test: &goPlugin.ServeTestConfig{
			Context:          ctx,
			ReattachConfigCh: ch,
			CloseCh:          closeCh,
		},
	})

	// We should get a config
	var config *goPlugin.ReattachConfig
	select {
	case config = <-ch:
	case <-time.After(2000 * time.Millisecond):
		t.Fatal("should've received reattach")
	}
	if config == nil {
		t.Fatal("config should not be nil")
	}

	// Connect!
	c := goPlugin.NewClient(&goPlugin.ClientConfig{
		Cmd:             nil,
		HandshakeConfig: testHandshake,
		Plugins:         pluginMap,
		Reattach:        config,
	})
	client, err := c.Client()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Pinging should work
	if err := client.Ping(); err != nil {
		t.Fatalf("should not err: %s", err)
	}

	// Kill which should do nothing
	c.Kill()
	if err := client.Ping(); err != nil {
		t.Fatalf("should not err: %s", err)
	}

	// Request the plugin
	raw, err := client.Dispense("RpcMetricsPlugin")
	if err != nil {
		t.Fail()
	}

	plugin := raw.(rpc.MetricsPlugin)

	err = plugin.NewMetricsPlugin(v1alpha1.Metric{
		Provider: v1alpha1.MetricProvider{
			Plugin: map[string]json.RawMessage{"prometheus": json.RawMessage(`{"address":"http://prometheus.local", "query":"machine_cpu_cores"}`)},
		},
	})
	if err != nil {
		t.Fail()
	}

	// Canceling should cause an exit
	cancel()
	<-closeCh
}

func TestOpsmxProfile(t *testing.T) {

	logCtx := *log.WithFields(log.Fields{"plugin-test": "opsmx"})

	rpcPluginImp := &RpcPlugin{
		LogCtx: logCtx,
		client: NewHttpClient(),
	}

	opsmxMetric := OPSMXMetric{Application: "newapp",
		Profile:         "opsmx-profile-test",
		LifetimeMinutes: 9,
		Pass:            90,
		Marginal:        85,
		Services: []OPSMXService{{LogTemplateName: "logtemp",
			LogScopeVariables: "pod_name",
			CanaryLogScope:    "podHashCanary",
			BaselineLogScope:  "podHashBaseline",
		}},
	}
	t.Run("cdIntegration is missing in the secret - an error should be raised", func(t *testing.T) {
		secretData := map[string][]byte{
			"opsmxIsdUrl": []byte("https://opsmx.secret.tst"),
			"sourceName":  []byte("sourcename"),
			"user":        []byte("admin"),
		}
		rpcPluginImp.kubeclientset = getFakeClient(secretData)
		_, err := getOpsmxProfile(rpcPluginImp, opsmxMetric, "ns")
		assert.Contains(t, err.Error(), "`cdIntegration` key not present in the secret file")
	})

	t.Run("sourceName is missing in the secret - an error should be raised", func(t *testing.T) {
		secretData := map[string][]byte{
			"cdIntegration": []byte("true"),
			"opsmxIsdUrl":   []byte("https://opsmx.secret.tst"),
			"user":          []byte("admin"),
		}
		rpcPluginImp.kubeclientset = getFakeClient(secretData)
		_, err := getOpsmxProfile(rpcPluginImp, opsmxMetric, "ns")
		assert.Contains(t, err.Error(), "`sourceName` key not present in the secret file")
	})

	t.Run("opsmxIsdUrl is missing in the secret - an error should be raised", func(t *testing.T) {
		secretData := map[string][]byte{
			"cdIntegration": []byte("true"),
			"sourceName":    []byte("sourcename"),
			"user":          []byte("admin"),
		}
		rpcPluginImp.kubeclientset = getFakeClient(secretData)
		_, err := getOpsmxProfile(rpcPluginImp, opsmxMetric, "ns")
		assert.Contains(t, err.Error(), "`opsmxIsdUrl` key not present in the secret file")
	})

	t.Run("opsmxIsdUrl is missing in the secret - an error should be raised", func(t *testing.T) {
		secretData := map[string][]byte{
			"cdIntegration": []byte("true"),
			"sourceName":    []byte("sourcename"),
			"user":          []byte("admin"),
		}
		rpcPluginImp.kubeclientset = getFakeClient(secretData)
		_, err := getOpsmxProfile(rpcPluginImp, opsmxMetric, "ns")
		assert.Contains(t, err.Error(), "`opsmxIsdUrl` key not present in the secret file")
	})

	t.Run("user is missing in the secret - an error should be raised", func(t *testing.T) {
		secretData := map[string][]byte{
			"cdIntegration": []byte("true"),
			"opsmxIsdUrl":   []byte("https://opsmx.secret.tst"),
		}
		rpcPluginImp.kubeclientset = getFakeClient(secretData)
		_, err := getOpsmxProfile(rpcPluginImp, opsmxMetric, "ns")
		assert.Contains(t, err.Error(), "`user` key not present in the secret file")
	})
	t.Run("cdIntegration is neither true nor false in the secret - an error should be raised", func(t *testing.T) {
		secretData := map[string][]byte{
			"cdIntegration": []byte("test"),
			"opsmxIsdUrl":   []byte("https://opsmx.secret.tst"),
			"user":          []byte("admin"),
			"sourceName":    []byte("sourcename"),
		}
		rpcPluginImp.kubeclientset = getFakeClient(secretData)
		_, err := getOpsmxProfile(rpcPluginImp, opsmxMetric, "ns")
		assert.Contains(t, err.Error(), "cdIntegration should be either true or false")
	})

	t.Run("basic flow - no error is raised", func(t *testing.T) {
		secretData := map[string][]byte{
			"cdIntegration": []byte("true"),
			"opsmxIsdUrl":   []byte("https://opsmx.secret.tst"),
			"user":          []byte("admin"),
			"sourceName":    []byte("sourcename"),
		}
		rpcPluginImp.kubeclientset = getFakeClient(secretData)
		_, err := getOpsmxProfile(rpcPluginImp, opsmxMetric, "ns")
		assert.Nil(t, err)
	})

	t.Run("when user and url are also defined in the metric - values from the metric get picked up", func(t *testing.T) {
		secretData := map[string][]byte{
			"cdIntegration": []byte("true"),
			"opsmxIsdUrl":   []byte("https://opsmx.secret.tst"),
			"user":          []byte("admin"),
			"sourceName":    []byte("sourcename"),
		}
		rpcPluginImp.kubeclientset = getFakeClient(secretData)

		opsmxMetric := OPSMXMetric{Application: "newapp",
			OpsmxIsdUrl:     "https://url.from.metric",
			User:            "userFromMetric",
			Profile:         "opsmx-profile-test",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}
		opsmxProfileData, err := getOpsmxProfile(rpcPluginImp, opsmxMetric, "ns")
		assert.Nil(t, err)
		assert.Equal(t, "https://url.from.metric", opsmxProfileData.opsmxIsdUrl)
		assert.Equal(t, "userFromMetric", opsmxProfileData.user)
	})

}

func getFakeClient(data map[string][]byte) *k8sfake.Clientset {
	opsmxSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultSecretName,
		},
		Data: data,
	}
	fakeClient := k8sfake.NewSimpleClientset()
	fakeClient.PrependReactor("get", "*", func(action kubetesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, opsmxSecret, nil
	})
	return fakeClient
}
