package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/melon0826/shaniu/utils"
)

const yybgoPanelsStorageKey = "yybgo_panels"

var legacyYybGoPanels = MakeBucket("yybgo_panels")

type YybGoPanel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	CreatedAt     int    `json:"created_at"`
	UpdatedAt     int    `json:"updated_at"`
	LastCheckedAt int    `json:"last_checked_at"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	Group         string `json:"group"`
	Namespace     string `json:"namespace"`
	AccountLimit  string `json:"account_limit"`
	AccountUsed   string `json:"account_used"`
	CreditBalance string `json:"credit_balance"`
}

type publicYybGoPanel struct {
	Index   int    `json:"index"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func init() {
	GinApi(GET, "/api/yybgo/panels", RequireAuth, func(ctx *gin.Context) {
		panels := getYybGoPanels()
		refreshYybGoPanelsStatus(panels)
		ApiList(ctx, panels, len(panels))
	})

	GinApi(POST, "/api/yybgo/panel/test", RequireAuth, func(ctx *gin.Context) {
		panel := YybGoPanel{}
		if err := ctx.BindJSON(&panel); err != nil {
			ApiFail(ctx, err.Error())
			return
		}
		if err := validateYybGoPanelInput(&panel); err != nil {
			ApiFail(ctx, err.Error())
			return
		}
		result, err := testYybGoPanel(panel)
		if err != nil {
			ApiFail(ctx, err.Error())
			return
		}
		ApiOK(ctx, result)
	})

	GinApi(POST, "/api/yybgo/panel", RequireAuth, func(ctx *gin.Context) {
		panel := YybGoPanel{}
		if err := ctx.BindJSON(&panel); err != nil {
			ApiFail(ctx, err.Error())
			return
		}
		if err := validateYybGoPanelInput(&panel); err != nil {
			ApiFail(ctx, err.Error())
			return
		}
		result, err := testYybGoPanel(panel)
		if err != nil {
			ApiFail(ctx, err.Error())
			return
		}
		now := int(time.Now().Unix())
		panels := getYybGoPanels()
		index := -1
		if panel.ID != "" {
			for i := range panels {
				if panels[i].ID == panel.ID {
					index = i
					break
				}
			}
		}
		if panel.ID == "" {
			panel.ID = utils.GenUUID()
			panel.CreatedAt = now
		} else if index >= 0 {
			if panels[index].CreatedAt != 0 {
				panel.CreatedAt = panels[index].CreatedAt
			} else {
				panel.CreatedAt = now
			}
		} else {
			panel.CreatedAt = now
		}
		if panel.Name == "" {
			panel.Name = panel.Address
		}
		panel.UpdatedAt = now
		panel.LastCheckedAt = now
		panel.Status = "online"
		panel.Message = result.Message
		if index >= 0 {
			panels[index] = panel
		} else {
			panels = append(panels, panel)
		}
		saveYybGoPanels(panels)
		ApiOK(ctx, panel)
	})

	GinApi(DELETE, "/api/yybgo/panel", RequireAuth, func(ctx *gin.Context) {
		panel := YybGoPanel{}
		if err := ctx.BindJSON(&panel); err != nil {
			ApiFail(ctx, err.Error())
			return
		}
		if panel.ID == "" {
			ApiFail(ctx, "缺少 yyb-go ID")
			return
		}
		panels := getYybGoPanels()
		next := make([]YybGoPanel, 0, len(panels))
		for _, item := range panels {
			if item.ID != panel.ID {
				next = append(next, item)
			}
		}
		saveYybGoPanels(next)
		ApiOK(ctx, nil)
	})
}

func getYybGoPanels() []YybGoPanel {
	raw := strings.TrimSpace(shaniu.GetString(yybgoPanelsStorageKey))
	if raw != "" {
		panels := []YybGoPanel{}
		if json.Unmarshal([]byte(strings.TrimPrefix(raw, "o:")), &panels) == nil {
			return panels
		}
	}
	panels := getLegacyYybGoPanels()
	if len(panels) > 0 {
		saveYybGoPanels(panels)
	}
	return panels
}

