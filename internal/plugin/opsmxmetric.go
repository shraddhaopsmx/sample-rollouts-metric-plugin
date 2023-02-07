package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"gopkg.in/yaml.v2"
)

type LogTemplateYaml struct {
	DisableDefaultsErrorTopics bool          `yaml:"disableDefaultErrorTopics" json:"-"`
	TemplateName               string        `yaml:"templateName" json:"templateName"`
	FilterKey                  string        `yaml:"filterKey" json:"filterKey"`
	TagEnabled                 bool          `yaml:"-" json:"tagEnabled"`
	MonitoringProvider         string        `yaml:"monitoringProvider" json:"monitoringProvider"`
	AccountName                string        `yaml:"accountName" json:"accountName"`
	ScoringAlgorithm           string        `yaml:"scoringAlgorithm" json:"scoringAlgorithm"`
	Index                      string        `yaml:"index,omitempty" json:"index,omitempty"`
	ResponseKeywords           string        `yaml:"responseKeywords" json:"responseKeywords"`
	ContextualCluster          bool          `yaml:"contextualCluster,omitempty" json:"contextualCluster,omitempty"`
	ContextualWindowSize       int           `yaml:"contextualWindowSize,omitempty" json:"contextualWindowSize,omitempty"`
	InfoScoring                bool          `yaml:"infoScoring,omitempty" json:"infoScoring,omitempty"`
	RegExFilter                bool          `yaml:"regExFilter,omitempty" json:"regExFilter,omitempty"`
	RegExResponseKey           string        `yaml:"regExResponseKey,omitempty" json:"regExResponseKey,omitempty"`
	RegularExpression          string        `yaml:"regularExpression,omitempty" json:"regularExpression,omitempty"`
	AutoBaseline               bool          `yaml:"autoBaseline,omitempty" json:"autoBaseline,omitempty"`
	Sensitivity                string        `yaml:"sensitivity,omitempty" json:"sensitivity,omitempty"`
	StreamID                   string        `yaml:"streamId,omitempty" json:"streamId,omitempty"`
	Tags                       []customTags  `yaml:"tags" json:"tags,omitempty"`
	ErrorTopics                []errorTopics `yaml:"errorTopics" json:"errorTopics"`
}

type customTags struct {
	ErrorStrings string `yaml:"errorString" json:"string"`
	Tag          string `yaml:"tag" json:"tag"`
}

type errorTopics struct {
	ErrorStrings string `yaml:"errorString" json:"string"`
	Topic        string `yaml:"topic" json:"topic"`
	Type         string `yaml:"-" json:"type"`
}

type OPSMXMetric struct {
	User                 string         `yaml:"user,omitempty"`
	OpsmxIsdUrl          string         `yaml:"opsmxIsdUrl,omitempty"`
	Application          string         `yaml:"application"`
	BaselineStartTime    string         `yaml:"baselineStartTime,omitempty"`
	CanaryStartTime      string         `yaml:"canaryStartTime,omitempty"`
	LifetimeMinutes      int            `yaml:"lifetimeMinutes,omitempty"`
	EndTime              string         `yaml:"endTime,omitempty"`
	GlobalLogTemplate    string         `yaml:"globalLogTemplate,omitempty"`
	GlobalMetricTemplate string         `yaml:"globalMetricTemplate,omitempty"`
	Pass                 int            `yaml:"passScore"`
	Services             []OPSMXService `yaml:"serviceList,omitempty"`
	IntervalTime         int            `yaml:"intervalTime,omitempty"`
	LookBackType         string         `yaml:"lookBackType,omitempty"`
	Delay                int            `yaml:"delay,omitempty"`
	GitOPS               bool           `yaml:"gitops,omitempty"`
}

