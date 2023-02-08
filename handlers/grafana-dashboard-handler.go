package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/grafana-tools/sdk"
	"net/http"
	"proxy-api-server/helpers"
	"proxy-api-server/log"
	"proxy-api-server/models"
	"proxy-api-server/util"
	"strings"
)

func GetGrafanaDashbordByUidHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("Starting GetGrafanaDashbordByUidHandler...")
	grafanaUrl := r.URL.Query().Get("grafanaUrl")
	apiKey := r.URL.Query().Get("apiKey")
	uid := r.URL.Query().Get("uid")
	if grafanaUrl == "" {
		log.Error("Grafana url not provided")
		http.Error(w, fmt.Sprintf("Grafana url not provided"), http.StatusBadRequest)
		return
	} else if apiKey == "" {
		log.Error("Grafana api key (userId:password) not provided")
		http.Error(w, fmt.Sprintf("Grafana api key (userId:password) not provided"), http.StatusBadRequest)
		return
	}

	client := util.NewGrafanaClient()
	req := &http.Request{
		Method: "GET",
	}
	if strings.HasSuffix(grafanaUrl, "/") {
		grafanaUrl = strings.Trim(grafanaUrl, "/")
	}
	grafanaSdkClient, err := sdk.NewClient(grafanaUrl, apiKey, client.HttpClient)
	if err != nil {
		log.Error("Error in grafana sdk client: ", err)
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}

	br, brProp, err := grafanaSdkClient.GetDashboardByUID(req.Context(), uid)
	if err != nil {
		log.Errorf("Error in getting dashboard by UID: %s, Error : %s", uid, err)
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	grafBoard := &models.GrafanaDashboard{
		Dashboard: &br,
		Meta:      &brProp,
	}
	//json.NewEncoder(w).Encode(grafBoard)

	if err := json.NewEncoder(w).Encode(grafBoard); err != nil {
		util.Error("Http request failed: ", err)
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}

	log.Info("GetGrafanaDashbordByUidHandler completed")

}

func GrafanaDashboardHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("Starting GrafanaDashboardHandler")
	grafanaUrl := r.URL.Query().Get("grafanaUrl")
	apiKey := r.URL.Query().Get("apiKey")
	if grafanaUrl == "" {
		log.Error("Grafana url not provided")
		return
	} else if apiKey == "" {
		log.Error("Grafana api key (userId:password) not provided")
		return
	}
	pref := &models.Preference{
		Grafana: &models.Grafana{
			GrafanaURL:    grafanaUrl,
			GrafanaAPIKey: apiKey,
		},
	}
	user := &models.User{
		UserID:    "admin",
		FirstName: "admin",
	}

	b := GrafanaBoardsHandler(pref, user)
	json.NewEncoder(w).Encode(b)
	log.Info("GrafanaDashboardHandler completed")
}

func GrafanaBoardsHandler(prefObj *models.Preference, user *models.User) []*models.GrafanaBoard {
	// if req.Method != http.MethodGet && req.Method != http.MethodPost {
	// 	w.WriteHeader(http.StatusNotFound)
	// 	return
	// }

	// No POST for now. Commented
	// if req.Method == http.MethodPost {
	// 	h.SaveSelectedGrafanaBoardsHandler(w, req, prefObj, user, p)
	// 	return
	// }
	log.Info("Starting GrafanaBoardsHandler")
	client := util.NewGrafanaClient()
	if prefObj.Grafana == nil || prefObj.Grafana.GrafanaURL == "" {
		// h.log.Error(ErrGrafanaConfig)
		// http.Error(w, "Invalid grafana endpoint", http.StatusBadRequest)
		log.Error("Grafana url not provided")
		return nil
	}
	req := &http.Request{
		Method: "GET",
	}

	if err := helpers.Validate(client, req.Context(), prefObj.Grafana.GrafanaURL, prefObj.Grafana.GrafanaAPIKey); err != nil {
		// h.log.Error(ErrGrafanaScan(err))
		// http.Error(w, "Unable to connect to grafana", http.StatusInternalServerError)
		log.Error("Unable to connect to grafana")
		return nil
	}

	var dashboardSearch = "" //req.URL.Query().Get("dashboardSearch")
	boards, err := GetGrafanaBoards(client, req.Context(), prefObj.Grafana.GrafanaURL, prefObj.Grafana.GrafanaAPIKey, dashboardSearch)
	if err != nil {
		// h.log.Error(ErrGrafanaBoards(err))
		// http.Error(w, "unable to get grafana boards", http.StatusInternalServerError)
		return nil
	}
	// fmt.Println(boards)
	w := new(bytes.Buffer)
	err = json.NewEncoder(w).Encode(boards)

	if err != nil {
		// obj := "boards payload"
		// h.log.Error(ErrMarshal(err, obj))
		// http.Error(w, "Unable to marshal the boards payload", http.StatusInternalServerError)
		log.Error("Unable to marshal the boards payload: ", err)
		return nil
	}
	log.Info("GrafanaBoardsHandler completed: ")
	return boards
}

func GetGrafanaBoards(g *models.GrafanaClient, ctx context.Context, BaseURL, APIKey, dashboardSearch string) ([]*models.GrafanaBoard, error) {
	log.Info("Starting GetGrafanaBoards")
	if strings.HasSuffix(BaseURL, "/") {
		BaseURL = strings.Trim(BaseURL, "/")
	}
	c, err := sdk.NewClient(BaseURL, APIKey, g.HttpClient)
	if err != nil {
		return nil, util.CommonError(err)
	}

	boardLinks, err := c.SearchDashboards(ctx, dashboardSearch, false)
	if err != nil {
		return nil, util.CommonError(err)
	}
	boards := []*models.GrafanaBoard{}
	for _, link := range boardLinks {
		if link.Type != "dash-db" {
			continue
		}
		// TODO Need to do the unitest for Grafana helper
		board, _, err := c.GetDashboardByUID(ctx, link.UID)
		// fmt.Println("DashBoard...... ", board)
		if err != nil {
			log.Error("ERROR in calling GetDashboardByUID: ", err)
			return nil, util.DashboardError(err, link.UID)
		}
		// b, _ := json.Marshal(board)
		// logrus.Debugf("Board before foramating: %s", b)
		// fmt.Println("")
		// fmt.Printf("Board before foramating %s", b)
		grafBoard, err := helpers.ProcessBoard(g, ctx, c, &board, &link)
		if err != nil {
			return nil, err
		}
		fmt.Println()
		fmt.Println()

		// fmt.Println("grafana dsboard")
		// fmt.Println(grafBoard)
		// b, _ := json.Marshal(grafBoard)
		// logrus.Debugf("Board after foramating: %s", b)
		// fmt.Println("")
		// fmt.Printf("Board after foramating %s", b)
		boards = append(boards, grafBoard)
	}
	// fmt.Println("Board after foramating ", boards)
	return boards, nil
}
