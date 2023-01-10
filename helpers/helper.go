package helpers

import (
	"context"
	"fmt"
	"proxy-api-server/models"
	"proxy-api-server/util"
	"strings"

	"github.com/gosimple/slug"
	"github.com/grafana-tools/sdk"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ProcessBoard(g *models.GrafanaClient, ctx context.Context, c *sdk.Client, board *sdk.Board, link *sdk.FoundBoard) (*models.GrafanaBoard, error) {
	var orgID uint
	if !g.PromMode {
		org, err := c.GetActualOrg(ctx)
		if err != nil {
			return nil, util.CommonError(err)
		}
		orgID = org.ID
	}
	grafBoard := &models.GrafanaBoard{
		URI:          link.URI,
		Title:        link.Title,
		UID:          board.UID,
		Slug:         slug.Make(board.Title),
		TemplateVars: []*models.GrafanaTemplateVars{},
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
				util.CommonError(err)
				return nil, err
			}
			if c != nil {
				ds, err = c.GetDatasourceByName(ctx, dsName)
				if err != nil {
					return nil, util.DashboardError(err, "Error getting Grafana Board's Datasource")
				}
			} else {
				ds.Name = dsName
			}

			tvVal := tmpVar.Current.Text
			grafBoard.TemplateVars = append(grafBoard.TemplateVars, &models.GrafanaTemplateVars{
				Name:  tmpVar.Name,
				Query: fmt.Sprint(tmpVar.Query),
				Datasource: &models.GrafanaDataSource{
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
						p1.Datasource = &models.GrafanaDataSource{
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
							p2.Datasource = &models.GrafanaDataSource{
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
						p2.Datasource = &models.GrafanaDataSource{
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

func Validate(g *models.GrafanaClient, ctx context.Context, BaseURL, APIKey string) error {
	fmt.Println("Staring Validate")
	if strings.HasSuffix(BaseURL, "/") {
		BaseURL = strings.Trim(BaseURL, "/")
	}
	c, err := sdk.NewClient(BaseURL, APIKey, g.HttpClient)
	if err != nil {
		return util.CommonError(err)
	}

	if _, err := c.GetActualOrg(ctx); err != nil {
		return util.CommonError(err)
	}
	fmt.Println("Validate completed")
	return nil
}
