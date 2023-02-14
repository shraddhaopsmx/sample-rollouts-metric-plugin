package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OPSMXMetric struct {
	User                 string         `yaml:"user,omitempty" json:"user,omitempty"`
	OpsmxIsdUrl          string         `yaml:"opsmxIsdUrl,omitempty" json:"opsmxIsdUrl,omitempty"`
	Application          string         `yaml:"application" json:"application"`
	Profile              string         `yaml:"profile,omitempty" json:"profile,omitempty"`
	BaselineStartTime    string         `yaml:"baselineStartTime,omitempty" json:"baselineStartTime,omitempty"`
	CanaryStartTime      string         `yaml:"canaryStartTime,omitempty" json:"canaryStartTime,omitempty"`
	LifetimeMinutes      int            `yaml:"lifetimeMinutes,omitempty" json:"lifetimeMinutes,omitempty"`
	EndTime              string         `yaml:"endTime,omitempty" json:"endTime,omitempty"`
	GlobalLogTemplate    string         `yaml:"globalLogTemplate,omitempty" json:"globalLogTemplate,omitempty"`
	GlobalMetricTemplate string         `yaml:"globalMetricTemplate,omitempty" json:"globalMetricTemplate,omitempty"`
	Pass                 int            `yaml:"passScore" json:"passScore"`
	Marginal             int            `yaml:"marginalScore" json:"marginalScore"`
	Services             []OPSMXService `yaml:"serviceList,omitempty" json:"serviceList,omitempty"`
	IntervalTime         int            `yaml:"intervalTime,omitempty" json:"intervalTime,omitempty"`
	LookBackType         string         `yaml:"lookBackType,omitempty" json:"lookBackType,omitempty"`
	Delay                int            `yaml:"delay,omitempty" json:"delay,omitempty"`
	GitOPS               bool           `yaml:"gitops,omitempty" json:"gitops,omitempty"`
}

type OPSMXService struct {
	LogTemplateName       string `yaml:"logTemplateName,omitempty" json:"logTemplateName,omitempty"`
	LogTemplateVersion    string `yaml:"logTemplateVersion,omitempty" json:"logTemplateVersion,omitempty"`
	MetricTemplateName    string `yaml:"metricTemplateName,omitempty" json:"metricTemplateName,omitempty"`
	MetricTemplateVersion string `yaml:"metricTemplateVersion,omitempty" json:"metricTemplateVersion,omitempty"`
	LogScopeVariables     string `yaml:"logScopeVariables,omitempty" json:"logScopeVariables,omitempty"`
	BaselineLogScope      string `yaml:"baselineLogScope,omitempty" json:"baselineLogScope,omitempty"`
	CanaryLogScope        string `yaml:"canaryLogScope,omitempty" json:"canaryLogScope,omitempty"`
	MetricScopeVariables  string `yaml:"metricScopeVariables,omitempty" json:"metricScopeVariables,omitempty"`
	BaselineMetricScope   string `yaml:"baselineMetricScope,omitempty" json:"baselineMetricScope,omitempty"`
	CanaryMetricScope     string `yaml:"canaryMetricScope,omitempty" json:"canaryMetricScope,omitempty"`
	ServiceName           string `yaml:"serviceName,omitempty" json:"serviceName,omitempty"`
}

type jobPayload struct {
	Application       string              `json:"application"`
	SourceName        string              `json:"sourceName"`
	SourceType        string              `json:"sourceType"`
	CanaryConfig      canaryConfig        `json:"canaryConfig"`
	CanaryDeployments []canaryDeployments `json:"canaryDeployments"`
}

type canaryConfig struct {
	LifetimeMinutes          string                   `json:"lifetimeMinutes"`
	LookBackType             string                   `json:"lookBackType,omitempty"`
	IntervalTime             string                   `json:"interval,omitempty"`
	Delays                   string                   `json:"delay,omitempty"`
	CanaryHealthCheckHandler canaryHealthCheckHandler `json:"canaryHealthCheckHandler"`
	CanarySuccessCriteria    canarySuccessCriteria    `json:"canarySuccessCriteria"`
}

