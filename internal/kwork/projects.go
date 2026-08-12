package kwork

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"kwork-assistant/internal/domain"
)

// FetchProjects retrieves new projects from Kwork.
func (c *Client) FetchProjects(ctx context.Context, token string, limit int) ([]domain.Project, error) {
	params := url.Values{}
	params.Set("token", token)
	params.Set("page", "1") // always fetch first page for recent projects

	data, err := c.doRequest(ctx, "POST", "projects", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}

	var rawItems []map[string]interface{}
	if err := json.Unmarshal(data, &rawItems); err != nil {
		return nil, fmt.Errorf("failed to parse projects array: %w", err)
	}

	var results []domain.Project
	for _, item := range rawItems {
		if limit > 0 && len(results) >= limit {
			break
		}

		idFloat, _ := item["id"].(float64)
		if idFloat == 0 {
			continue
		}
		id := int64(idFloat)

		title, _ := item["title"].(string)
		desc, _ := item["description"].(string)

		priceFloat, _ := item["price"].(float64)
		var catID int64
		if cat, ok := item["category_id"].(float64); ok {
			catID = int64(cat)
		}

		var timeAdded int64
		if ta, ok := item["date_confirm"].(float64); ok {
			timeAdded = int64(ta)
		} else {
			timeAdded = time.Now().Unix()
		}

		rawJson, _ := json.Marshal(item)

		username, _ := item["username"].(string)

		var buyerID int64
		if u, ok := item["user_id"].(float64); ok {
			buyerID = int64(u)
		}

		results = append(results, domain.Project{
			ExternalID:  id,
			Source:      "kwork",
			URL:         fmt.Sprintf("https://kwork.ru/projects/%d/view", id),
			Title:       title,
			Description: desc,
			BudgetFrom: sql.NullFloat64{
				Float64: priceFloat,
				Valid:   priceFloat > 0,
			},
			Currency: sql.NullString{
				String: "RUB",
				Valid:  true,
			},
			CategoryID: sql.NullInt64{
				Int64: catID,
				Valid: catID > 0,
			},
			BuyerID: sql.NullInt64{
				Int64: buyerID,
				Valid: buyerID > 0,
			},
			BuyerName: sql.NullString{
				String: username,
				Valid:  username != "",
			},
			PublishedAt: time.Unix(timeAdded, 0),
			FetchedAt:   time.Now(),
			RawJSON:     string(rawJson),
		})
	}

	return results, nil
}
