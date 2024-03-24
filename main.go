package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

var (
	baseUrl string
	listen  string
)

func init() {
	flag.StringVar(&baseUrl, "baseurl", "", "-baseurl=http://remote.example.com")
	flag.StringVar(&listen, "l", "", "-l :8080")

	flag.Parse()
}

func main() {
	checkParameters()

	log.Println("starting...")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if err := http.ListenAndServe(listen, mux{
		client: &http.Client{
			Transport: tr,
		},
	}); err != nil {
		panic(err)
	}
}

type mux struct {
	client *http.Client
}

func (h mux) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//log.Println("RequestURI:", req.RequestURI)
	var logReq = req
	var logRes *http.Response
	defer func() {
		logHttpRequest(logReq, logRes)
	}()

	newReq, err := http.NewRequest(req.Method, fmt.Sprint(baseUrl, req.RequestURI), req.Body)
	if err != nil {
		res.WriteHeader(http.StatusBadGateway)
		log.Println("prepare request failed:", err)
		return
	}
	copyHeader(req.Header, newReq.Header)
	newRes, err := h.client.Do(newReq)
	if err != nil {
		res.WriteHeader(http.StatusBadGateway)
		log.Println("send request failed:", err)
		return
	}
	logRes = newRes
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(newRes.Body)
	copyHeader(newRes.Header, res.Header())
	res.WriteHeader(newRes.StatusCode)
	if _, err := io.Copy(res, newRes.Body); err != nil {
		log.Println("handle response body failed:", err)
		return
	}
}

func logHttpRequest(req *http.Request, res *http.Response) {
	headerJson, err := json.MarshalIndent(req.Header, "", "  ")
	if err != nil {
		headerJson = []byte(fmt.Sprint(err))
	}
	resHeaderJson, err := json.MarshalIndent(res.Header, "", "  ")
	if err != nil {
		resHeaderJson = []byte(fmt.Sprint(err))
	}
	s := fmt.Sprintf(`Request Details:
--------------------------> Request Method: %s  URI: %s
--------------------------> Headers:
%s
--------------------------> Response Status Code: %d
--------------------------> Headers:
%s
<<<<=======================
`, req.Method, req.RequestURI, string(headerJson), res.StatusCode, string(resHeaderJson))
	log.Println(s)
}

func checkParameters() {
	if baseUrl == "" {
		panic(errors.New("base.url is empty"))
	}
	if listen == "" {
		panic(errors.New("listen address is empty"))
	}
}

func copyHeader(src, dst http.Header) {
	for k, v := range src {
		for _, vd := range v {
			dst.Add(k, vd)
		}
	}
}