type canaryHealthCheckHandler struct {
	MinimumCanaryResultScore string `json:"minimumCanaryResultScore"`
}

type canarySuccessCriteria struct {
	CanaryResultScore string `json:"canaryResultScore"`
}

type canaryDeployments struct {
	CanaryStartTimeMs   string     `json:"canaryStartTimeMs"`
	BaselineStartTimeMs string     `json:"baselineStartTimeMs"`
	Canary              *logMetric `json:"canary,omitempty"`
	Baseline            *logMetric `json:"baseline,omitempty"`
}
type logMetric struct {
	Log    map[string]map[string]string `json:"log,omitempty"`
	Metric map[string]map[string]string `json:"metric,omitempty"`
}

type service struct {
	logMetric              string
	serviceName            string
	serviceGate            string
	template               string
	templateVersion        string
	templateSha            string
	scopeVariables         string
	canaryScopeVariables   string
	baselineScopeVariables string
}

func (metric *OPSMXMetric) process(g *RpcPlugin, opsmxProfileData opsmxProfile, ns string) (string, error) {
	if err := metric.basicChecks(); err != nil {
		return "", err
	}
	if err := metric.setCanaryStartTime(); err != nil {
		return "", err
	}
	if err := metric.setBaselineStartTime(); err != nil {
		return "", err
	}
	if err := metric.setIntervalTime(); err != nil {
		return "", err
	}

	services, err := metric.processServices(g, opsmxProfileData, ns)
	if err != nil {
		return "", err
	}

	payload, err := metric.generatePayload(opsmxProfileData, services)
	if err != nil {
		return "", nil
	}
	return payload, nil
}

// TODO: may or may not convert to validators
func (metric *OPSMXMetric) basicChecks() error {
	if metric.Pass <= metric.Marginal {
		return errors.New("pass score cannot be less than marginal score")
	}
	if metric.LifetimeMinutes == 0 && metric.EndTime == "" {
		return errors.New("provide either lifetimeMinutes or end time")
	}
	if metric.CanaryStartTime != metric.BaselineStartTime && metric.LifetimeMinutes == 0 {
		return errors.New("both canaryStartTime and baselineStartTime should be kept same while using endTime argument for analysis")
	}
	if metric.LifetimeMinutes != 0 && metric.LifetimeMinutes < 3 {
		return errors.New("lifetimeMinutes cannot be less than 3 minutes")
	}
	if metric.IntervalTime != 0 && metric.IntervalTime < 3 {
		return errors.New("intervalTime cannot be less than 3 minutes")
	}
	if metric.LookBackType != "" && metric.IntervalTime == 0 {
		return errors.New("intervalTime should be given along with lookBackType to perform interval analysis")
	}
	return nil
}

func (metric *OPSMXMetric) setCanaryStartTime() error {
	canaryStartTime := fmt.Sprintf("%d", time.Now().UnixMilli())
	if metric.CanaryStartTime != "" {
		tsStart, err := time.Parse(time.RFC3339, metric.CanaryStartTime)
		if err != nil {
			return fmt.Errorf("error in parsing canaryStartTime: %v", err)
		}
		canaryStartTime = fmt.Sprintf("%d", tsStart.UnixMilli())
	}
	metric.CanaryStartTime = canaryStartTime
	return nil
}

func (metric *OPSMXMetric) setBaselineStartTime() error {
	baselineStartTime := fmt.Sprintf("%d", time.Now().UnixMilli())
	if metric.BaselineStartTime != "" {
		tsStart, err := time.Parse(time.RFC3339, metric.BaselineStartTime)
		if err != nil {
			return fmt.Errorf("error in parsing baselineStartTime: %v", err)
		}
		baselineStartTime = fmt.Sprintf("%d", tsStart.UnixMilli())
	}
	metric.BaselineStartTime = baselineStartTime
	return nil
}

