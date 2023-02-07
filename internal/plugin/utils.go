package plugin

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"net/http"
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