type OPSMXService struct {
	LogTemplateName       string `yaml:"logTemplateName,omitempty"`
	LogTemplateVersion    string `yaml:"logTemplateVersion,omitempty"`
	MetricTemplateName    string `yaml:"metricTemplateName,omitempty"`
	MetricTemplateVersion string `yaml:"metricTemplateVersion,omitempty"`
	LogScopeVariables     string `yaml:"logScopeVariables,omitempty"`
	BaselineLogScope      string `yaml:"baselineLogScope,omitempty"`
	CanaryLogScope        string `yaml:"canaryLogScope,omitempty"`
	MetricScopeVariables  string `yaml:"metricScopeVariables,omitempty"`
	BaselineMetricScope   string `yaml:"baselineMetricScope,omitempty"`
	CanaryMetricScope     string `yaml:"canaryMetricScope,omitempty"`
	ServiceName           string `yaml:"serviceName,omitempty"`
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

const DefaultsErrorTopicsJson = `{
	"errorTopics": [
	  {
		"string": "OnOutOfMemoryError",
		"topic": "critical",
		"type": "default"
	  },
	  {
		"string": "StackOverflowError",
		"topic": "critical",
		"type": "default"
	  },
	  {
		"string": "ClassNotFoundException",
		"topic": "critical",
		"type": "default"
	  },
	  {
		"string": "FileNotFoundException",
		"topic": "critical",
		"type": "default"
	  },
	  {
		"string": "ArrayIndexOutOfBounds",
		"topic": "critical",
		"type": "default"
	  },
	  {
		"string": "NullPointerException",
		"topic": "critical",
		"type": "default"
	  },
	  {
		"string": "StringIndexOutOfBoundsException",
		"topic": "critical",
		"type": "default"
	  },
	  {
		"string": "FATAL",
		"topic": "critical",
		"type": "default"
	  },
	  {
		"string": "SEVERE",
		"topic": "critical",
		"type": "default"
	  },
	  {
		"string": "NoClassDefFoundError",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "NoSuchMethodFoundError",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "NumberFormatException",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "IllegalArgumentException",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "ParseException",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "SQLException",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "ArithmeticException",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "status=404",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "status=500",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "EXCEPTION",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "ERROR",
		"topic": "error",
		"type": "default"
	  },
	  {
		"string": "WARN",
		"topic": "warn",
		"type": "default"
	  }
	]
  }`

// Check few conditions pre-analysis
func (metric *OPSMXMetric) basicChecks() error {
	if metric.LifetimeMinutes == 0 && metric.EndTime == "" {
		return errors.New("provider config map validation error: provide either lifetimeMinutes or end time")
	}
	if metric.CanaryStartTime != metric.BaselineStartTime && metric.LifetimeMinutes == 0 {
		return errors.New("provider config map validation error: both canaryStartTime and baselineStartTime should be kept same while using endTime argument for analysis")
	}
	if metric.LifetimeMinutes != 0 && metric.LifetimeMinutes < 3 {
		return errors.New("provider config map validation error: lifetimeMinutes cannot be less than 3 minutes")
	}
	if metric.IntervalTime != 0 && metric.IntervalTime < 3 {
		return errors.New("provider config map validation error: intervalTime cannot be less than 3 minutes")
	}
	if metric.LookBackType != "" && metric.IntervalTime == 0 {
		return errors.New("provider config map validation error: intervalTime should be given along with lookBackType to perform interval analysis")
	}
	return nil
}

// Return epoch values of the specific time provided along with lifetimeMinutes for the Run
func (metric *OPSMXMetric) getTimeVariables() error {

	var canaryStartTime string
	var baselineStartTime string
	tm := time.Now()

	if metric.CanaryStartTime == "" {
		canaryStartTime = fmt.Sprintf("%d", tm.UnixNano()/int64(time.Millisecond))
	} else {
		tsStart, err := time.Parse(time.RFC3339, metric.CanaryStartTime)
		if err != nil {
			errorMsg := fmt.Sprintf("provider config map validation error: Error in parsing canaryStartTime: %v", err)
			return errors.New(errorMsg)
		}
		canaryStartTime = fmt.Sprintf("%d", tsStart.UnixNano()/int64(time.Millisecond))
	}

	if metric.BaselineStartTime == "" {
		baselineStartTime = fmt.Sprintf("%d", tm.UnixNano()/int64(time.Millisecond))
	} else {
		tsStart, err := time.Parse(time.RFC3339, metric.BaselineStartTime)
		if err != nil {
			errorMsg := fmt.Sprintf("provider config map validation error: Error in parsing baselineStartTime: %v", err)
			return errors.New(errorMsg)
		}
		baselineStartTime = fmt.Sprintf("%d", tsStart.UnixNano()/int64(time.Millisecond))
	}

	//If lifetimeMinutes not given calculate using endTime
	if metric.LifetimeMinutes == 0 {
		tsEnd, err := time.Parse(time.RFC3339, metric.EndTime)
		if err != nil {
			errorMsg := fmt.Sprintf("provider config map validation error: Error in parsing endTime: %v", err)
			return errors.New(errorMsg)
		}
		if metric.CanaryStartTime != "" && metric.CanaryStartTime > metric.EndTime {
			err := errors.New("provider config map validation error: canaryStartTime cannot be greater than endTime")
			return err
		}
		tsStart := tm
		if metric.CanaryStartTime != "" {
			tsStart, _ = time.Parse(time.RFC3339, metric.CanaryStartTime)
		}
		tsDifference := tsEnd.Sub(tsStart)
		min, _ := time.ParseDuration(tsDifference.String())
		metric.LifetimeMinutes = int(roundFloat(min.Minutes(), 0))
	}
	metric.BaselineStartTime = baselineStartTime
	metric.CanaryStartTime = canaryStartTime
	return nil
}

func getScopeValues(scope string) (string, error) {
	splitScope := strings.Split(scope, ",")
	for i, items := range splitScope {
		if strings.Contains(items, "{{env.") {
			extrctVal := strings.Split(items, "{{env.")
			extractkey := strings.Split(extrctVal[1], "}}")
			podName, ok := os.LookupEnv(extractkey[0])
			if !ok {
				err := fmt.Sprintf("analysisTemplate validation error: environment variable %s not set", extractkey[0])
				return "", errors.New(err)
			}
			old := fmt.Sprintf("{{env.%s}}", extractkey[0])
			testresult := strings.Replace(items, old, podName, 1)
			splitScope[i] = testresult
		}
	}
	scopeValue := strings.Join(splitScope, ",")
	return scopeValue, nil
}

func getTemplateDataYaml(templateFileData []byte, template string, templateType string, ScopeVariables string) ([]byte, error) {
	if templateType == "LOG" {
		var logdata LogTemplateYaml
		if err := yaml.Unmarshal([]byte(templateFileData), &logdata); err != nil {
			errorMessage := fmt.Sprintf("gitops '%s' template config map validation error: %v", template, err)
			return nil, errors.New(errorMessage)
		}
		logdata.TemplateName = template
		logdata.FilterKey = ScopeVariables
		if len(logdata.Tags) >= 1 {
			logdata.TagEnabled = true
		}

		var defaults LogTemplateYaml
		err := json.Unmarshal([]byte(DefaultsErrorTopicsJson), &defaults)
		if err != nil {
			return nil, err
		}

		var defaultErrorString []string
		defaultErrorStringMapType := make(map[string]string)
		for _, items := range defaults.ErrorTopics {
			defaultErrorStringMapType[items.ErrorStrings] = items.Topic
			defaultErrorString = append(defaultErrorString, items.ErrorStrings)
		}

		var errorStringsAvailable []string

		for i, items := range logdata.ErrorTopics {
			errorStringsAvailable = append(errorStringsAvailable, items.ErrorStrings)

			if isExists(defaultErrorString, items.ErrorStrings) {
				if items.Topic == defaultErrorStringMapType[items.ErrorStrings] {
					logdata.ErrorTopics[i].Type = "default"
				} else {
					logdata.ErrorTopics[i].Type = "custom"
				}
			}
		}

		if !logdata.DisableDefaultsErrorTopics {
			log.Info("loading defaults tags for log template")
			for _, items := range defaults.ErrorTopics {
				if !isExists(errorStringsAvailable, items.ErrorStrings) {
					logdata.ErrorTopics = append(logdata.ErrorTopics, items)
				}
			}
		}
		if logdata.ErrorTopics == nil {
			logdata.ErrorTopics = make([]errorTopics, 0)
		}
		log.Info("processed template and converting to json", logdata)
		return json.Marshal(logdata)
	}

	metricStruct, err := processYamlMetrics(templateFileData, template, ScopeVariables)
	if err != nil {
		return nil, err
	}
	return json.Marshal(metricStruct)

}

func getTemplateData(client http.Client, secretData map[string]string, template string, templateType string, basePath string, ScopeVariables string) (string, error) {
	log.Info("processing gitops template", template)
	var templateData string
	templatePath := filepath.Join(basePath, "templates/")
	path := filepath.Join(templatePath, template)
	templateFileData, err := os.ReadFile(path)
	if err != nil {
		errorMsg := fmt.Sprintf("gitops '%s' template config map validation error: %v\n Action Required: Template has to be mounted on '/etc/config/templates' in AnalysisTemplate and must carry data element '%s'", template, err, template)
		err = errors.New(errorMsg)
		return "", err
	}
	log.Info("checking if json or yaml for template ", template)
	if !isJSON(string(templateFileData)) {
		log.Info("template not recognized in json format")
		templateFileData, err = getTemplateDataYaml(templateFileData, template, templateType, ScopeVariables)
		log.Info("json for template ", template, string(templateFileData))
		if err != nil {
			return "", err
		}
	} else {
		checktemplateName := gjson.Get(string(templateFileData), "templateName")
		if checktemplateName.String() == "" {
			errmessage := fmt.Sprintf("gitops '%s' template config map validation error: template name not provided inside json", template)
			return "", errors.New(errmessage)
		}
		if template != checktemplateName.String() {
			errmessage := fmt.Sprintf("gitops '%s' template config map validation error: Mismatch between templateName and data.%s key", template, template)
			return "", errors.New(errmessage)
		}
	}

	sha1Code := generateSHA1(string(templateFileData))
	tempLink := fmt.Sprintf(templateApi, sha1Code, templateType, template)
	s := []string{secretData["opsmxIsdUrl"], tempLink}
	templateUrl := strings.Join(s, "")

	log.Debug("sending a GET request to gitops API")
	data, _, _, err := makeRequest(client, "GET", templateUrl, "", secretData["user"])
	if err != nil {
		return "", err
	}
	var templateVerification bool
	err = json.Unmarshal(data, &templateVerification)
	if err != nil {
		errorMessage := fmt.Sprintf("analysis Error: Expected bool response from gitops verifyTemplate response  Error: %v. Action: Check endpoint given in secret/providerConfig.", err)
		return "", errors.New(errorMessage)
	}
	templateData = sha1Code
	var templateCheckSave map[string]interface{}
	if !templateVerification {
		log.Debug("sending a POST request to gitops API")
		data, _, _, err = makeRequest(client, "POST", templateUrl, string(templateFileData), secretData["user"])
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
			err = fmt.Errorf("gitops '%s' template config map validation error: %s", template, errorss)
			return "", err
		}
	}
	return templateData, nil
}