// TODO: refactor
func (metric *OPSMXMetric) setIntervalTime() error {
	if metric.LifetimeMinutes == 0 {
		tsEnd, err := time.Parse(time.RFC3339, metric.EndTime)
		if err != nil {
			return fmt.Errorf("provider config map validation error: Error in parsing endTime: %v", err)
		}
		//TODO: move this out
		if metric.CanaryStartTime != "" && metric.CanaryStartTime > metric.EndTime {
			err := errors.New("provider config map validation error: canaryStartTime cannot be greater than endTime")
			return err
		}
		tsStart := time.Now()
		//TODO: dont ignore the errors
		if metric.CanaryStartTime != "" {
			tsStart, _ = time.Parse(time.RFC3339, metric.CanaryStartTime)
		}
		tsDifference := tsEnd.Sub(tsStart)
		min, _ := time.ParseDuration(tsDifference.String())
		metric.LifetimeMinutes = int(roundFloat(min.Minutes(), 0))
	}
	return nil
}
func (metric *OPSMXMetric) processServices(g *RpcPlugin, opsmxProfileData opsmxProfile, namespace string) ([]service, error) {
	servicesArray := []service{}
	for i, item := range metric.Services {
		//TODO: check why this has been addded
		// valid := false
		serviceName := fmt.Sprintf("service%d", i+1)
		if item.ServiceName != "" {
			serviceName = item.ServiceName
		}
		if serviceExists(servicesArray, serviceName) {
			return []service{}, fmt.Errorf("serviceName '%s' mentioned exists more than once", serviceName)
		}
		gateName := fmt.Sprintf("gate%d", i+1)
		isLog, err := metric.validateLogs(item, serviceName)
		if err != nil {
			return []service{}, err
		}
		if isLog {
			logTemplate := item.LogTemplateName
			if logTemplate == "" {
				logTemplate = metric.GlobalLogTemplate
			}
			serviceData := service{logMetric: "LOG",
				serviceName:            serviceName,
				serviceGate:            gateName,
				template:               logTemplate,
				scopeVariables:         item.LogScopeVariables,
				canaryScopeVariables:   item.CanaryLogScope,
				baselineScopeVariables: item.BaselineLogScope}

			if metric.GitOPS && item.LogTemplateVersion == "" {
				shaLog, err := generateTemplateGitops(logTemplate, "LOG", item.LogScopeVariables, g, opsmxProfileData, namespace)
				if err != nil {
					return []service{}, err
				}
				serviceData.templateSha = shaLog
			}
			//TODO: handle more here
			if item.LogTemplateVersion == "" {
				serviceData.templateVersion = item.LogTemplateVersion
			}
			servicesArray = append(servicesArray, serviceData)
		}
		isMetric, err := metric.validateMetrics(item, serviceName)
		if err != nil {
			return []service{}, err
		}
		if isMetric {
			metricTemplate := item.MetricTemplateName
			if metricTemplate == "" {
				metricTemplate = metric.GlobalMetricTemplate
			}
			serviceData := service{logMetric: "METRIC",
				serviceName:            serviceName,
				serviceGate:            gateName,
				template:               metricTemplate,
				scopeVariables:         item.MetricScopeVariables,
				canaryScopeVariables:   item.CanaryMetricScope,
				baselineScopeVariables: item.BaselineMetricScope}

			if metric.GitOPS && item.LogTemplateVersion == "" {
				shaMetric, err := generateTemplateGitops(metricTemplate, "METRIC", item.MetricScopeVariables, g, opsmxProfileData, namespace)
				if err != nil {
					return []service{}, err
				}
				serviceData.templateSha = shaMetric
			}
			//TODO: handle more here
			if item.MetricTemplateVersion == "" {
				serviceData.templateVersion = item.LogTemplateVersion
			}
			servicesArray = append(servicesArray, serviceData)
		}
	}
	return servicesArray, nil
}

