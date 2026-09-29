package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestSecretKeyRoundTrip 验证 AES-256-GCM 加解密往返与空值处理。
func TestSecretKeyRoundTrip(t *testing.T) {
	k, err := LoadSecretKey("", t.TempDir()+"/hirezo.db")
	if err != nil {
		t.Fatalf("LoadSecretKey 失败: %v", err)
	}

	// 空值不加密
	enc, err := k.Encrypt("")
	if err != nil || enc != "" {
		t.Fatalf("Encrypt(\"\") = (%q, %v)，期望空", enc, err)
	}
	dec, err := k.Decrypt("")
	if err != nil || dec != "" {
		t.Fatalf("Decrypt(\"\") = (%q, %v)，期望空", dec, err)
	}

	plain := "sk-abcdefghijklmnopqrstuvwxyz"
	enc, err = k.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt 失败: %v", err)
	}
	if !strings.HasPrefix(enc, secretPrefix) {
		t.Fatalf("密文缺少前缀: %q", enc)
	}
	if strings.Contains(enc, plain) {
		t.Fatal("密文泄漏了明文内容")
	}

	dec, err = k.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt 失败: %v", err)
	}
	if dec != plain {
		t.Fatalf("解密结果 = %q，期望 %q", dec, plain)
	}

	// 同一明文两次加密结果不同（随机 nonce）
	enc2, _ := k.Encrypt(plain)
	if enc == enc2 {
		t.Fatal("两次加密结果相同（nonce 未随机化）")
	}
}

// TestSecretKeyLegacyPlaintext 历史明文（无前缀）应原样透传，保证旧版本数据可读。
func TestSecretKeyLegacyPlaintext(t *testing.T) {
	k, err := LoadSecretKey("", t.TempDir()+"/hirezo.db")
	if err != nil {
		t.Fatalf("LoadSecretKey 失败: %v", err)
	}
	legacy := "sk-legacy-plain-key"
	got, err := k.Decrypt(legacy)
	if err != nil || got != legacy {
		t.Fatalf("legacy 明文透传 = (%q, %v)，期望原样返回", got, err)
	}
}

// TestSecretKeyWrongKeyFails 用错误的密钥解密应明确报错（提示密钥文件不一致）。
func TestSecretKeyWrongKeyFails(t *testing.T) {
	k1, err := LoadSecretKey("", t.TempDir()+"/db1.sqlite")
	if err != nil {
		t.Fatalf("k1 失败: %v", err)
	}
	enc, err := k1.Encrypt("sk-secret-value")
	if err != nil {
		t.Fatalf("Encrypt 失败: %v", err)
	}

	k2, err := LoadSecretKey("", t.TempDir()+"/db2.sqlite")
	if err != nil {
		t.Fatalf("k2 失败: %v", err)
	}
	if _, err := k2.Decrypt(enc); err == nil {
		t.Fatal("错误密钥解密应当失败")
	}
}

// TestSecretKeyPersistence 密钥文件落盘后可重新加载（重启后仍能解密）。
func TestSecretKeyPersistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/hirezo.db"
	k1, err := LoadSecretKey("", dbPath)
	if err != nil {
		t.Fatalf("首次加载失败: %v", err)
	}
	enc, err := k1.Encrypt("sk-persist-me")
	if err != nil {
		t.Fatalf("Encrypt 失败: %v", err)
	}

	k2, err := LoadSecretKey("", dbPath)
	if err != nil {
		t.Fatalf("二次加载失败: %v", err)
	}
	dec, err := k2.Decrypt(enc)
	if err != nil || dec != "sk-persist-me" {
		t.Fatalf("重启后解密 = (%q, %v)", dec, err)
	}
}

// TestLLMKeyStoredEncrypted 通过 API 写入密钥后，数据库中不得出现明文
// （app.secret 为 nil 的测试环境除外 —— 这里用真实密钥注入验证落库形态）。
func TestLLMKeyStoredEncrypted(t *testing.T) {
	a, srv := newTestApp(t)
	// 注入真实加密密钥，模拟生产启动路径
	k, err := LoadSecretKey("", t.TempDir()+"/hirezo.db")
	if err != nil {
		t.Fatalf("LoadSecretKey 失败: %v", err)
	}
	a.secret = k

	cookie := loginAs(t, srv)
	resp := req(t, srv, "PUT", "/api/settings/llm", map[string]any{
		"endpoint": "https://api.example.com/v1",
		"protocol": "openai-chat",
		"model":    "deepseek-chat",
		"api_key":  "sk-topsecret-123456",
	}, cookie)
	if resp.StatusCode != 200 {
		t.Fatalf("保存配置失败: %d", resp.StatusCode)
	}

	stored, err := getSetting(a.db, settingLLMAPIKey)
	if err != nil {
		t.Fatalf("读设置失败: %v", err)
	}
	if !strings.HasPrefix(stored, secretPrefix) {
		t.Fatalf("落库值应为加密形态，实际: %q", stored)
	}
	if strings.Contains(stored, "sk-topsecret-123456") {
		t.Fatal("数据库明文泄漏了 API Key")
	}

	// loadLLMConfig 解出明文，配置可用
	cfg, err := a.loadLLMConfig()
	if err != nil {
		t.Fatalf("loadLLMConfig 失败: %v", err)
	}
	if cfg.APIKey != "sk-topsecret-123456" {
		t.Fatalf("解密后的 API Key = %q", cfg.APIKey)
	}
	// API 回显仍是掩码
	resp = req(t, srv, "GET", "/api/settings/llm", nil, cookie)
	if resp.StatusCode != 200 {
		t.Fatalf("读配置失败: %d", resp.StatusCode)
	}
	view := decodeLLMView(t, resp)
	b, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("序列化视图失败: %v", err)
	}
	if strings.Contains(string(b), "sk-topsecret-123456") {
		t.Fatal("模型接口回显了明文 API Key")
	}
}

// TestSecretKeyFileFormat 密钥文件格式：base64 32 字节 + 换行。
func TestSecretKeyFileFormat(t *testing.T) {
	dir := t.TempDir()
	k, err := LoadSecretKey("", dir+"/app.db")
	if err != nil {
		t.Fatalf("LoadSecretKey 失败: %v", err)
	}
	raw, err := os.ReadFile(dir + "/.hirezo-secret")
	if err != nil {
		t.Fatalf("读取密钥文件失败: %v", err)
	}
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("密钥文件解码失败: %v", err)
	}
	if len(b) != 32 {
		t.Fatalf("密钥长度 = %d，期望 32", len(b))
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatal("密钥文件应以换行结尾")
	}
	_ = k
}