func getLegacyYybGoPanels() []YybGoPanel {
	panels := []YybGoPanel{}
	legacyYybGoPanels.Foreach(func(_, data []byte) error {
		panel := YybGoPanel{}
		if json.Unmarshal(data, &panel) == nil && panel.ID != "" {
			panels = append(panels, panel)
		}
		return nil
	})
	return panels
}

func saveYybGoPanels(panels []YybGoPanel) {
	shaniu.Set(yybgoPanelsStorageKey, utils.JsonMarshal(panels))
}

func validateYybGoPanelInput(panel *YybGoPanel) error {
	panel.Name = strings.TrimSpace(panel.Name)
	panel.Address = normalizeYybGoAddress(panel.Address)
	if panel.Address == "" {
		return errors.New("yyb-go 地址不能为空")
	}
	parsed, err := url.ParseRequestURI(panel.Address)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("yyb-go 地址格式错误：%v", err)
	}
	return nil
}

func normalizeYybGoAddress(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return ""
	}
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}
	return strings.TrimRight(address, "/")
}

func testYybGoPanel(panel YybGoPanel) (*YybGoPanel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, panel.Address, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yyb-go 接口连接失败：%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("yyb-go 接口 HTTP %d", resp.StatusCode)
	}
	panel.Address = normalizeYybGoAddress(panel.Address)
	panel.Status = "online"
	panel.Message = "连接成功"
	panel.LastCheckedAt = int(time.Now().Unix())
	return &panel, nil
}

func refreshYybGoPanelsStatus(panels []YybGoPanel) {
	var wg sync.WaitGroup
	for index := range panels {
		wg.Add(1)
		go func(panel *YybGoPanel) {
			defer wg.Done()
			refreshYybGoPanelStatus(panel)
		}(&panels[index])
	}
	wg.Wait()
}

func refreshYybGoPanelStatus(panel *YybGoPanel) {
	if panel == nil || panel.Address == "" {
		return
	}
	panel.Address = normalizeYybGoAddress(panel.Address)
	panel.LastCheckedAt = int(time.Now().Unix())
	raw, err := requestYybGoJSONWithTimeout(panel, http.MethodGet, "/api/auth/validate", nil, nil, 4*time.Second)
	if err != nil {
		panel.Status = "offline"
		panel.Message = err.Error()
		return
	}
	if err := yybgoEnvelopeError(raw, "验证失败"); err != nil {
		panel.Status = "offline"
		panel.Message = err.Error()
		return
	}
	panel.Status = "online"
	panel.Message = "验证通过"
	applyYybGoAuthStatus(panel, raw)
	if err := refreshYybGoCreditBalance(panel); err != nil && panel.Message == "验证通过" {
		panel.Message = "积分读取失败：" + err.Error()
	}
}

func refreshYybGoCreditBalance(panel *YybGoPanel) error {
	raw, err := requestYybGoJSONWithTimeout(panel, http.MethodGet, "/credits/balance", nil, nil, 4*time.Second)
	if err != nil {
		return err
	}
	if err := yybgoEnvelopeError(raw, "积分读取失败"); err != nil {
		return err
	}
	value := decodeYybGoJSONValue(raw)
	panel.CreditBalance = yybgoStringValue(firstYybGoValue(value, "balance", "credits", "credit", "points"))
	return nil
}

func yybgoEnvelopeError(raw json.RawMessage, fallback string) error {
	envelope := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil
	}
	statusRaw, ok := envelope["status"]
	if !ok {
		return nil
	}
	status := false
	if err := json.Unmarshal(statusRaw, &status); err != nil || status {
		return nil
	}
	messageRaw, ok := envelope["message"]
	if !ok {
		return errors.New(fallback)
	}
	message := ""
	if err := json.Unmarshal(messageRaw, &message); err != nil {
		return errors.New(fallback)
	}
	return errors.New(message)
}

