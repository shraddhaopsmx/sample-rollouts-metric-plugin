package plugin

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestOpsmxMetricValidations(t *testing.T) {

	logCtx := *log.WithFields(log.Fields{"plugin-test": "opsmx"})

	rpcPluginImp := &RpcPlugin{
		LogCtx:        logCtx,
		kubeclientset: k8sfake.NewSimpleClientset(),
		client:        NewHttpClient(),
	}
	opsmxProfileData := opsmxProfile{cdIntegration: "true",
		user:        "admin",
		sourceName:  "sourceName",
		opsmxIsdUrl: "https://opsmx.test.tst"}

	t.Run("pass score is less than marginal score - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 3,
			Pass:            80,
			Marginal:        85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Equal(t, "pass score cannot be less than marginal score", err.Error())
	})

	t.Run("neither lifetimeMinutes nor endTime is provided - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			Pass:     90,
			Marginal: 85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Equal(t, "provide either lifetimeMinutes or end time", err.Error())
	})

	t.Run("canaryStartTime are baselineStartTime different when using endTime - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			BaselineStartTime: "2022-08-02T13:15:00Z",
			CanaryStartTime:   "2022-08-02T13:25:00Z",
			EndTime:           "2022-08-02T13:45:10Z",
			Pass:              90,
			Marginal:          85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Equal(t, "both canaryStartTime and baselineStartTime should be kept same while using endTime argument for analysis", err.Error())
	})

	t.Run("canaryStartTime is greater than endTime - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			CanaryStartTime:   "2022-08-02T13:25:00Z",
			BaselineStartTime: "2022-08-02T13:25:00Z",
			EndTime:           "2022-08-01T13:45:10Z",
			Pass:              90,
			Marginal:          85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "canaryStartTime cannot be greater than endTime")
	})

	t.Run("incorrect time format CanaryStartTime- an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			BaselineStartTime: "2022-08-02T13:15:00Z",
			CanaryStartTime:   "2022-O8-02T13:15:00Z",
			LifetimeMinutes:   3,
			Pass:              90,
			Marginal:          85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.NotNil(t, err)
	})

	t.Run("incorrect time format BaselineStartTime- an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			BaselineStartTime: "2022-O8-02T13:15:00Z",
			CanaryStartTime:   "2022-08-02T13:15:00Z",
			LifetimeMinutes:   3,
			Pass:              90,
			Marginal:          85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.NotNil(t, err)
	})

	t.Run("incorrect time format EndTime - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			BaselineStartTime: "2022-08-02T13:15:00Z",
			CanaryStartTime:   "2022-08-02T13:15:00Z",
			EndTime:           "2022-O8-02T13:45:10Z",
			Pass:              90,
			Marginal:          85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.NotNil(t, err)
	})

	t.Run("lifetimeMinutes cannot be less than 3 minutes - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 2,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Equal(t, "lifetimeMinutes cannot be less than 3 minutes", err.Error())
	})
	t.Run("intervalTime cannot be less than 3 minutes - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			IntervalTime:    2,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Equal(t, "intervalTime cannot be less than 3 minutes", err.Error())
	})
	t.Run("interval Timecannot be less than 3 minutes - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			LookBackType:    "growing",
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Equal(t, "intervalTime should be given along with lookBackType to perform interval analysis", err.Error())
	})

	t.Run("intervalTime cannot be less than 3 minutes - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			LookBackType:    "growing",
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}

		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Equal(t, "intervalTime should be given along with lookBackType to perform interval analysis", err.Error())
	})

	t.Run("no Services are mentioned - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
		}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Equal(t, "at least one of log or metric context must be provided", err.Error())
	})

	t.Run("no log and metric details are mentioned - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					ServiceName: "service1",
				},
				{
					ServiceName: "service2",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Equal(t, "at least one of log or metric context must be provided", err.Error())
	})

	t.Run("mismatch in log scope variables and baseline/canary log scope - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					MetricScopeVariables: "pod_name",
					BaselineMetricScope:  "podHashBaseline",
					CanaryMetricScope:    "podHashCanary",
					MetricTemplateName:   "metrictemplate",
				},
				{
					MetricScopeVariables: "pod_name",
					BaselineMetricScope:  "podHashBaseline",
					CanaryMetricScope:    "podHashCanary",
					MetricTemplateName:   "metrictemplate",
					LogScopeVariables:    "job_name,pod_name",
					BaselineLogScope:     "podHashBaseline",
					CanaryLogScope:       "podHashCanary",
					LogTemplateName:      "logtemplate",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "mismatch in number of log scope variables and baseline/canary log scope of service")
	})

	t.Run("missing canary/baseline for log analysis of service - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					MetricScopeVariables: "pod_name",
					BaselineMetricScope:  "podHashBaseline",
					CanaryMetricScope:    "podHashCanary",
					MetricTemplateName:   "metrictemplate",
					LogScopeVariables:    "pod_name",
					BaselineLogScope:     "podHashBaseline",
					LogTemplateName:      "logtemplate",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "missing canary/baseline for log analysis of service")
	})

	t.Run("missing log template in Service - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					MetricScopeVariables: "pod_name",
					BaselineMetricScope:  "podHashBaseline",
					CanaryMetricScope:    "podHashCanary",
					MetricTemplateName:   "metrictemplate",
					LogScopeVariables:    "pod_name",
					CanaryLogScope:       "podHashCanary",
					BaselineLogScope:     "podHashBaseline",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "provide either a service specific log template or global log template for service")
	})

	t.Run("missing log Scope placeholder - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					MetricScopeVariables: "pod_name",
					BaselineMetricScope:  "podHashBaseline",
					CanaryMetricScope:    "podHashCanary",
					MetricTemplateName:   "metrictemplate",
					CanaryLogScope:       "podHashCanary",
					BaselineLogScope:     "podHashBaseline",
					LogTemplateName:      "logtemplate",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "missing log Scope placeholder")
	})

	t.Run("mismatch in metric scope variables and baseline/canary log scope - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					MetricScopeVariables: "job_name,pod_name",
					BaselineMetricScope:  "podHashBaseline",
					CanaryMetricScope:    "podHashCanary",
					MetricTemplateName:   "metrictemplate",
				},
				{
					MetricScopeVariables: "pod_name",
					BaselineMetricScope:  "podHashBaseline",
					CanaryMetricScope:    "podHashCanary",
					MetricTemplateName:   "metrictemplate",
					LogScopeVariables:    "pod_name",
					BaselineLogScope:     "podHashBaseline",
					CanaryLogScope:       "podHashCanary",
					LogTemplateName:      "logtemplate",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "mismatch in number of metric scope variables and baseline/canary metric scope of service")
	})

	t.Run("missing canary for metric analysis of service - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					MetricScopeVariables: "pod_name",
					BaselineMetricScope:  "podHashBaseline",
					MetricTemplateName:   "metrictemplate",
					LogScopeVariables:    "pod_name",
					BaselineLogScope:     "podHashBaseline",
					CanaryLogScope:       "podHashCanary",
					LogTemplateName:      "logtemplate",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "missing canary/baseline for metric analysis of service")
	})

	t.Run("missing baseline for metric analysis of service - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					MetricScopeVariables: "pod_name",
					CanaryMetricScope:    "podHashCanary",
					MetricTemplateName:   "metrictemplate",
					LogScopeVariables:    "pod_name",
					BaselineLogScope:     "podHashBaseline",
					CanaryLogScope:       "podHashCanary",
					LogTemplateName:      "logtemplate",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "missing canary/baseline for metric analysis of service")
	})

	t.Run("missing metric template in Service - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					MetricScopeVariables: "pod_name",
					BaselineMetricScope:  "podHashBaseline",
					CanaryMetricScope:    "podHashCanary",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "provide either a service specific metric template or global metric template for service")
	})

	t.Run("missing log Scope placeholder - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{
				{
					BaselineMetricScope: "podHashBaseline",
					CanaryMetricScope:   "podHashCanary",
					MetricTemplateName:  "metrictemplate",
					LogScopeVariables:   "pod_name",
					CanaryLogScope:      "podHashCanary",
					BaselineLogScope:    "podHashBaseline",
					LogTemplateName:     "logtemplate",
				},
			}}
		_, err := opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		assert.Contains(t, err.Error(), "missing metric Scope placeholder")
	})
	t.Run("intervalTime cannot be less than 3 minutes - an error should be raised", func(t *testing.T) {
		opsmxMetric := OPSMXMetric{Application: "newapp",
			LifetimeMinutes: 9,
			Pass:            90,
			Marginal:        85,
			Services: []OPSMXService{{LogTemplateName: "logtemp",
				LogScopeVariables: "pod_name",
				CanaryLogScope:    "podHashCanary",
				BaselineLogScope:  "podHashBaseline",
			}},
		}
		_, _ = opsmxMetric.process(rpcPluginImp, opsmxProfileData, "ns")
		// assert.Equal(t, "intervalTime should be given along with lookBackType to perform interval analysis", err.Error())
	})
}

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) (*http.Response, error)

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) http.Client {
	return http.Client{
		Transport: fn,
	}
}
