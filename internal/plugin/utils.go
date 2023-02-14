package plugin

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
)

// func getNamespace() string {
// 	if dataByte, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
// 		ns := strings.TrimSpace(string(dataByte))
// 		if len(ns) > 0 {
// 			return ns
// 		}
// 	}
// 	return ""
// }

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

// TODO: Refactor
func makeRequest(client http.Client, requestType string, url string, body string, user string) ([]byte, string, string, error) {
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
		return []byte{}, "", "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return []byte{}, "", "", err
	}
	var urlScore string
	var urlToken string
	if strings.Contains(url, "registerCanary") {
		urlScore = res.Header.Get("Location")
		urlToken = res.Header.Get("x-opsmx-report-token")
	}
	return data, urlScore, urlToken, err
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

func isJSON(s string) bool {
	var j map[string]interface{}
	if err := json.Unmarshal([]byte(s), &j); err != nil {
		return false
	}
	return true
}

func generateSHA1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	sha1_hash := hex.EncodeToString(h.Sum(nil))
	return sha1_hash
}

// func isUrl(str string) bool {
// 	u, err := url.Parse(str)
// 	if err != nil {
// 		log.Errorf("Error in parsing url: %v", err)
// 	}
// 	log.Infof("Parsed url: %v", u)
// 	return err == nil && u.Scheme != "" && u.Host != ""
// }

func getTemplateUrl(opsmxUrl string, sha1Code string, templateType string, templateName string) (string, error) {
	urlParse, err := url.Parse(opsmxUrl)
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
