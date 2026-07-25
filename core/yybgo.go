package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/melon0826/shaniu/utils"
)

const yybgoPanelsStorageKey = "yybgo_panels"

var legacyYybGoPanels = MakeBucket("yybgo_panels")

type YybGoPanel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	CreatedAt     int    `json:"created_at"`
	UpdatedAt     int    `json:"updated_at"`
	LastCheckedAt int    `json:"last_checked_at"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

func init() {
	GinApi(GET, "/api/yybgo/panels", RequireAuth, func(ctx *gin.Context) {
		panels := getYybGoPanels()
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
	panel.Username = strings.TrimSpace(panel.Username)
	panel.Password = strings.TrimSpace(panel.Password)
	if panel.Username != "" || panel.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(panel.Username + ":" + panel.Password))
		req.Header.Set("Authorization", "Basic "+auth)
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
