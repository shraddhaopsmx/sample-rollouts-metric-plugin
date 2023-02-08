package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-rollouts/utils/plugin/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-rollouts/metricproviders/plugin"
	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	metricutil "github.com/argoproj/argo-rollouts/utils/metric"
	timeutil "github.com/argoproj/argo-rollouts/utils/time"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	templateApi               = "/autopilot/api/v5/external/template?sha1=%s&templateType=%s&templateName=%s"
	v5configIdLookupURLFormat = `/autopilot/api/v5/registerCanary`
	scoreUrlFormat            = `/autopilot/v5/canaries/`
	resumeAfter               = 3 * time.Second
	defaultTimeout            = 30
	defaultSecretName         = "opsmx-profile"
	cdIntegrationArgoRollouts = "argorollouts"
	cdIntegrationArgoCD       = "argocd"
	opsmxPlugin               = "opsmx"
)

// Here is a real implementation of MetricsPlugin
type RpcPlugin struct {
	LogCtx        log.Entry
	kubeclientset kubernetes.Interface
	client        http.Client
}

func (g *RpcPlugin) NewMetricsPlugin(metric v1alpha1.Metric) types.RpcError {
	config, err := rest.InClusterConfig()
	if err != nil {
		return types.RpcError{ErrorString: err.Error()}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return types.RpcError{ErrorString: err.Error()}
	}

	httpclient := NewHttpClient()
	g.client = httpclient
	g.kubeclientset = clientset

	return types.RpcError{}
}

func (g *RpcPlugin) Run(anaysisRun *v1alpha1.AnalysisRun, metric v1alpha1.Metric) v1alpha1.Measurement {
	startTime := timeutil.MetaNow()
	newMeasurement := v1alpha1.Measurement{
		StartedAt: &startTime,
	}

	OPSMXMetric := OPSMXMetric{}
	if err := json.Unmarshal(metric.Provider.Plugin[opsmxPlugin], &OPSMXMetric); err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}
	log.Info("The metric is ---")
	res2B, _ := json.Marshal(OPSMXMetric)
	log.Info("It is ----")
	log.Infof(string(res2B))
	if err := OPSMXMetric.basicChecks(); err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	secretData, err := OPSMXMetric.getDataSecret(g, anaysisRun)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	if err := OPSMXMetric.checkISDUrl(g, secretData["opsmxIsdUrl"]); err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	err = OPSMXMetric.getTimeVariables()
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	log.Info("generating the payload")
	canaryurl, err := url.JoinPath(secretData["opsmxIsdUrl"], v5configIdLookupURLFormat)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}
	payload, err := OPSMXMetric.generatePayload(g, secretData, "/tmp")
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}
	log.Info(payload)
	log.Info("sending a POST request to registerCanary with the payload")
	data, scoreURL, urlToken, err := makeRequest(g.client, "POST", canaryurl, payload, secretData["user"])
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}
	//Struct to record canary Response
	type canaryResponse struct {
		Error    string      `json:"error,omitempty"`
		Message  string      `json:"message,omitempty"`
		CanaryId json.Number `json:"canaryId,omitempty"`
	}
	var canary canaryResponse

	err = json.Unmarshal(data, &canary)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}
	log.Info("register canary response ", canary)
	if canary.Error != "" {
		errMessage := fmt.Sprintf("analysis Error: %s\nMessage: %s", canary.Error, canary.Message)
		err := errors.New(errMessage)
		if err != nil {
			return metricutil.MarkMeasurementError(newMeasurement, err)
		}
	}
	if scoreURL == "" {
		return metricutil.MarkMeasurementError(newMeasurement, errors.New("analysis Error: score url not found"))
	}
	data, _, _, err = makeRequest(g.client, "GET", scoreURL, "", secretData["user"])
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	var status map[string]interface{}
	var reportUrlJson map[string]interface{}

	err = json.Unmarshal(data, &status)
	if err != nil {
		errorMessage := fmt.Sprintf("analysis Error: Error in post processing canary Response: %v", err)
		return metricutil.MarkMeasurementError(newMeasurement, errors.New(errorMessage))
	}
	jsonBytes, _ := json.MarshalIndent(status["canaryResult"], "", "   ")
	err = json.Unmarshal(jsonBytes, &reportUrlJson)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}
	reportUrl := reportUrlJson["canaryReportURL"]

	mapMetadata := make(map[string]string)
	mapMetadata["canaryId"] = string(canary.CanaryId)
	mapMetadata["gateUrl"] = secretData["gateUrl"]
	mapMetadata["reportUrl"] = fmt.Sprintf("%s", reportUrl)
	mapMetadata["reportId"] = urlToken

	resumeTime := metav1.NewTime(timeutil.Now().Add(resumeAfter))
	newMeasurement.Metadata = mapMetadata
	newMeasurement.ResumeAt = &resumeTime
	newMeasurement.Phase = v1alpha1.AnalysisPhaseRunning
	finishedTime := timeutil.MetaNow()
	newMeasurement.FinishedAt = &finishedTime
	return newMeasurement
}