func applyYybGoAuthStatus(panel *YybGoPanel, raw json.RawMessage) {
	value := decodeYybGoJSONValue(raw)
	panel.Group = yybgoGroupLabel(firstNonEmpty(
		yybgoStringValue(firstYybGoValue(value, "group")),
		yybgoStringValue(firstYybGoValue(value, "user_group")),
	))
	panel.Namespace = yybgoStringValue(firstYybGoValue(value, "namespace"))
	panel.AccountLimit = yybgoStringValue(firstYybGoValue(value, "limit", "account_limit", "max_accounts"))
	panel.AccountUsed = yybgoStringValue(firstYybGoValue(value, "used", "account_used", "current_accounts", "accounts_count"))
	if quota := firstYybGoValue(value, "quota"); quota != nil {
		panel.AccountLimit = firstNonEmpty(panel.AccountLimit, yybgoStringValue(firstYybGoValue(quota, "limit", "max", "total")))
		panel.AccountUsed = firstNonEmpty(panel.AccountUsed, yybgoStringValue(firstYybGoValue(quota, "used", "current", "count")))
	}
}

func decodeYybGoJSONValue(raw json.RawMessage) any {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil
	}
	return value
}

func firstYybGoValue(value any, keys ...string) any {
	wanted := map[string]struct{}{}
	for _, key := range keys {
		wanted[strings.ToLower(key)] = struct{}{}
	}
	return walkYybGoValue(value, wanted)
}

func walkYybGoValue(value any, keys map[string]struct{}) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if _, ok := keys[strings.ToLower(key)]; ok && item != nil && item != "" {
				return item
			}
		}
		for _, item := range typed {
			if found := walkYybGoValue(item, keys); found != nil && found != "" {
				return found
			}
		}
	case []any:
		for _, item := range typed {
			if found := walkYybGoValue(item, keys); found != nil && found != "" {
				return found
			}
		}
	}
	return nil
}

func yybgoStringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(data)
	}
}

func yybgoGroupLabel(group string) string {
	switch strings.ToLower(strings.TrimSpace(group)) {
	case "":
		return ""
	case "normal":
		return "普通用户组"
	case "pro":
		return "PRO"
	case "vip":
		return "VIP"
	default:
		return group
	}
}

func publicYybGoPanels() []publicYybGoPanel {
	panels := getYybGoPanels()
	result := make([]publicYybGoPanel, 0, len(panels))
	for index, panel := range panels {
		result = append(result, publicYybGoPanel{
			Index:   index + 1,
			ID:      panel.ID,
			Name:    firstNonEmpty(panel.Name, fmt.Sprintf("yyb-go #%d", index+1)),
			Status:  panel.Status,
			Message: panel.Message,
		})
	}
	return result
}

func yybGoPanelByIndex(index int) (*YybGoPanel, error) {
	panels := getYybGoPanels()
	if len(panels) == 0 {
		return nil, errors.New("后台未绑定 yyb-go")
	}
	if index <= 0 {
		index = 1
	}
	if index > len(panels) {
		return nil, fmt.Errorf("yyb-go 编号 %d 不存在", index)
	}
	panel := panels[index-1]
	if panel.Address == "" {
		return nil, errors.New("yyb-go 配置不完整")
	}
	return &panel, nil
}

func requestYybGoJSON(panel *YybGoPanel, method string, path string, body interface{}, query map[string]string) (json.RawMessage, error) {
	return requestYybGoJSONWithTimeout(panel, method, path, body, query, 15*time.Second)
}

func requestYybGoJSONWithTimeout(panel *YybGoPanel, method string, path string, body interface{}, query map[string]string, timeout time.Duration) (json.RawMessage, error) {
	if panel == nil {
		return nil, errors.New("yyb-go 配置不存在")
	}
	address := normalizeYybGoAddress(panel.Address)
	values := url.Values{}
	for key, value := range query {
		values.Set(key, value)
	}
	requestURL := address + path
	if encoded := values.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yyb-go 请求失败：%v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(raw))
		if len(message) > 200 {
			message = message[:200]
		}
		return nil, fmt.Errorf("yyb-go HTTP %d：%s", resp.StatusCode, message)
	}
	if !json.Valid(raw) {
		message := strings.TrimSpace(string(raw))
		if len(message) > 200 {
			message = message[:200]
		}
		return nil, fmt.Errorf("yyb-go 返回非 JSON：%s", message)
	}
	return json.RawMessage(raw), nil
}
