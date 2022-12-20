package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gosimple/slug"
	"github.com/grafana-tools/sdk"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"github.com/rs/cors"
)

type GrafanaBoard struct {
	URI          string                 `json:"uri,omitempty"`
	Title        string                 `json:"title,omitempty"`
	Slug         string                 `json:"slug,omitempty"`
	UID          string                 `json:"uid,omitempty"`
	OrgID        uint                   `json:"org_id,omitempty"`
	Panels       []*sdk.Panel           `json:"panels,omitempty"`
	TemplateVars []*GrafanaTemplateVars `json:"template_vars,omitempty"`
}
type GrafanaTemplateVars struct {
	Name       string             `json:"name,omitempty"`
	Query      string             `json:"query,omitempty"`
	Datasource *GrafanaDataSource `json:"datasource,omitempty"`
	Hide       uint8              `json:"hide,omitempty"`
	Value      interface{}        `json:"value,omitempty"`
}
type GrafanaDataSource struct {
	ID   uint   `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type FoundBoard struct {
	ID          uint     `json:"id"`
	UID         string   `json:"uid"`
	Title       string   `json:"title"`
	URI         string   `json:"uri"`
	URL         string   `json:"url"`
	Slug        string   `json:"slug"`
	Type        string   `json:"type"`
	Tags        []string `json:"tags"`
	IsStarred   bool     `json:"isStarred"`
	FolderID    int      `json:"folderId"`
	FolderUID   string   `json:"folderUid"`
	FolderTitle string   `json:"folderTitle"`
	FolderURL   string   `json:"folderUrl"`
}

// Preference represents the data stored in session / local DB
type Preference struct {
	Grafana *Grafana `json:"grafana,omitempty"`
}

// Grafana represents the Grafana session config
type Grafana struct {
	GrafanaURL    string `json:"grafanaURL,omitempty"`
	GrafanaAPIKey string `json:"grafanaAPIKey,omitempty"`
	// GrafanaBoardSearch string          `json:"grafanaBoardSearch,omitempty"`
	GrafanaBoards []*SelectedGrafanaConfig `json:"selectedBoardsConfigs,omitempty"`
}

// SelectedGrafanaConfig represents the selected boards, panels, and template variables
type SelectedGrafanaConfig struct {
	GrafanaBoard         *GrafanaBoard `json:"board,omitempty"`
	GrafanaPanels        []*sdk.Panel  `json:"panels,omitempty"`
	SelectedTemplateVars []string      `json:"templateVars,omitempty"`
}
type User struct {
	UserID    string `json:"user_id,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Provider  string `json:"provider,omitempty" db:"provider"`
	Email     string `json:"email,omitempty" db:"email"`
	Bio       string `json:"bio,omitempty" db:"bio"`
}
type Provider interface {
}

type GrafanaClient struct {
	httpClient *http.Client
	promMode   bool
}

type HandlerConfig struct {
	GrafanaClient         *GrafanaClient
	GrafanaClientForQuery *GrafanaClient
}

func NewGrafanaClient() *GrafanaClient {
	return NewGrafanaClientWithHTTPClient(&http.Client{
		Timeout: 25 * time.Second,
	})
}

// NewGrafanaClientWithHTTPClient returns a new GrafanaClient with the given HTTP Client
func NewGrafanaClientWithHTTPClient(client *http.Client) *GrafanaClient {
	return &GrafanaClient{
		httpClient: client,
	}
}

func handleRequests() {
	// http.HandleFunc("/grafana-dashboard", getDs)
	// log.Fatal(http.ListenAndServe(":10000", nil))

	// creates a new instance of a mux router
	myRouter := mux.NewRouter().StrictSlash(true)
	c := cors.New(cors.Options{
        AllowedOrigins:   []string{"http://localhost:3006"},
        AllowCredentials: true,
    })
    
	// replace http.HandleFunc with myRouter.HandleFunc
	myRouter.HandleFunc("/grafana-ds", getDs)
	myRouter.HandleFunc("/grafana-ds-query", GrafanaQueryHandler)
	myRouter.HandleFunc("/grafana-ds/query-range", GrafanaQueryRangeHandler)

	// finally, instead of passing in nil, we want
	// to pass in our newly created router as the second
	// argument
	log.Fatal(http.ListenAndServe(":10000", c.Handler(myRouter)))
}

func main() {
	handleRequests()
}

// func main() {
// 	fmt.Println("Starting process")
// 	getDs()
// 	fmt.Println("Process completed")
// }