func processResume(data []byte, metric OPSMXMetric, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	var (
		canaryScore string
		result      map[string]interface{}
		finalScore  map[string]interface{}
	)

	if !json.Valid(data) {
		err := errors.New("invalid Response")
		return metricutil.MarkMeasurementError(measurement, err)
	}

	json.Unmarshal(data, &result)
	jsonBytes, _ := json.MarshalIndent(result["canaryResult"], "", "   ")
	json.Unmarshal(jsonBytes, &finalScore)
	if finalScore["overallScore"] == nil {
		canaryScore = "0"
	} else {
		canaryScore = fmt.Sprintf("%v", finalScore["overallScore"])
	}

	var score int
	if strings.Contains(canaryScore, ".") {
		floatScore, _ := strconv.ParseFloat(canaryScore, 64)
		score = int(roundFloat(floatScore, 0))
	} else {
		score, _ = strconv.Atoi(canaryScore)
	}
	measurement.Value = canaryScore
	measurement.Phase = evaluateResult(score, metric.Pass)
	if measurement.Phase == "Failed" && metric.LookBackType != "" {
		measurement.Metadata["interval analysis message"] = fmt.Sprintf("Interval Analysis Failed at intervalNo. %s", measurement.Metadata["Current intervalNo"])
	}
	return measurement
}

func (g *RpcPlugin) Resume(analysisRun *v1alpha1.AnalysisRun, metric v1alpha1.Metric, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	OPSMXMetric := OPSMXMetric{}
	if err := json.Unmarshal(metric.Provider.Plugin[opsmxPlugin], &OPSMXMetric); err != nil {
		return metricutil.MarkMeasurementError(measurement, err)
	}

	secretData, err := OPSMXMetric.getDataSecret(g, analysisRun)
	if err != nil {
		return metricutil.MarkMeasurementError(measurement, err)
	}

	scoreURL, err := url.JoinPath(secretData["opsmxIsdUrl"], scoreUrlFormat, measurement.Metadata["canaryId"])
	if err != nil {
		return metricutil.MarkMeasurementError(measurement, err)
	}

	data, _, _, err := makeRequest(g.client, "GET", scoreURL, "", secretData["user"])
	if err != nil {
		return metricutil.MarkMeasurementError(measurement, err)
	}
	var status map[string]interface{}
	json.Unmarshal(data, &status)
	a, _ := json.MarshalIndent(status["status"], "", "   ")
	json.Unmarshal(a, &status)

	var reportUrlJson map[string]interface{}
	jsonBytes, _ := json.MarshalIndent(status["canaryResult"], "", "   ")
	json.Unmarshal(jsonBytes, &reportUrlJson)
	reportUrl := reportUrlJson["canaryReportURL"]
	measurement.Metadata["reportUrl"] = fmt.Sprintf("%s", reportUrl)

	if OPSMXMetric.LookBackType != "" {
		measurement.Metadata["Current intervalNo"] = fmt.Sprintf("%v", reportUrlJson["intervalNo"])
	}
	//if the status is Running, resume analysis after delay
	if status["status"] == "RUNNING" {
		resumeTime := metav1.NewTime(timeutil.Now().Add(resumeAfter))
		measurement.ResumeAt = &resumeTime
		measurement.Phase = v1alpha1.AnalysisPhaseRunning
		return measurement
	}
	//if run is cancelled mid-run
	if status["status"] == "CANCELLED" {
		measurement.Phase = v1alpha1.AnalysisPhaseFailed
		measurement.Message = "Analysis Cancelled"
	} else {
		//POST-Run process
		measurement = processResume(data, OPSMXMetric, measurement)
	}
	finishTime := timeutil.MetaNow()
	measurement.FinishedAt = &finishTime
	return measurement
}

func (g *RpcPlugin) Terminate(analysisRun *v1alpha1.AnalysisRun, metric v1alpha1.Metric, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	return measurement
}

func (g *RpcPlugin) GarbageCollect(*v1alpha1.AnalysisRun, v1alpha1.Metric, int) types.RpcError {
	return types.RpcError{}
}

func (g *RpcPlugin) Type() string {
	return plugin.ProviderType
}

func (g *RpcPlugin) GetMetadata(metric v1alpha1.Metric) map[string]string {
	metricsMetadata := make(map[string]string)
	return metricsMetadata
}

func NewHttpClient() http.Client {
	c := http.Client{
		Timeout: defaultTimeout * time.Second,
	}
	return c
}
