package kwork

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/rtexty/gokwork/pkg/kwork/errors"
	"github.com/rtexty/gokwork/pkg/kwork/types"
	"golang.org/x/net/proxy"
)

const (
	apiHost          = "https://api.kwork.ru"
	authHeader       = "Basic bW9iaWxlX2FwaTpxRnZmUmw3dw=="
)

// Client представляет клиент Kwork API
type Client struct {
	httpClient *http.Client
	login      string
	password   string
	token      string
	phoneLast  string
}

// Config конфигурация клиента
type Config struct {
	Login     string
	Password  string
	PhoneLast string
	ProxyURL  string
}

// NewClient создает новый клиент Kwork
func NewClient(cfg Config) (*Client, error) {
	httpClient := &http.Client{}

	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}

		if strings.HasPrefix(cfg.ProxyURL, "socks5://") || strings.HasPrefix(cfg.ProxyURL, "socks4://") {
			dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, nil, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
			}
			httpClient.Transport = &http.Transport{
				Dial: dialer.Dial,
			}
		} else {
			httpClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	return &Client{
		httpClient: httpClient,
		login:      cfg.Login,
		password:   cfg.Password,
		phoneLast:  cfg.PhoneLast,
	}, nil
}

// APIResponse общий формат ответа API
type APIResponse struct {
	Success  bool                   `json:"success"`
	Error    string                 `json:"error,omitempty"`
	Response map[string]interface{} `json:"response,omitempty"`
}

// apiRequest выполняет запрос к API
func (c *Client) apiRequest(ctx context.Context, method, apiMethod string, params map[string]string) (map[string]interface{}, error) {
	// Убираем nil значения
	cleanParams := make(map[string]string)
	for k, v := range params {
		if v != "" {
			cleanParams[k] = v
		}
	}

	urlStr := fmt.Sprintf("%s/%s", apiHost, apiMethod)

	var req *http.Request
	var err error

	if method == "POST" {
		values := url.Values{}
		for k, v := range cleanParams {
			values.Set(k, v)
		}
		req, err = http.NewRequestWithContext(ctx, method, urlStr, bytes.NewBufferString(values.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, urlStr, nil)
		if err != nil {
			return nil, err
		}
		q := req.URL.Query()
		for k, v := range cleanParams {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		return nil, errors.NewKworkError(string(body))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if !apiResp.Success {
		return nil, errors.NewKworkError(apiResp.Error)
	}

	return apiResp.Response, nil
}

// Close закрывает клиент
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

// GetToken получает токен авторизации
func (c *Client) GetToken(ctx context.Context) (string, error) {
	if c.token != "" {
		return c.token, nil
	}

	params := map[string]string{
		"login":      c.login,
		"password":   c.password,
		"phone_last": c.phoneLast,
	}

	resp, err := c.apiRequest(ctx, "POST", "signIn", params)
	if err != nil {
		return "", err
	}

	token, ok := resp["token"].(string)
	if !ok {
		return "", errors.NewKworkError("invalid token in response")
	}

	c.token = token
	return token, nil
}

// GetMe получает профиль текущего пользователя
func (c *Client) GetMe(ctx context.Context) (*types.Actor, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"token": token,
	}

	resp, err := c.apiRequest(ctx, "POST", "actor", params)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var actor types.Actor
	if err := json.Unmarshal(data, &actor); err != nil {
		return nil, err
	}

	return &actor, nil
}

// GetUser получает профиль пользователя по ID
func (c *Client) GetUser(ctx context.Context, userID int) (*types.User, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"token": token,
		"id":    fmt.Sprintf("%d", userID),
	}

	resp, err := c.apiRequest(ctx, "POST", "user", params)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var user types.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// SetTyping устанавливает статус "печатает" для получателя
func (c *Client) SetTyping(ctx context.Context, recipientID int) error {
	token, err := c.GetToken(ctx)
	if err != nil {
		return err
	}

	params := map[string]string{
		"token":       token,
		"recipientId": fmt.Sprintf("%d", recipientID),
	}

	_, err = c.apiRequest(ctx, "POST", "typing", params)
	return err
}

// GetAllDialogs получает все диалоги
func (c *Client) GetAllDialogs(ctx context.Context) ([]types.Dialog, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	var dialogs []types.Dialog
	page := 1

	for {
		params := map[string]string{
			"token":  token,
			"filter": "all",
			"page":   fmt.Sprintf("%d", page),
		}

		resp, err := c.apiRequest(ctx, "POST", "dialogs", params)
		if err != nil {
			return nil, err
		}

		if len(resp) == 0 {
			break
		}

		data, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}

		var pageDialogs []types.Dialog
		if err := json.Unmarshal(data, &pageDialogs); err != nil {
			return nil, err
		}

		dialogs = append(dialogs, pageDialogs...)
		page++
	}

	return dialogs, nil
}

// SetOffline устанавливает статус оффлайн
func (c *Client) SetOffline(ctx context.Context) error {
	token, err := c.GetToken(ctx)
	if err != nil {
		return err
	}

	params := map[string]string{
		"token": token,
	}

	_, err = c.apiRequest(ctx, "POST", "offline", params)
	return err
}

// GetDialogWithUser получает диалог с пользователем по имени
func (c *Client) GetDialogWithUser(ctx context.Context, username string) ([]types.InboxMessage, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	var messages []types.InboxMessage
	page := 1

	for {
		params := map[string]string{
			"token":    token,
			"username": username,
			"page":     fmt.Sprintf("%d", page),
		}

		resp, err := c.apiRequest(ctx, "POST", "inboxes", params)
		if err != nil {
			return nil, err
		}

		if len(resp) == 0 {
			break
		}

		data, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}

		var pageMessages []types.InboxMessage
		if err := json.Unmarshal(data, &pageMessages); err != nil {
			return nil, err
		}

		messages = append(messages, pageMessages...)

		// Проверяем пагинацию
		if paging, ok := resp["paging"].(map[string]interface{}); ok {
			if pages, ok := paging["pages"].(float64); ok && page >= int(pages) {
				break
			}
		}

		page++
	}

	return messages, nil
}

