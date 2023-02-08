package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"proxy-api-server/log"
	"proxy-api-server/models"
	"proxy-api-server/util"
	"strings"

	"github.com/sirupsen/logrus"
)

func GrafanaQueryHandler(w http.ResponseWriter, r *http.Request) {

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

	prefObj := &models.Preference{
		Grafana: &models.Grafana{
			// GrafanaURL:    "http://grafana.synectiks.net",
			// GrafanaAPIKey: "admin:password",
			GrafanaURL:    grafanaUrl,
			GrafanaAPIKey: apiKey,
		},
	}
	// user := &User{
	// 	UserID:    "admin",
	// 	FirstName: "admin",
	// }

	reqQuery := r.URL.Query()

	// if prefObj.Grafana == nil || prefObj.Grafana.GrafanaURL == "" {
	// 	err := ErrGrafanaConfig
	// 	h.log.Error(err)
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }
	log.Info("Getting grafana dashboard with uid")
	client := util.NewGrafanaClient()
	data, err := GrafanaQuery(client, r.Context(), prefObj.Grafana.GrafanaURL, prefObj.Grafana.GrafanaAPIKey, &reqQuery)
	if err != nil {
		util.Error("Http request failed: ", err)
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func GrafanaQuery(g *models.GrafanaClient, ctx context.Context, BaseURL, APIKey string, queryData *url.Values) ([]byte, error) {
	if queryData == nil {
		return nil, errors.New("query data passed is nil")
	}
	query := strings.TrimSpace(queryData.Get("query"))
	dsID := queryData.Get("dsid")
	log.Info("Dashboard uid: ")
	var queryURL string
	switch {
	case strings.HasPrefix(query, "label_values("):
		val := strings.Replace(query, "label_values(", "", 1)
		val = strings.TrimSpace(strings.TrimSuffix(val, ")"))
		if strings.Contains(val, ",") {
			start := queryData.Get("start")
			end := queryData.Get("end")
			comInd := strings.LastIndex(val, ", ")
			if comInd > -1 {
				val = val[:comInd]
			}
			for key := range *queryData {
				if key != "query" && key != "dsid" && key != "start" && key != "end" {
					val1 := queryData.Get(key)
					val = strings.Replace(val, "$"+key, val1, -1)
				}
			}
			var reqURL string
			if g.PromMode {
				reqURL = fmt.Sprintf("%s/api/v1/series", BaseURL)
			} else {
				reqURL = fmt.Sprintf("%s/api/datasources/proxy/%s/api/v1/series", BaseURL, dsID)
			}
			queryURLInst, _ := url.Parse(reqURL)
			qParams := queryURLInst.Query()
			qParams.Set("match[]", val)
			if start != "" && end != "" {
				qParams.Set("start", start)
				qParams.Set("end", end)
			}
			queryURLInst.RawQuery = qParams.Encode()
			queryURL = queryURLInst.String()
		} else {
			if g.PromMode {
				queryURL = fmt.Sprintf("%s/api/v1/label/%s/values", BaseURL, val)
			} else {
				queryURL = fmt.Sprintf("%s/api/datasources/proxy/%s/api/v1/label/%s/values", BaseURL, dsID, val)
			}
		}
	case strings.HasPrefix(query, "query_result("):
		val := strings.Replace(query, "query_result(", "", 1)
		val = strings.TrimSpace(strings.TrimSuffix(val, ")"))
		for key := range *queryData {
			if key != "query" && key != "dsid" {
				val1 := queryData.Get(key)
				val = strings.Replace(val, "$"+key, val1, -1)
			}
		}
		var reqURL string
		if g.PromMode {
			reqURL = fmt.Sprintf("%s/api/v1/query", BaseURL)
		} else {
			reqURL = fmt.Sprintf("%s/api/datasources/proxy/%s/api/v1/query", BaseURL, dsID)
		}
		newURL, _ := url.Parse(reqURL)
		q := url.Values{}
		q.Set("query", val)
		newURL.RawQuery = q.Encode()
		queryURL = newURL.String()
	default:
		return json.Marshal(map[string]interface{}{
			"status": "success",
			"data":   []string{query},
		})
	}
	logrus.Debugf("derived query url: %s", queryURL)

	data, err := g.MakeRequest(ctx, queryURL, APIKey)
	if err != nil {
		return nil, errors.New("error getting data from Grafana API")
	}
	return data, nil
}