func (metric *OPSMXMetric) generatePayload(opsmxProfileData opsmxProfile, services []service) (string, error) {

	var intervalTime string
	if metric.IntervalTime != 0 {
		intervalTime = fmt.Sprintf("%d", metric.IntervalTime)
	}

	var opsmxdelay string
	if metric.Delay != 0 {
		opsmxdelay = fmt.Sprintf("%d", metric.Delay)
	}

	payload := jobPayload{Application: metric.Application,
		SourceName: opsmxProfileData.sourceName,
		SourceType: opsmxProfileData.cdIntegration,
		CanaryConfig: canaryConfig{
			LifetimeMinutes: fmt.Sprintf("%d", metric.LifetimeMinutes),
			LookBackType:    metric.LookBackType,
			IntervalTime:    intervalTime,
			Delays:          opsmxdelay,
			CanaryHealthCheckHandler: canaryHealthCheckHandler{
				MinimumCanaryResultScore: fmt.Sprintf("%d", metric.Marginal),
			},
			CanarySuccessCriteria: canarySuccessCriteria{
				CanaryResultScore: fmt.Sprintf("%d", metric.Pass),
			},
		},
	}
	payload.CanaryDeployments = metric.populateCanaryDeployment(services)
	buffer, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(buffer), err

}

func (metric *OPSMXMetric) populateCanaryDeployment(services []service) []canaryDeployments {
	canaryDeploymentsArray := []canaryDeployments{}
	deployment := canaryDeployments{
		BaselineStartTimeMs: metric.BaselineStartTime,
		CanaryStartTimeMs:   metric.CanaryStartTime,
		Baseline: &logMetric{
			Log:    map[string]map[string]string{},
			Metric: map[string]map[string]string{},
		},
		Canary: &logMetric{
			Log:    map[string]map[string]string{},
			Metric: map[string]map[string]string{},
		},
	}
	for _, svc := range services {
		if svc.logMetric == "LOG" {
			deployment.Baseline.Log[svc.serviceName] = map[string]string{
				svc.scopeVariables: svc.baselineScopeVariables,
				"serviceGate":      svc.serviceGate,
				"template":         svc.template,
				"templateSha1":     svc.templateSha,
				"templateVersion":  svc.templateVersion,
			}
			deployment.Canary.Log[svc.serviceName] = map[string]string{
				svc.scopeVariables: svc.canaryScopeVariables,
				"serviceGate":      svc.serviceGate,
				"template":         svc.template,
				"templateSha1":     svc.templateSha,
				"templateVersion":  svc.templateVersion,
			}
		}
		deployment.Baseline.Metric[svc.serviceName] = map[string]string{
			svc.scopeVariables: svc.baselineScopeVariables,
			"serviceGate":      svc.serviceGate,
			"template":         svc.template,
			"templateSha1":     svc.templateSha,
			"templateVersion":  svc.templateVersion,
		}
		deployment.Canary.Metric[svc.serviceName] = map[string]string{
			svc.scopeVariables: svc.canaryScopeVariables,
			"serviceGate":      svc.serviceGate,
			"template":         svc.template,
			"templateSha1":     svc.templateSha,
			"templateVersion":  svc.templateVersion,
		}

	}
	canaryDeploymentsArray = append(canaryDeploymentsArray, deployment)
	return canaryDeploymentsArray

}

