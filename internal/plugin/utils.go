package plugin

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
)

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func makeRequest(client http.Client, requestType string, url string, body string, user string) ([]byte, string, error) {
	reqBody := strings.NewReader(body)
	req, _ := http.NewRequest(
		requestType,
		url,
		reqBody,
	)
	req.Header.Set("x-spinnaker-user", user)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return []byte{}, "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return []byte{}, "", err
	}

	var urlToken string
	if strings.Contains(url, "registerCanary") {
		urlToken = res.Header.Get("x-opsmx-report-token")
	}
	return data, urlToken, err
}

func isExists(list []string, item string) bool {
	for _, v := range list {
		if item == v {
			return true
		}
	}
	return false
}

func serviceExists(list []service, serviceName string) bool {
	for _, v := range list {
		if v.serviceName == serviceName {
			return true
		}
	}
	return false
}

func generateSHA1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	sha1_hash := hex.EncodeToString(h.Sum(nil))
	return sha1_hash
}

func getTemplateUrl(opsmxUrl string, sha1Code string, templateType string, templateName string) (string, error) {
	_url, err := url.JoinPath(opsmxUrl, templateApi)
	if err != nil {
		return "", err
	}

	urlParse, err := url.Parse(_url)
	if err != nil {
		return "", err
	}
	values := urlParse.Query()
	values.Add("sha1", sha1Code)
	values.Add("templateType", templateType)
	values.Add("templateName", templateName)
	urlParse.RawQuery = values.Encode()
	return urlParse.String(), nil
}

func processScoreResponse(data []byte) (map[string]interface{}, error) {
	var response map[string]interface{}
	var reportUrlJson map[string]interface{}
	var status map[string]interface{}
	scoreResponseMap := make(map[string]interface{})

	err := json.Unmarshal(data, &response)
	if err != nil {
		return scoreResponseMap, fmt.Errorf("analysis Error: Error in post processing canary Response: %v", err)
	}
	canaryResultBytes, err := json.MarshalIndent(response["canaryResult"], "", "   ")
	if err != nil {
		return scoreResponseMap, err
	}
	err = json.Unmarshal(canaryResultBytes, &reportUrlJson)
	if err != nil {
		return scoreResponseMap, err
	}
	statusBytes, err := json.MarshalIndent(response["status"], "", "   ")
	if err != nil {
		return scoreResponseMap, err
	}
	err = json.Unmarshal(statusBytes, &status)
	if err != nil {
		return scoreResponseMap, err
	}

	scoreResponseMap["canaryReportURL"] = reportUrlJson["canaryReportURL"]
	scoreResponseMap["intervalNo"] = reportUrlJson["intervalNo"]
	scoreResponseMap["status"] = status["status"]

	return scoreResponseMap, nil
}
