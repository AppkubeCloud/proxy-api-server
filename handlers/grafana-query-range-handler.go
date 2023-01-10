package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/grafana-tools/sdk"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"proxy-api-server/models"
	"proxy-api-server/util"
)

// GrafanaQueryRangeHandler is used for handling Grafana Range queries
func GrafanaQueryRangeHandler(w http.ResponseWriter, req *http.Request) {
	// if req.Method != http.MethodGet {
	// 	w.WriteHeader(http.StatusNotFound)
	// 	return
	// }

	reqQuery := req.URL.Query()
	client := util.NewGrafanaClient()
	data, err := GrafanaQueryRange(client, req.Context(), reqQuery.Get("url"), reqQuery.Get("api-key"), &reqQuery)
	if err != nil {
		// h.log.Error(ErrGrafanaQuery(err))
		// http.Error(w, ErrGrafanaQuery(err).Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func GrafanaQueryRange(g *models.GrafanaClient, ctx context.Context, BaseURL, APIKey string, queryData *url.Values) ([]byte, error) {
	if queryData == nil {
		return nil, errors.New("query data passed is nil")
	}

	c, err := sdk.NewClient(BaseURL, APIKey, g.HttpClient)
	if err != nil {
		return nil, util.CommonError(err)
	}

	ds, err := c.GetDatasourceByName(ctx, queryData.Get("ds"))
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	var reqURL string
	if g.PromMode {
		reqURL = fmt.Sprintf("%s/api/v1/query_range", BaseURL)
	} else {
		reqURL = fmt.Sprintf("%s/api/datasources/proxy/%d/api/v1/query_range", BaseURL, ds.ID)
	}

	newURL, _ := url.Parse(reqURL)
	q := url.Values{}
	q.Set("query", queryData.Get("query"))
	q.Set("start", queryData.Get("start"))
	q.Set("end", queryData.Get("end"))
	q.Set("step", queryData.Get("step"))
	newURL.RawQuery = q.Encode()
	queryURL := newURL.String()
	data, err := g.MakeRequest(ctx, queryURL, APIKey)
	if err != nil {
		return nil, util.CommonError(err)
	}
	return data, nil
}
