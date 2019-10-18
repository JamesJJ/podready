package podready

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type HTTPHeader struct {
	Name  string
	Value string
}

func Wait() {

	verbose := false
	if verboseFromEnv := os.Getenv("PODREADY_VERBOSE"); verboseFromEnv == "true" {
		verbose = true
	}

	// If we are not running in K8S, return immediately
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		logTrue(verbose, "PodReady: Not in K8S\n")
		return
	}

	logTrue(verbose, "PodReady: ")
	start := now()

	for now() < start+20000 {

		istio := waitIstio()

		if true && istio { // This to join several future tests; not just for ISTIO
			break
		}

		time.Sleep(600 * time.Millisecond)
	}
	logTrue(verbose, fmt.Sprintf("Waited %.2f seconds\n", float32(now()-start)/1000.0))
}

func waitIstio() bool {
	if os.Getenv("PODREADY_DO_NOT_WAIT_FOR_ISTIO") == "true" {
		return true
	}

	istioReadyUrl := "http://localhost:15020/healthz/ready"

	if istioReadyUrlFromEnv := os.Getenv("PODREADY_ISTIO_READY_URL"); istioReadyUrlFromEnv != "" {
		istioReadyUrl = istioReadyUrlFromEnv
	}

	reqHeaders := []HTTPHeader{
		HTTPHeader{Name: "User-Agent", Value: "PodReady/1.0"},
	}
	code, _ := httpCheck(istioReadyUrl, &reqHeaders)
	return (code == http.StatusOK)
}

func logTrue(toggle bool, text string) {
	if toggle {
		fmt.Printf(text)
	}
}

func now() int {
	return int(time.Now().UTC().UnixNano() / 1000000)
}

// Return a truncated, single line, with no leading or trailing whitespace
func maxString(s string, maxLen int, singleLineTrimWhitespace bool) string {

	if len(s) > maxLen {
		s = s[:maxLen]
	}
	for len(s) > maxLen || !utf8.ValidString(s) {
		s = s[:len(s)-1] // remove a byte
	}
	if singleLineTrimWhitespace {
		re := regexp.MustCompile(`[\r\n]+`)
		s = strings.TrimSpace(re.ReplaceAllString(s, " "))
	}
	return s
}

// GET a url and return status code and a
//  truncated response body
func httpCheck(url string, Headers *[]HTTPHeader) (int, string) {
	code := 0

	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 2 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 2 * time.Second,
	}
	var netClient = &http.Client{
		Timeout:   time.Second * 2,
		Transport: netTransport,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, ""
	}
	for _, h := range *Headers {
		req.Header.Add(h.Name, h.Value)
	}
	resp, err := netClient.Do(req)
	if err != nil {
		return 0, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode > 0 {
		code = resp.StatusCode
	}
	if code != http.StatusOK {
		return code, ""
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return code, ""
	}

	infoString := maxString(string(body), 128, true)
	return code, infoString
}