func (metric *OPSMXMetric) validateLogs(item OPSMXService, serviceName string) (bool, error) {

	var isLog bool
	if item.LogScopeVariables == "" && item.BaselineLogScope != "" || item.LogScopeVariables == "" && item.CanaryLogScope != "" {
		errorMsg := fmt.Sprintf("provider config map validation error: missing log Scope placeholder for the provided baseline/canary of service '%s'", serviceName)
		return isLog, errors.New(errorMsg)
	}
	//For Log Analysis is to be added in analysis-run
	if item.LogScopeVariables != "" {
		isLog = true
		//Check if no baseline or canary
		if item.BaselineLogScope != "" && item.CanaryLogScope == "" {
			errorMsg := fmt.Sprintf("provider config map validation error: missing canary for log analysis of service '%s'", serviceName)
			return isLog, errors.New(errorMsg)
		}
		//Check if the number of placeholders provided dont match
		if len(strings.Split(item.LogScopeVariables, ",")) != len(strings.Split(item.BaselineLogScope, ",")) || len(strings.Split(item.LogScopeVariables, ",")) != len(strings.Split(item.CanaryLogScope, ",")) {
			errorMsg := fmt.Sprintf("provider config map validation error: mismatch in number of log scope variables and baseline/canary log scope of service '%s'", serviceName)
			return isLog, errors.New(errorMsg)
		}
		if item.LogTemplateName == "" && metric.GlobalLogTemplate == "" {
			errorMsg := fmt.Sprintf("provider config map validation error: provide either a service specific log template or global log template for service '%s'", serviceName)
			return isLog, errors.New(errorMsg)
		}
	}
	return isLog, nil
}

func (metric *OPSMXMetric) validateMetrics(item OPSMXService, serviceName string) (bool, error) {

	var isMetric bool
	if item.MetricScopeVariables == "" && item.BaselineMetricScope != "" || item.MetricScopeVariables == "" && item.CanaryMetricScope != "" {
		errorMsg := fmt.Sprintf("provider config map validation error: missing metric Scope placeholder for the provided baseline/canary of service '%s'", serviceName)
		return isMetric, errors.New(errorMsg)
	}
	//For metric analysis is to be added in analysis-run
	if item.MetricScopeVariables != "" {
		isMetric = true
		//Check if no baseline or canary
		if item.BaselineMetricScope == "" || item.CanaryMetricScope == "" {
			errorMsg := fmt.Sprintf("provider config map validation error: missing baseline/canary for metric analysis of service '%s'", serviceName)
			return isMetric, errors.New(errorMsg)
		}
		//Check if the number of placeholders provided dont match
		if len(strings.Split(item.MetricScopeVariables, ",")) != len(strings.Split(item.BaselineMetricScope, ",")) || len(strings.Split(item.MetricScopeVariables, ",")) != len(strings.Split(item.CanaryMetricScope, ",")) {
			errorMsg := fmt.Sprintf("provider config map validation error: mismatch in number of metric scope variables and baseline/canary metric scope of service '%s'", serviceName)
			return isMetric, errors.New(errorMsg)
		}
		if item.MetricTemplateName == "" && metric.GlobalMetricTemplate == "" {
			errorMsg := fmt.Sprintf("provider config map validation error: provide either a service specific metric template or global metric template for service: %s", serviceName)
			return isMetric, errors.New(errorMsg)
		}
	}
	return isMetric, nil
}

