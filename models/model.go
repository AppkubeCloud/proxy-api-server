package models

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana-tools/sdk"
	"github.com/sirupsen/logrus"
)

type GrafanaDashboard struct {
	Dashboard *sdk.Board           `json:"dashboard,omitempty"`
	Meta      *sdk.BoardProperties `json:"meta,omitempty"`
}
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
	HttpClient *http.Client
	PromMode   bool
}

type HandlerConfig struct {
	GrafanaClient         *GrafanaClient
	GrafanaClientForQuery *GrafanaClient
}

func (g *GrafanaClient) MakeRequest(ctx context.Context, queryURL, APIKey string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, err
	}
	if !g.PromMode {
		req.Header.Set("Authorization", APIKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autograf")
	// c := &http.Client{}
	resp, err := g.HttpClient.Do(req)
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
