package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var yybGoClientAPIEndpoints = map[string]string{
	"createQr":        "/api/qr/start",
	"checkQr":         "/api/qr/status",
	"addUser":         "/api/accounts/add",
	"rescanUser":      "/api/accounts/rescan",
	"userList":        "/api/accounts",
	"checkUsers":      "/api/accounts/status",
	"setUserRemark":   "/api/accounts/remark",
	"setUserDisabled": "/api/accounts/disable",
	"deleteUser":      "/api/accounts/delete",
	"proxyList":       "/api/proxies",
	"testProxy":       "/api/proxies/test",
	"addProxy":        "/api/proxies/add",
	"deleteProxy":     "/api/proxies/delete",
	"creditBalance":   "/credits/balance",
	"creditLedger":    "/credits/ledger",
	"getCode":         "/wx/code",
	"getSession":      "/wx/getsession",
	"refreshSession":  "/wx/refresh",
	"getUserInfo":     "/wx/getuserinfo",
	"getEncryptKey":   "/wx/encryptkey",
	"getPhoneNumber":  "/wx/getphonenumber",
	"cloud":           "/wx/cloud",
	"gateway":         "/wx/gateway",
	"qrCodeAuth":      "/wx/qrcodeauth",
	"oAuth":           "/wx/oauth",
	"translateLink":   "/wx/translatelink",
	"autoAuth":        "/wx/autoauth",
	"appMsgExt":       "/wx/appmsgext",
	"appMsgLike":      "/wx/appmsglike",
}

func yybGoSourceBlock(t *testing.T, source, start, end string) string {
	t.Helper()
	from := strings.Index(source, start)
	if from < 0 {
		t.Fatalf("YybGo start marker %q is missing", start)
	}
	block := source[from:]
	if to := strings.Index(block, end); to >= 0 {
		block = block[:to]
	}
	return block
}

func readYybGoSource(t *testing.T, relativePath, start, end string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", relativePath))
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}
	return yybGoSourceBlock(t, string(raw), start, end)
}

func TestYybGoInlineEndpointWrappers(t *testing.T) {
	runtimes := map[string]string{
		"node preload":  yybGoSourceBlock(t, nodeRuntimePreloadScript, "class YybGo {", "class DaiDai {"),
		"node module":   readYybGoSource(t, "proto3/shaniu.js", "class YybGo {", "class DaiDai {"),
		"python module": readYybGoSource(t, "proto3/shaniu.py", "class YybGo:", "class DaiDai:"),
	}
	typings := map[string]string{
		"grpc plugin typings": yybGoSourceBlock(t, typeat, "declare class YybGo {", "declare class DaiDai {"),
		"node typings":        readYybGoSource(t, "proto3/shaniu.d.ts", "declare class YybGo {", "declare class DaiDai {"),
	}

	for method, path := range yybGoClientAPIEndpoints {
		for name, source := range runtimes {
			if !strings.Contains(source, method+"(") {
				t.Errorf("%s: YybGo.%s is missing", name, method)
			}
			if !strings.Contains(source, `"`+path+`"`) {
				t.Errorf("%s: YybGo.%s route %s is missing", name, method, path)
			}
		}
		for name, source := range typings {
			if !strings.Contains(source, method+"(") {
				t.Errorf("%s: YybGo.%s is missing", name, method)
			}
		}
	}

	for _, method := range []string{"health", "validateAuth"} {
		for name, source := range runtimes {
			if strings.Contains(source, method+"(") {
				t.Errorf("%s: removed YybGo.%s wrapper is still exposed", name, method)
			}
		}
		for name, source := range typings {
			if strings.Contains(source, method+"(") {
				t.Errorf("%s: removed YybGo.%s typing is still exposed", name, method)
			}
		}
	}
}
