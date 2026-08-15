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

		proj := ParseProject(item)
		if proj.ExternalID != 0 {
			results = append(results, proj)
		}
	}

	return results, nil
}

func ParseProject(item map[string]interface{}) domain.Project {
	var proj domain.Project
	idFloat, _ := item["id"].(float64)
	if idFloat == 0 {
		return proj
	}
	id := int64(idFloat)

	title, _ := item["title"].(string)
	desc, _ := item["description"].(string)

	parsePrice := func(v interface{}) float64 {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case string:
			var f float64
			fmt.Sscanf(val, "%f", &f)
			return f
		default:
			return 0
		}
	}

	priceFloat := parsePrice(item["price"])
	possiblePriceFloat := parsePrice(item["possible_price_limit"])

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

	proj.ExternalID = id
	proj.Source = "kwork"
	proj.URL = ""
	proj.Title = title
	proj.Description = desc
	proj.BudgetFrom = sql.NullFloat64{Float64: priceFloat, Valid: priceFloat > 0}
	proj.BudgetTo = sql.NullFloat64{Float64: possiblePriceFloat, Valid: possiblePriceFloat > 0}
	
	if curr, ok := item["currency"].(string); ok && curr != "" {
		proj.Currency = sql.NullString{String: curr, Valid: true}
	} else if priceFloat > 0 || possiblePriceFloat > 0 {
		proj.Currency = sql.NullString{String: "RUB", Valid: true}
	}

	proj.CategoryID = sql.NullInt64{Int64: catID, Valid: catID > 0}
	proj.BuyerID = sql.NullInt64{Int64: buyerID, Valid: buyerID > 0}
	proj.BuyerName = sql.NullString{String: username, Valid: username != ""}
	proj.PublishedAt = time.Unix(timeAdded, 0)
	proj.FetchedAt = time.Now()
	proj.RawJSON = string(rawJson)

	return proj
}
