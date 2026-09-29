package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

// newTestApp 构建完整应用（含中间件链），使用临时 SQLite 文件库。
func newTestApp(t *testing.T) (*app, *httptest.Server) {
	t.Helper()
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("打开测试库失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := initDB(db); err != nil {
		t.Fatalf("initDB 失败: %v", err)
	}
	if err := migrateDB(db); err != nil {
		t.Fatalf("migrateDB 失败: %v", err)
	}
	if _, err := setupDefaultAdmin(db, "admin123"); err != nil {
		t.Fatalf("setupDefaultAdmin 失败: %v", err)
	}
	a := &app{
		db:           db,
		auth:         newAuthStore(),
		loginLimiter: newRateLimiter(10, time.Minute),
	}
	srv := httptest.NewServer(securityHeaders(a.authMiddleware(a.routes())))
	t.Cleanup(srv.Close)
	return a, srv
}

// req 发起一次请求；cookie 可为 nil（未登录）。
func req(t *testing.T, srv *httptest.Server, method, path string, body any, cookie *http.Cookie) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	httpReq, err := http.NewRequest(method, srv.URL+path, rdr)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		httpReq.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("请求 %s %s 失败: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// decodeBody 解析 JSON 响应到 v。
func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
}

// login 登录并返回会话 Cookie；失败则终止测试。
func login(t *testing.T, srv *httptest.Server, username, password string) *http.Cookie {
	t.Helper()
	resp := req(t, srv, http.MethodPost, "/api/login",
		map[string]string{"username": username, "password": password}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("登录失败：状态码 %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	t.Fatal("登录响应未包含会话 Cookie")
	return nil
}

// loginAs 使用默认管理员账号登录。
func loginAs(t *testing.T, srv *httptest.Server) *http.Cookie {
	t.Helper()
	return login(t, srv, "admin", "admin123")
}

// requireStatus 断言状态码，失败时打印响应体便于定位。
func requireStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("状态码 = %d，期望 %d；响应体: %s", resp.StatusCode, want, body)
	}
}

// errorOf 解析错误信封中的 message。
func errorOf(t *testing.T, resp *http.Response) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decodeBody(t, resp, &payload)
	return payload.Error.Message
}