func generateTemplateGitops(templateName string, templateType string, scopeVars string, c *RpcPlugin, secret opsmxProfile, nameSpace string) (string, error) {
	v1ConfigMap, err := c.kubeclientset.CoreV1().ConfigMaps(nameSpace).Get(context.TODO(), templateName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("gitops '%s' template config map validation error: %v\n Action Required: Template must carry data element '%s'", templateName, err, templateName)
	}
	templateDataCM, ok := v1ConfigMap.Data[templateName]
	if !ok {
		return "", errors.New("something went wrong")
	}

	templateFileData := []byte(templateDataCM)
	if !isJSON(templateDataCM) {
		templateFileData, err = getTemplateDataYaml(templateFileData, templateName, templateType, scopeVars)
		if err != nil {
			return "", err
		}
	} else {
		checktemplateName := gjson.Get(string(templateFileData), "templateName")
		if checktemplateName.String() == "" {
			errmessage := fmt.Sprintf("gitops '%s' template config map validation error: template name not provided inside json", templateName)
			return "", errors.New(errmessage)
		}
		if templateName != checktemplateName.String() {
			errmessage := fmt.Sprintf("gitops '%s' template config map validation error: Mismatch between templateName and data.%s key", templateName, templateName)
			return "", errors.New(errmessage)
		}
	}

	sha1Code := generateSHA1(string(templateFileData))
	templateUrl, err := getTemplateUrl(secret.opsmxIsdUrl, sha1Code, templateType, templateName)
	if err != nil {
		return "", err
	}

	log.Debug("sending a GET request to gitops API")
	data, _, _, err := makeRequest(c.client, "GET", templateUrl, "", secret.user)
	if err != nil {
		return "", err
	}
	var templateVerification bool
	err = json.Unmarshal(data, &templateVerification)
	if err != nil {
		errorMessage := fmt.Sprintf("analysis Error: Expected bool response from gitops verifyTemplate response  Error: %v. Action: Check endpoint given in secret/providerConfig.", err)
		return "", errors.New(errorMessage)
	}
	templateData := sha1Code
	var templateCheckSave map[string]interface{}
	//TODO: refactor
	if !templateVerification {
		log.Debug("sending a POST request to gitops API")
		data, _, _, err = makeRequest(c.client, "POST", templateUrl, string(templateFileData), secret.user)
		if err != nil {
			return "", err
		}
		err = json.Unmarshal(data, &templateCheckSave)
		if err != nil {
			return "", err
		}
		log.Debugf("the value of templateCheckSave var is %v", templateCheckSave)
		var errorss string
		if templateCheckSave["errorMessage"] != nil && templateCheckSave["errorMessage"] != "" {
			errorss = fmt.Sprintf("%v", templateCheckSave["errorMessage"])
		} else {
			errorss = fmt.Sprintf("%v", templateCheckSave["error"])
		}
		errorss = strings.Replace(strings.Replace(errorss, "[", "", -1), "]", "", -1)
		if templateCheckSave["status"] != "CREATED" {
			err = fmt.Errorf("gitops '%s' template config map validation error: %s", templateName, errorss)
			return "", err
		}
	}
	return templateData, nil
}

func getTemplateDataYaml(templateFileData []byte, template string, templateType string, ScopeVariables string) ([]byte, error) {
	if templateType == "LOG" {
		logData, err := processYamlLogs(templateFileData, template, ScopeVariables)
		if err != nil {
			return nil, err
		}
		return json.Marshal(logData)
	}
	metricStruct, err := processYamlMetrics(templateFileData, template, ScopeVariables)
	if err != nil {
		return nil, err
	}
	return json.Marshal(metricStruct)
}

// func (metric *OPSMXMetric) checkISDUrl(c *RpcPlugin, opsmxIsdUrl string) error {
// 	resp, err := c.client.Get(opsmxIsdUrl)
// 	if err != nil && metric.OpsmxIsdUrl != "" && !strings.Contains(err.Error(), "timeout") {
// 		errorMsg := fmt.Sprintf("provider config map validation error: incorrect opsmxIsdUrl: %v", opsmxIsdUrl)
// 		return errors.New(errorMsg)
// 	} else if err != nil && metric.OpsmxIsdUrl == "" && !strings.Contains(err.Error(), "timeout") {
// 		errorMsg := fmt.Sprintf("opsmx profile secret validation error: incorrect opsmxIsdUrl: %v", opsmxIsdUrl)
// 		return errors.New(errorMsg)
// 	} else if err != nil {
// 		return errors.New(err.Error())
// 	} else if resp.StatusCode != 200 {
// 		return errors.New(resp.Status)
// 	}
// 	return nil
// }