func (metric *OPSMXMetric) generatePayload(c *RpcPlugin, secretData map[string]string, basePath string) (string, error) {
	var intervalTime string
	if metric.IntervalTime != 0 {
		intervalTime = fmt.Sprintf("%d", metric.IntervalTime)
	}

	var opsmxdelay string
	if metric.Delay != 0 {
		opsmxdelay = fmt.Sprintf("%d", metric.Delay)
	}
	var services []string
	//Generate the payload
	payload := jobPayload{
		Application: metric.Application,
		SourceName:  secretData["sourceName"],
		SourceType:  secretData["cdIntegration"],
		CanaryConfig: canaryConfig{
			LifetimeMinutes: fmt.Sprintf("%d", metric.LifetimeMinutes),
			LookBackType:    metric.LookBackType,
			IntervalTime:    intervalTime,
			Delays:          opsmxdelay,
			CanaryHealthCheckHandler: canaryHealthCheckHandler{
				MinimumCanaryResultScore: fmt.Sprintf("%d", metric.Pass),
			},
			CanarySuccessCriteria: canarySuccessCriteria{
				CanaryResultScore: fmt.Sprintf("%d", metric.Pass),
			},
		},
		CanaryDeployments: []canaryDeployments{},
	}
	if metric.Services != nil || len(metric.Services) != 0 {
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
		for i, item := range metric.Services {
			valid := false
			serviceName := fmt.Sprintf("service%d", i+1)
			if item.ServiceName != "" {
				serviceName = item.ServiceName
			}
			if isExists(services, serviceName) {
				errorMsg := fmt.Sprintf("provider config map validation error: serviceName '%s' mentioned in provider Config exists more than once", serviceName)
				return "", errors.New(errorMsg)
			}
			services = append(services, serviceName)
			gateName := fmt.Sprintf("gate%d", i+1)
			if item.LogScopeVariables == "" && item.BaselineLogScope != "" || item.LogScopeVariables == "" && item.CanaryLogScope != "" {
				errorMsg := fmt.Sprintf("provider config map validation error: missing log Scope placeholder for the provided baseline/canary of service '%s'", serviceName)
				err := errors.New(errorMsg)
				if err != nil {
					return "", err
				}
			}
			//For Log Analysis is to be added in analysis-run
			if item.LogScopeVariables != "" {
				//Check if no baseline or canary
				if item.BaselineLogScope != "" && item.CanaryLogScope == "" {
					errorMsg := fmt.Sprintf("provider config map validation error: missing canary for log analysis of service '%s'", serviceName)
					err := errors.New(errorMsg)
					if err != nil {
						return "", err
					}
				}
				//Check if the number of placeholders provided dont match
				if len(strings.Split(item.LogScopeVariables, ",")) != len(strings.Split(item.BaselineLogScope, ",")) || len(strings.Split(item.LogScopeVariables, ",")) != len(strings.Split(item.CanaryLogScope, ",")) {
					errorMsg := fmt.Sprintf("provider config map validation error: mismatch in number of log scope variables and baseline/canary log scope of service '%s'", serviceName)
					err := errors.New(errorMsg)
					if err != nil {
						return "", err
					}
				}
				if item.LogTemplateName == "" && metric.GlobalLogTemplate == "" {
					errorMsg := fmt.Sprintf("provider config map validation error: provide either a service specific log template or global log template for service '%s'", serviceName)
					err := errors.New(errorMsg)
					if err != nil {
						return "", err
					}
				}

				baslineLogScope, errors := getScopeValues(item.BaselineLogScope)
				if errors != nil {
					return "", errors
				}
				//Add mandatory field for baseline
				deployment.Baseline.Log[serviceName] = map[string]string{
					item.LogScopeVariables: baslineLogScope,
					"serviceGate":          gateName,
				}

				canaryLogScope, errors := getScopeValues(item.CanaryLogScope)
				if errors != nil {
					return "", errors
				}
				//Add mandatory field for canary
				deployment.Canary.Log[serviceName] = map[string]string{
					item.LogScopeVariables: canaryLogScope,
					"serviceGate":          gateName,
				}

				var tempName string
				tempName = item.LogTemplateName
				if item.LogTemplateName == "" {
					tempName = metric.GlobalLogTemplate
				}

				//Add service specific templateName
				deployment.Baseline.Log[serviceName]["template"] = tempName
				deployment.Canary.Log[serviceName]["template"] = tempName

				var templateData string
				var err error
				if metric.GitOPS && item.LogTemplateVersion == "" {
					templateData, err = getTemplateData(c.client, secretData, tempName, "LOG", basePath, item.LogScopeVariables)
					if err != nil {
						return "", err
					}
				}

				if metric.GitOPS && templateData != "" && item.LogTemplateVersion == "" {
					deployment.Baseline.Log[serviceName]["templateSha1"] = templateData
					deployment.Canary.Log[serviceName]["templateSha1"] = templateData
				}
				//Add non-mandatory field of Templateversion if provided
				if item.LogTemplateVersion != "" {
					deployment.Baseline.Log[serviceName]["templateVersion"] = item.LogTemplateVersion
					deployment.Canary.Log[serviceName]["templateVersion"] = item.LogTemplateVersion
				}
				valid = true
			}

			if item.MetricScopeVariables == "" && item.BaselineMetricScope != "" || item.MetricScopeVariables == "" && item.CanaryMetricScope != "" {
				errorMsg := fmt.Sprintf("provider config map validation error: missing metric Scope placeholder for the provided baseline/canary of service '%s'", serviceName)
				err := errors.New(errorMsg)
				if err != nil {
					return "", err
				}
			}
			//For metric analysis is to be added in analysis-run
			if item.MetricScopeVariables != "" {
				//Check if no baseline or canary
				if item.BaselineMetricScope == "" || item.CanaryMetricScope == "" {
					errorMsg := fmt.Sprintf("provider config map validation error: missing baseline/canary for metric analysis of service '%s'", serviceName)
					err := errors.New(errorMsg)
					if err != nil {
						return "", err
					}
				}
				//Check if the number of placeholders provided dont match
				if len(strings.Split(item.MetricScopeVariables, ",")) != len(strings.Split(item.BaselineMetricScope, ",")) || len(strings.Split(item.MetricScopeVariables, ",")) != len(strings.Split(item.CanaryMetricScope, ",")) {
					errorMsg := fmt.Sprintf("provider config map validation error: mismatch in number of metric scope variables and baseline/canary metric scope of service '%s'", serviceName)
					err := errors.New(errorMsg)
					if err != nil {
						return "", err
					}
				}
				if item.MetricTemplateName == "" && metric.GlobalMetricTemplate == "" {
					errorMsg := fmt.Sprintf("provider config map validation error: provide either a service specific metric template or global metric template for service: %s", serviceName)
					err := errors.New(errorMsg)
					if err != nil {
						return "", err
					}
				}

				baselineMetricScope, errors := getScopeValues(item.BaselineMetricScope)
				if errors != nil {
					return "", errors
				}
				//Add mandatory field for baseline
				deployment.Baseline.Metric[serviceName] = map[string]string{
					item.MetricScopeVariables: baselineMetricScope,
					"serviceGate":             gateName,
				}

				canaryMetricScope, errors := getScopeValues(item.CanaryMetricScope)
				if errors != nil {
					return "", errors
				}
				//Add mandatory field for canary
				deployment.Canary.Metric[serviceName] = map[string]string{
					item.MetricScopeVariables: canaryMetricScope,
					"serviceGate":             gateName,
				}

				var tempName string
				tempName = item.MetricTemplateName
				if item.MetricTemplateName == "" {
					tempName = metric.GlobalMetricTemplate
				}

				//Add templateName
				deployment.Baseline.Metric[serviceName]["template"] = tempName
				deployment.Canary.Metric[serviceName]["template"] = tempName

				var templateData string
				var err error
				if metric.GitOPS && item.MetricTemplateVersion == "" {
					templateData, err = getTemplateData(c.client, secretData, tempName, "METRIC", basePath, item.MetricScopeVariables)
					if err != nil {
						return "", err
					}
				}

				if metric.GitOPS && templateData != "" && item.MetricTemplateVersion == "" {
					deployment.Baseline.Metric[serviceName]["templateSha1"] = templateData
					deployment.Canary.Metric[serviceName]["templateSha1"] = templateData
				}

				//Add non-mandatory field of Template Version if provided
				if item.MetricTemplateVersion != "" {
					deployment.Baseline.Metric[serviceName]["templateVersion"] = item.MetricTemplateVersion
					deployment.Canary.Metric[serviceName]["templateVersion"] = item.MetricTemplateVersion
				}
				valid = true

			}
			//Check if no logs or metrics were provided
			if !valid {
				err := errors.New("provider config map validation error: at least one of log or metric context must be provided")
				if err != nil {
					return "", err
				}
			}
		}
		payload.CanaryDeployments = append(payload.CanaryDeployments, deployment)
	} else {
		//Check if no services were provided
		err := errors.New("provider config map validation error: no services provided")
		return "", err
	}
	buffer, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(buffer), err
}