// GetCategories получает категории
func (c *Client) GetCategories(ctx context.Context) ([]types.Category, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"token": token,
		"type":  "1",
	}

	resp, err := c.apiRequest(ctx, "POST", "categories", params)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var categories []types.Category
	if err := json.Unmarshal(data, &categories); err != nil {
		return nil, err
	}

	return categories, nil
}

// GetConnects получает информацию о коннектах
func (c *Client) GetConnects(ctx context.Context) (*types.Connects, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"token":      token,
		"categories": "",
	}

	resp, err := c.apiRequest(ctx, "POST", "projects", params)
	if err != nil {
		return nil, err
	}

	connectsData, ok := resp["connects"].(map[string]interface{})
	if !ok {
		return nil, errors.NewKworkError("invalid connects data in response")
	}

	data, err := json.Marshal(connectsData)
	if err != nil {
		return nil, err
	}

	var connects types.Connects
	if err := json.Unmarshal(data, &connects); err != nil {
		return nil, err
	}

	return &connects, nil
}

// ProjectsParams параметры для получения проектов
type ProjectsParams struct {
	CategoriesIDs   []int
	PriceFrom       int
	PriceTo         int
	HiringFrom      int
	KworksFilterFrom int
	KworksFilterTo  int
	Page            int
	Query           string
}

// GetProjects получает проекты с биржи
func (c *Client) GetProjects(ctx context.Context, params ProjectsParams) ([]types.Project, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	// Формируем строку категорий
	categoriesStr := ""
	if len(params.CategoriesIDs) > 0 {
		cats := make([]string, len(params.CategoriesIDs))
		for i, id := range params.CategoriesIDs {
			cats[i] = fmt.Sprintf("%d", id)
		}
		categoriesStr = strings.Join(cats, ",")
	}

	apiParams := map[string]string{
		"token":      token,
		"categories": categoriesStr,
	}

	if params.PriceFrom > 0 {
		apiParams["price_from"] = fmt.Sprintf("%d", params.PriceFrom)
	}
	if params.PriceTo > 0 {
		apiParams["price_to"] = fmt.Sprintf("%d", params.PriceTo)
	}
	if params.HiringFrom > 0 {
		apiParams["hiring_from"] = fmt.Sprintf("%d", params.HiringFrom)
	}
	if params.KworksFilterFrom > 0 {
		apiParams["kworks_filter_from"] = fmt.Sprintf("%d", params.KworksFilterFrom)
	}
	if params.KworksFilterTo > 0 {
		apiParams["kworks_filter_to"] = fmt.Sprintf("%d", params.KworksFilterTo)
	}
	if params.Page > 0 {
		apiParams["page"] = fmt.Sprintf("%d", params.Page)
	}
	if params.Query != "" {
		apiParams["query"] = params.Query
	}

	resp, err := c.apiRequest(ctx, "POST", "projects", apiParams)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var projects []types.Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// SendMessage отправляет сообщение пользователю
func (c *Client) SendMessage(ctx context.Context, userID int, text string) error {
	token, err := c.GetToken(ctx)
	if err != nil {
		return err
	}

	params := map[string]string{
		"token":   token,
		"user_id": fmt.Sprintf("%d", userID),
		"text":    url.QueryEscape(text),
	}

	_, err = c.apiRequest(ctx, "POST", "inboxCreate", params)
	return err
}

// DeleteMessage удаляет сообщение
func (c *Client) DeleteMessage(ctx context.Context, messageID int) error {
	token, err := c.GetToken(ctx)
	if err != nil {
		return err
	}

	params := map[string]string{
		"token": token,
		"id":    fmt.Sprintf("%d", messageID),
	}

	_, err = c.apiRequest(ctx, "POST", "inboxDelete", params)
	return err
}

// GetNotifications получает уведомления
func (c *Client) GetNotifications(ctx context.Context) (map[string]interface{}, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"token": token,
	}

	return c.apiRequest(ctx, "POST", "notifications", params)
}

// GetWorkerOrders получает заказы работника
func (c *Client) GetWorkerOrders(ctx context.Context) (map[string]interface{}, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"token":  token,
		"filter": "all",
	}

	return c.apiRequest(ctx, "POST", "workerOrders", params)
}

// GetPayerOrders получает заказы заказчика
func (c *Client) GetPayerOrders(ctx context.Context) (map[string]interface{}, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"token":  token,
		"filter": "all",
	}

	return c.apiRequest(ctx, "POST", "payerOrders", params)
}

// getChannel получает канал для WebSocket
func (c *Client) getChannel(ctx context.Context) (string, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return "", err
	}

	params := map[string]string{
		"token": token,
	}

	resp, err := c.apiRequest(ctx, "POST", "getChannel", params)
	if err != nil {
		return "", err
	}

	channel, ok := resp["channel"].(string)
	if !ok {
		return "", errors.NewKworkError("invalid channel in response")
	}

	return channel, nil
}
