package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestHealthPublic(t *testing.T) {
	_, srv := newTestApp(t)
	resp := req(t, srv, http.MethodGet, "/api/health", nil, nil)
	requireStatus(t, resp, http.StatusOK)
}

func TestUnknownAPIReturns404JSON(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)
	resp := req(t, srv, http.MethodGet, "/api/not-exist", nil, c)
	requireStatus(t, resp, http.StatusNotFound)
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q，期望 JSON", ct)
	}
}

func TestProtectedAPIRequiresLogin(t *testing.T) {
	_, srv := newTestApp(t)
	for _, path := range []string{"/api/teachers", "/api/dictionaries/subject", "/api/settings"} {
		method := http.MethodGet
		if path == "/api/settings" {
			// settings 为公开读取，改用需要管理员的写接口验证
			resp := req(t, srv, http.MethodPut, "/api/settings", map[string]string{"site_name": "x"}, nil)
			requireStatus(t, resp, http.StatusUnauthorized)
			continue
		}
		resp := req(t, srv, method, path, nil, nil)
		requireStatus(t, resp, http.StatusUnauthorized)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	_, srv := newTestApp(t)
	resp := req(t, srv, http.MethodPost, "/api/login",
		map[string]string{"username": "admin", "password": "wrong-pass"}, nil)
	requireStatus(t, resp, http.StatusUnauthorized)
	if msg := errorOf(t, resp); msg != "用户名或密码错误" {
		t.Fatalf("错误信息 = %q", msg)
	}
}

func TestLoginLogoutAndStatus(t *testing.T) {
	_, srv := newTestApp(t)

	// 未登录状态
	resp := req(t, srv, http.MethodGet, "/api/auth/status", nil, nil)
	requireStatus(t, resp, http.StatusOK)
	var st struct {
		Data struct {
			OK   bool  `json:"ok"`
			User *User `json:"user"`
		} `json:"data"`
	}
	decodeBody(t, resp, &st)
	if st.Data.OK {
		t.Fatal("未登录时 auth/status 应为 ok=false")
	}

	// 登录
	cookie := loginAs(t, srv)
	resp = req(t, srv, http.MethodGet, "/api/auth/status", nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &st)
	if !st.Data.OK || st.Data.User == nil || st.Data.User.Username != "admin" {
		t.Fatalf("登录后 auth/status 异常: %+v", st.Data)
	}

	// 登出后会话失效
	resp = req(t, srv, http.MethodPost, "/api/logout", nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	resp = req(t, srv, http.MethodGet, "/api/teachers", nil, cookie)
	requireStatus(t, resp, http.StatusUnauthorized)
}

func TestPasswordStoredHashed(t *testing.T) {
	a, _ := newTestApp(t)
	_, stored, err := getUserAuth(a.db, "admin")
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if !strings.HasPrefix(stored, "$2") {
		t.Fatalf("密码未以 bcrypt 存储，前缀 = %q", stored[:min(3, len(stored))])
	}
	if !checkPassword(stored, "admin123") {
		t.Fatal("bcrypt 校验初始密码失败")
	}
}

func TestPasswordChangeRevokesOtherSessions(t *testing.T) {
	_, srv := newTestApp(t)
	c1 := loginAs(t, srv)
	c2 := loginAs(t, srv)

	resp := req(t, srv, http.MethodGet, "/api/teachers", nil, c2)
	requireStatus(t, resp, http.StatusOK)

	// c1 改密
	resp = req(t, srv, http.MethodPut, "/api/profile/password",
		map[string]string{"old_password": "admin123", "new_password": "newpass99"}, c1)
	requireStatus(t, resp, http.StatusOK)

	// 其他会话失效，当前会话保留
	resp = req(t, srv, http.MethodGet, "/api/teachers", nil, c2)
	requireStatus(t, resp, http.StatusUnauthorized)
	resp = req(t, srv, http.MethodGet, "/api/teachers", nil, c1)
	requireStatus(t, resp, http.StatusOK)

	// 新密码可登录
	resp = req(t, srv, http.MethodPost, "/api/login",
		map[string]string{"username": "admin", "password": "newpass99"}, nil)
	requireStatus(t, resp, http.StatusOK)
}

func TestPasswordChangeRejectsWrongOld(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)
	resp := req(t, srv, http.MethodPut, "/api/profile/password",
		map[string]string{"old_password": "nope", "new_password": "newpass99"}, c)
	requireStatus(t, resp, http.StatusBadRequest)
}

func TestLoginRateLimit(t *testing.T) {
	_, srv := newTestApp(t)
	for i := 0; i < 10; i++ {
		resp := req(t, srv, http.MethodPost, "/api/login",
			map[string]string{"username": "admin", "password": "bad"}, nil)
		_ = resp
	}
	resp := req(t, srv, http.MethodPost, "/api/login",
		map[string]string{"username": "admin", "password": "bad"}, nil)
	requireStatus(t, resp, http.StatusTooManyRequests)
}

func TestSecurityHeaders(t *testing.T) {
	_, srv := newTestApp(t)
	resp := req(t, srv, http.MethodGet, "/api/health", nil, nil)
	for k, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "same-origin",
	} {
		if got := resp.Header.Get(k); got != want {
			t.Errorf("%s = %q，期望 %q", k, got, want)
		}
	}
	if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("Content-Security-Policy 缺失: %q", csp)
	}
	// 简历原件预览用 blob: iframe，frame-src 必须放行 blob
	if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "frame-src 'self' blob:") {
		t.Errorf("Content-Security-Policy 未放行 blob frame: %q", csp)
	}
}

func TestSettingsWhitelist(t *testing.T) {
	a, srv := newTestApp(t)
	if err := setSetting(a.db, "site_name", "Hirezo 测试"); err != nil {
		t.Fatalf("写设置失败: %v", err)
	}
	// 敏感键即使写入也不会出现在公开接口
	if err := setSetting(a.db, "api_key", "secret-value"); err != nil {
		t.Fatalf("写设置失败: %v", err)
	}

	resp := req(t, srv, http.MethodGet, "/api/settings", nil, nil)
	requireStatus(t, resp, http.StatusOK)
	var payload struct {
		Data map[string]string `json:"data"`
	}
	decodeBody(t, resp, &payload)
	if payload.Data["site_name"] != "Hirezo 测试" {
		t.Fatalf("site_name = %q", payload.Data["site_name"])
	}
	if _, ok := payload.Data["api_key"]; ok {
		t.Fatal("敏感键 api_key 不应出现在公开设置中")
	}
}

func TestMigrateIdempotent(t *testing.T) {
	a, _ := newTestApp(t)
	for i := 0; i < 3; i++ {
		if err := migrateDB(a.db); err != nil {
			t.Fatalf("第 %d 次迁移失败: %v", i+1, err)
		}
	}
}

func TestDictSeedExists(t *testing.T) {
	a, _ := newTestApp(t)
	for kind, wantMin := range map[string]int{dictKindSubject: 10, dictKindEducation: 4} {
		ds, err := allDictionaries(a.db, kind)
		if err != nil {
			t.Fatalf("查询 %s 字典失败: %v", kind, err)
		}
		if len(ds) < wantMin {
			t.Fatalf("%s 种子数量 = %d，期望 ≥ %d", kind, len(ds), wantMin)
		}
	}
}