// Evaluate canaryScore and accordingly set the AnalysisPhase
func evaluateResult(score int, pass int) v1alpha1.AnalysisPhase {
	if score >= pass {
		return v1alpha1.AnalysisPhaseSuccessful
	}
	return v1alpha1.AnalysisPhaseFailed
}

// Extract the canaryScore and evaluateResult
func (metric *OPSMXMetric) processResume(data []byte) (v1alpha1.AnalysisPhase, string, error) {
	var (
		canaryScore string
		result      map[string]interface{}
		finalScore  map[string]interface{}
	)

	err := json.Unmarshal(data, &result)
	if err != nil {
		errorMessage := fmt.Sprintf("analysis Error: Error in post processing canary Response. Error: %v", err)
		return "", "", errors.New(errorMessage)
	}
	jsonBytes, _ := json.MarshalIndent(result["canaryResult"], "", "   ")
	err = json.Unmarshal(jsonBytes, &finalScore)
	if err != nil {
		return "", "", err
	}
	if finalScore["overallScore"] == nil {
		canaryScore = "0"
	} else {
		canaryScore = fmt.Sprintf("%v", finalScore["overallScore"])
	}

	var score int
	// var err error
	if strings.Contains(canaryScore, ".") {
		floatScore, err := strconv.ParseFloat(canaryScore, 64)
		if err != nil {
			return "", "", err
		}
		score = int(roundFloat(floatScore, 0))
	} else {
		score, err = strconv.Atoi(canaryScore)
		if err != nil {
			return "", "", err
		}
	}

	Phase := evaluateResult(score, int(metric.Pass))
	return Phase, fmt.Sprintf("%v", score), nil
}

