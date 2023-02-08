package handlers

import (
	"fmt"
	"io"
	"net/http"
	"proxy-api-server/log"
	"proxy-api-server/util"
	"strings"
)

func GrafanaApiHandler(w http.ResponseWriter, r *http.Request) {

	grafanaUrl := r.URL.Query().Get("grafanaUrl")
	apiKey := r.URL.Query().Get("apiKey")
	if grafanaUrl == "" {
		log.Error("Grafana url not provided")
		http.Error(w, fmt.Sprintf("Grafana url not provided"), http.StatusBadRequest)
		return
	} else if apiKey == "" {
		log.Error("Grafana api key (userId:password) not provided")
		http.Error(w, fmt.Sprintf("Grafana api key (userId:password) not provided"), http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		util.Error("Cannot read request body.", err)
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	payload := strings.NewReader(string(body))
	resPbody, statusCode, err := util.HandleHttpRequest("POST", grafanaUrl, apiKey, payload)
	if err != nil {
		util.Error("Http request failed: ", err)
		http.Error(w, fmt.Sprintf("%s", err), statusCode)
		return
	}
	//fmt.Println(string(resPbody))

	_, _ = w.Write(resPbody)
}