func getDs(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Starting getDs")
	grafanaUrl := r.URL.Query().Get("grafanaUrl")
	apiKey := r.URL.Query().Get("apiKey")
	if grafanaUrl == "" {
		fmt.Println("Grafana url not provided")
		return
	} else if apiKey == "" {
		fmt.Println("Grafana api key (userId:password) not provided")
		return
	}
	pref := &Preference{
		Grafana: &Grafana{
			// GrafanaURL:    "http://grafana.synectiks.net",
			// GrafanaAPIKey: "admin:password",
			GrafanaURL:    grafanaUrl,
			GrafanaAPIKey: apiKey,
		},
	}
	user := &User{
		UserID:    "admin",
		FirstName: "admin",
	}

	b := GrafanaBoardsHandler(pref, user)
	json.NewEncoder(w).Encode(b)
	fmt.Println("getDs completed")
}

func GrafanaBoardsHandler(prefObj *Preference, user *User) []*GrafanaBoard {
	// if req.Method != http.MethodGet && req.Method != http.MethodPost {
	// 	w.WriteHeader(http.StatusNotFound)
	// 	return
	// }

	// No POST for now. Commented
	// if req.Method == http.MethodPost {
	// 	h.SaveSelectedGrafanaBoardsHandler(w, req, prefObj, user, p)
	// 	return
	// }
	fmt.Println("Starting GrafanaBoardsHandler")
	client := NewGrafanaClient()
	if prefObj.Grafana == nil || prefObj.Grafana.GrafanaURL == "" {
		// h.log.Error(ErrGrafanaConfig)
		// http.Error(w, "Invalid grafana endpoint", http.StatusBadRequest)
		fmt.Println("Grafana URL null. Exiting")
		return nil
	}
	req := &http.Request{
		Method: "GET",
	}

	if err := Validate(client, req.Context(), prefObj.Grafana.GrafanaURL, prefObj.Grafana.GrafanaAPIKey); err != nil {
		// h.log.Error(ErrGrafanaScan(err))
		// http.Error(w, "Unable to connect to grafana", http.StatusInternalServerError)
		fmt.Println("Unable to connect to grafana. Exiting")
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
		fmt.Println("Unable to marshal the boards payload")
		return nil
	}
	fmt.Println("GrafanaBoardsHandler completed: ", w.String())
	return boards
}