func (metric *OPSMXMetric) getDataSecret(c *RpcPlugin, ar *v1alpha1.AnalysisRun) (map[string]string, error) {

	secretData := map[string]string{}
	secretName := defaultSecretName

	secret, err := c.kubeclientset.CoreV1().Secrets(ar.Namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return secretData, err
	}

	secretUser, ok := secret.Data["user"]
	if !ok {
		err = errors.New("opsmx profile secret validation error: `user` key not present in the secret file\n Action Required: secret file must carry data element 'user'")
		return secretData, err
	}

	secretOpsmxIsdUrl, ok := secret.Data["opsmxIsdUrl"]
	if !ok {
		err = errors.New("opsmx profile secret validation error: `opsmxIsdUrl` key not present in the secret file\n Action Required: secret file must carry data element 'opsmxIsdUrl'")
		return secretData, err
	}
	secretsourcename, ok := secret.Data["sourceName"]
	if !ok {
		err = errors.New("opsmx profile secret validation error: `sourceName` key not present in the secret file\n Action Required: secret file must carry data element 'sourceName'")
		return secretData, err
	}
	secretcdintegration, ok := secret.Data["cdIntegration"]
	if !ok {
		err = errors.New("opsmx profile secret validation error: `cdIntegration` key not present in the secret file\n Action Required: secret file must carry data element 'cdIntegration'")
		return secretData, err
	}

	opsmxIsdURL := metric.OpsmxIsdUrl
	if opsmxIsdURL == "" {
		opsmxIsdURL = string(secretOpsmxIsdUrl)
	}
	secretData["opsmxIsdUrl"] = opsmxIsdURL

	user := metric.User
	if user == "" {
		user = string(secretUser)
	}
	secretData["user"] = user

	var cdIntegration string
	if string(secretcdintegration) == "true" {
		cdIntegration = cdIntegrationArgoCD
	} else if string(secretcdintegration) == "false" {
		cdIntegration = cdIntegrationArgoRollouts
	} else {
		err := errors.New("opsmx profile secret validation error: cdIntegration should be either true or false")
		return nil, err
	}
	secretData["cdIntegration"] = cdIntegration

	secretData["sourceName"] = string(secretsourcename)

	return secretData, nil
}
