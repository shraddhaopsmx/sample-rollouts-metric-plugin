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

	log "github.com/sirupsen/logrus"
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

func isUrl(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		log.Errorf("Error in parsing url: %v", err)
	}
	log.Infof("Parsed url: %v", u)
	return err == nil && u.Scheme != "" && u.Host != ""
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