func GetGrafanaBoards(g *GrafanaClient, ctx context.Context, BaseURL, APIKey, dashboardSearch string) ([]*GrafanaBoard, error) {
	fmt.Println("Starting GetGrafanaBoards")
	if strings.HasSuffix(BaseURL, "/") {
		BaseURL = strings.Trim(BaseURL, "/")
	}
	c, err := sdk.NewClient(BaseURL, APIKey, g.httpClient)
	if err != nil {
		return nil, commonError(err)
	}

	boardLinks, err := c.SearchDashboards(ctx, dashboardSearch, false)
	if err != nil {
		return nil, commonError(err)
	}
	boards := []*GrafanaBoard{}
	for _, link := range boardLinks {
		if link.Type != "dash-db" {
			continue
		}
		// TODO Need to do the unitest for Grafana helper
		board, _, err := c.GetDashboardByUID(ctx, link.UID)
		// fmt.Println("DashBoard...... ", board)
		if err != nil {
			fmt.Println("ERROR in GetDashboardByUID")
			return nil, dashboardError(err, link.UID)
		}
		// b, _ := json.Marshal(board)
		// logrus.Debugf("Board before foramating: %s", b)
		// fmt.Println("")
		// fmt.Printf("Board before foramating %s", b)
		grafBoard, err := ProcessBoard(g, ctx, c, &board, &link)
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

func ProcessBoard(g *GrafanaClient, ctx context.Context, c *sdk.Client, board *sdk.Board, link *sdk.FoundBoard) (*GrafanaBoard, error) {
	var orgID uint
	if !g.promMode {
		org, err := c.GetActualOrg(ctx)
		if err != nil {
			return nil, commonError(err)
		}
		orgID = org.ID
	}
	grafBoard := &GrafanaBoard{
		URI:          link.URI,
		Title:        link.Title,
		UID:          board.UID,
		Slug:         slug.Make(board.Title),
		TemplateVars: []*GrafanaTemplateVars{},
		Panels:       []*sdk.Panel{},
		OrgID:        orgID,
	}
	var err error
	// fmt.Println()
	// fmt.Println("Dashboard panels::::::  ", board.Panels)
	// Process Template Variables
	tmpDsName := map[string]string{}
	if len(board.Templating.List) > 0 {
		for _, tmpVar := range board.Templating.List {
			var ds sdk.Datasource
			var dsName string

			if tmpVar.Type == "datasource" {
				dsName = cases.Title(language.Und).String(strings.ToLower(fmt.Sprint(tmpVar.Query))) // datasource name can be found in the query field
				tmpDsName[tmpVar.Name] = dsName
			} else if tmpVar.Type == "query" && tmpVar.Datasource != nil {
				tempDsStr := fmt.Sprint(tmpVar.Datasource)
				if !strings.HasPrefix(tempDsStr, "$") {
					dsName = tempDsStr
				} else {
					dsName = tmpDsName[strings.Replace(tempDsStr, "$", "", 1)]
				}
				// if !strings.HasPrefix(*tmpVar.Datasource, "$") {
				// 	dsName = *tmpVar.Datasource
				// } else {
				// 	dsName = tmpDsName[strings.Replace(*tmpVar.Datasource, "$", "", 1)]
				// }
			} else {
				err := fmt.Errorf("unable to get datasource name for tmpvar: %+#v", tmpVar)
				// logrus.Error(err)
				commonError(err)
				return nil, err
			}
			if c != nil {
				ds, err = c.GetDatasourceByName(ctx, dsName)
				if err != nil {
					return nil, dashboardError(err, "Error getting Grafana Board's Datasource")
				}
			} else {
				ds.Name = dsName
			}

			tvVal := tmpVar.Current.Text
			grafBoard.TemplateVars = append(grafBoard.TemplateVars, &GrafanaTemplateVars{
				Name:  tmpVar.Name,
				Query: fmt.Sprint(tmpVar.Query),
				Datasource: &GrafanaDataSource{
					ID:   ds.ID,
					Name: ds.Name,
				},
				Hide:  tmpVar.Hide,
				Value: tvVal,
			})
		}
	}

	//Process Board Panels
	if len(board.Panels) > 0 {
		for _, p1 := range board.Panels {
			if p1.OfType != sdk.TextType && p1.OfType != sdk.TableType && p1.Type != "row" { // turning off text ,table and row panels for now
				if p1.Datasource != nil {
					tempDsStr := fmt.Sprint(p1.Datasource)
					if strings.HasPrefix(tempDsStr, "$") { // Formating Datasource id
						p1.Datasource = &GrafanaDataSource{
							Name: tmpDsName[strings.Replace(tempDsStr, "$", "", 1)],
						}

					}
					// if strings.HasPrefix(*p1.Datasource, "$") { // Formating Datasource id
					// 	*p1.Datasource = tmpDsName[strings.Replace(*p1.Datasource, "$", "", 1)]
					// }
				}
				grafBoard.Panels = append(grafBoard.Panels, p1)
			} else if p1.OfType != sdk.TextType && p1.OfType != sdk.TableType && p1.Type == "row" && len(p1.Panels) > 0 { // Looking for Panels with Row
				for _, p2 := range p1.Panels { // Adding Panels inside the Row Panel to grafBoard
					if p2.OfType != sdk.TextType && p2.OfType != sdk.TableType && p2.Type != "row" {
						tempDsStr := fmt.Sprint(p2.Datasource)
						if strings.HasPrefix(tempDsStr, "$") { // Formating Datasource id
							p2.Datasource = &GrafanaDataSource{
								Name: tmpDsName[strings.Replace(tempDsStr, "$", "", 1)],
							}
						}
						// if strings.HasPrefix(*p2.Datasource, "$") { // Formating Datasource id
						// 	*p2.Datasource = tmpDsName[strings.Replace(*p2.Datasource, "$", "", 1)]
						// }
						p3, _ := p2.MarshalJSON()
						p4 := &sdk.Panel{}
						if err := p4.UnmarshalJSON(p3); err != nil {
							continue
						}
						grafBoard.Panels = append(grafBoard.Panels, p4)
					}
				}
			}
		}
	} else if len(board.Rows) > 0 { //Process Board Rows
		for _, r1 := range board.Rows {
			for _, p2 := range r1.Panels {
				if p2.OfType != sdk.TextType && p2.OfType != sdk.TableType && p2.Type != "row" { // turning off text, table and row panels for now
					tempDsStr := fmt.Sprint(p2.Datasource)
					if strings.HasPrefix(tempDsStr, "$") { // Formating Datasource id
						p2.Datasource = &GrafanaDataSource{
							Name: tmpDsName[strings.Replace(tempDsStr, "$", "", 1)],
						}
					}
					// if strings.HasPrefix(*p2.Datasource, "$") { // Formating Datasource id
					// 	*p2.Datasource = tmpDsName[strings.Replace(*p2.Datasource, "$", "", 1)]
					// }
					p3, _ := p2.MarshalJSON()
					p4 := &sdk.Panel{}
					_ = p4.UnmarshalJSON(p3)
					// logrus.Debugf("board: %d, Row panel id: %d", board.ID, p4.ID)
					grafBoard.Panels = append(grafBoard.Panels, p4)
				}
			}
		}
	}
	return grafBoard, nil
}

func Validate(g *GrafanaClient, ctx context.Context, BaseURL, APIKey string) error {
	fmt.Println("Staring Validate")
	if strings.HasSuffix(BaseURL, "/") {
		BaseURL = strings.Trim(BaseURL, "/")
	}
	c, err := sdk.NewClient(BaseURL, APIKey, g.httpClient)
	if err != nil {
		return commonError(err)
	}

	if _, err := c.GetActualOrg(ctx); err != nil {
		return commonError(err)
	}
	fmt.Println("Validate completed")
	return nil
}

func commonError(err error) error {
	fmt.Println("Error: ", err)
	return err
}
func dashboardError(err error, UID string) error {
	fmt.Println("UID: "+UID+", Error: ", err)
	return err
}

func GrafanaQueryHandler(w http.ResponseWriter, r *http.Request) {

	grafanaUrl := r.URL.Query().Get("grafanaUrl")
	apiKey := r.URL.Query().Get("apiKey")
	if grafanaUrl == "" {
		fmt.Println("Grafana url not provided")
		return
	} else if apiKey == "" {
		fmt.Println("Grafana api key (userId:password) not provided")
		return
	}

	prefObj := &Preference{
		Grafana: &Grafana{
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
	client := NewGrafanaClient()
	data, err := GrafanaQuery(client, r.Context(), prefObj.Grafana.GrafanaURL, prefObj.Grafana.GrafanaAPIKey, &reqQuery)
	if err != nil {
		// h.log.Error(ErrGrafanaQuery(err))
		// http.Error(w, ErrGrafanaQuery(err).Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func GrafanaQuery(g *GrafanaClient, ctx context.Context, BaseURL, APIKey string, queryData *url.Values) ([]byte, error) {
	if queryData == nil {
		return nil, errors.New("query data passed is nil")
	}
	query := strings.TrimSpace(queryData.Get("query"))
	dsID := queryData.Get("dsid")
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
			if g.promMode {
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
			if g.promMode {
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
		if g.promMode {
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

	data, err := g.makeRequest(ctx, queryURL, APIKey)
	if err != nil {
		return nil, errors.New("error getting data from Grafana API")
	}
	return data, nil
}

func (g *GrafanaClient) makeRequest(ctx context.Context, queryURL, APIKey string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, err
	}
	if !g.promMode {
		req.Header.Set("Authorization", APIKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autograf")
	// c := &http.Client{}
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		// return nil, fmt.Errorf("%s", data)
		logrus.Errorf("unable to get data from URL: %s due to status code: %d", queryURL, resp.StatusCode)
		return nil, fmt.Errorf("unable to fetch data from url: %s", queryURL)
	}
	return data, nil
}

// GrafanaQueryRangeHandler is used for handling Grafana Range queries
func GrafanaQueryRangeHandler(w http.ResponseWriter, req *http.Request) {
	// if req.Method != http.MethodGet {
	// 	w.WriteHeader(http.StatusNotFound)
	// 	return
	// }

	reqQuery := req.URL.Query()
	client := NewGrafanaClient()
	data, err := GrafanaQueryRange(client, req.Context(), reqQuery.Get("url"), reqQuery.Get("api-key"), &reqQuery)
	if err != nil {
		// h.log.Error(ErrGrafanaQuery(err))
		// http.Error(w, ErrGrafanaQuery(err).Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

// GrafanaQueryRange parses the given params and performs Grafana range queries
func GrafanaQueryRange(g *GrafanaClient, ctx context.Context, BaseURL, APIKey string, queryData *url.Values) ([]byte, error) {
	if queryData == nil {
		return nil, errors.New("query data passed is nil")
	}

	c, err := sdk.NewClient(BaseURL, APIKey, g.httpClient)
	if err != nil {
		return nil, commonError(err)
	}

	ds, err := c.GetDatasourceByName(ctx, queryData.Get("ds"))
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	var reqURL string
	if g.promMode {
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
	data, err := g.makeRequest(ctx, queryURL, APIKey)
	if err != nil {
		return nil, commonError(err)
	}
	return data, nil
}
