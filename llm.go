package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cn-maul/rosetta"
)

// 模型配置存于 settings 表（改完即生效，无需重启）。
const (
	settingLLMEndpoint = "llm_endpoint"
	settingLLMProtocol = "llm_protocol"
	settingLLMModel    = "llm_model"
	settingLLMAPIKey   = "llm_api_key"
)

// llmTimeout 单次模型调用上限（parse 接口的上下文超时）。
const llmTimeout = 90 * time.Second

// llmDefaultProtocol 未配置协议时使用的默认值（OpenAI Chat 兼容面最广）。
const llmDefaultProtocol = string(rosetta.ProtoOpenAIChat)

// llmPlaceholderKey 本地端点（Ollama / vLLM 等）不需要密钥时的占位值：
// rosetta.NewClient 要求 key 非空，填一个不会被校验的假值即可。
const llmPlaceholderKey = "hirezo-local"

// llmProtocolSet 支持的协议白名单（同时也是唯一合法的 llm_protocol 取值）。
var llmProtocolSet = map[string]bool{
	string(rosetta.ProtoOpenAIChat):      true,
	string(rosetta.ProtoOpenAIResponses): true,
	string(rosetta.ProtoAnthropic):       true,
}

// llmConfig 大模型接入配置。
type llmConfig struct {
	Endpoint string
	Protocol string
	Model    string
	APIKey   string
}

// loadLLMConfig 读取模型配置（缺省键返回空串）。
// 存储的 API Key 为「enc:v1:」前缀密文时自动解密为明文；历史明文直接兼容。
func (a *app) loadLLMConfig() (*llmConfig, error) {
	c := &llmConfig{}
	var err error
	db := a.db
	if c.Endpoint, err = getSetting(db, settingLLMEndpoint); err != nil {
		return nil, err
	}
	if c.Protocol, err = getSetting(db, settingLLMProtocol); err != nil {
		return nil, err
	}
	if c.Model, err = getSetting(db, settingLLMModel); err != nil {
		return nil, err
	}
	stored, err := getSetting(db, settingLLMAPIKey)
	if err != nil {
		return nil, err
	}
	if c.APIKey, err = a.decryptLLMKey(stored); err != nil {
		return nil, err
	}
	return c, nil
}

// saveLLMConfig 落库模型配置；API Key 加密后存储（AES-256-GCM），
// 即使 hirezo.db 文件泄露也无法直接读出明文密钥。
func (a *app) saveLLMConfig(endpoint, protocol, model, apiKey string) error {
	enc, err := a.encryptLLMKey(apiKey)
	if err != nil {
		return fmt.Errorf("加密 API Key 失败: %w", err)
	}
	for k, v := range map[string]string{
		settingLLMEndpoint: endpoint,
		settingLLMProtocol: protocol,
		settingLLMModel:    model,
		settingLLMAPIKey:   enc,
	} {
		if err := setSetting(a.db, k, v); err != nil {
			return err
		}
	}
	return nil
}

// ready 判定是否具备发起调用的最小配置（端点 + 模型）。
// 密钥可为空：内网本地推理服务通常无需鉴权。
func (c *llmConfig) ready() bool {
	return strings.TrimSpace(c.Endpoint) != "" && strings.TrimSpace(c.Model) != ""
}

func (c *llmConfig) proto() rosetta.Protocol {
	p := strings.TrimSpace(c.Protocol)
	if p == "" {
		p = llmDefaultProtocol
	}
	return rosetta.Protocol(p)
}

// client 按当前配置构造 rosetta 客户端；配置不完整时返回 error（→ 503）。
func (c *llmConfig) client() (*rosetta.Client, error) {
	if !c.ready() {
		return nil, errors.New("模型尚未配置")
	}
	key := strings.TrimSpace(c.APIKey)
	if key == "" {
		key = llmPlaceholderKey
	}
	return rosetta.NewClient(
		rosetta.WithEndpoint(strings.TrimSpace(c.Endpoint)),
		rosetta.WithAPIKey(key),
		rosetta.WithProtocol(c.proto()),
		rosetta.WithTimeout(llmTimeout),
	)
}

// maskKey 生成 API Key 的回显掩码：只露后 4 位，短值全部打码。
func maskKey(k string) string {
	k = strings.TrimSpace(k)
	if k == "" {
		return ""
	}
	if len([]rune(k)) <= 4 {
		return "****"
	}
	return "****" + string([]rune(k)[len([]rune(k))-4:])
}

// llmConfigView 是 /api/settings/llm 的回显结构：密钥永不回显原文。
type llmConfigView struct {
	Endpoint   string `json:"endpoint"`
	Protocol   string `json:"protocol"`
	Model      string `json:"model"`
	APIKeySet  bool   `json:"api_key_set"`
	APIKeyHint string `json:"api_key_hint"`
	Configured bool   `json:"configured"`
}

func toLLMConfigView(c *llmConfig) llmConfigView {
	v := llmConfigView{
		Endpoint:   c.Endpoint,
		Protocol:   c.Protocol,
		Model:      c.Model,
		APIKeySet:  strings.TrimSpace(c.APIKey) != "",
		APIKeyHint: maskKey(c.APIKey),
		Configured: c.ready(),
	}
	if v.Protocol == "" {
		v.Protocol = llmDefaultProtocol
	}
	return v
}

// llmConfigBody 是 /api/settings/llm 的写入请求体。
// api_key 留空表示不修改；clear_api_key 为 true 时清除已存密钥。
type llmConfigBody struct {
	Endpoint    string `json:"endpoint"`
	Protocol    string `json:"protocol"`
	Model       string `json:"model"`
	APIKey      string `json:"api_key"`
	ClearAPIKey bool   `json:"clear_api_key"`
}

// validate 校验并归一化模型配置；失败返回中文错误信息。
func (b llmConfigBody) validate() (endpoint, protocol, model, apiKey string, errMsg string) {
	endpoint = strings.TrimSpace(b.Endpoint)
	protocol = strings.TrimSpace(b.Protocol)
	model = strings.TrimSpace(b.Model)
	apiKey = strings.TrimSpace(b.APIKey)

	if endpoint == "" {
		return "", "", "", "", "模型端点不能为空"
	}
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", "", "", "", "模型端点须为 http(s):// 地址"
	}
	// 端点禁止携带 query / fragment：query 可能藏凭据，会泄入错误信息与重试日志
	if u.RawQuery != "" || u.Fragment != "" {
		return "", "", "", "", "模型端点不能携带查询参数"
	}
	if protocol == "" {
		protocol = llmDefaultProtocol
	}
	if !llmProtocolSet[protocol] {
		return "", "", "", "", "协议只能是 openai-chat / openai-responses / anthropic"
	}
	if model == "" {
		return "", "", "", "", "模型名称不能为空"
	}
	if len([]rune(model)) > 100 {
		return "", "", "", "", "模型名称过长（最多 100 个字符）"
	}
	if len([]rune(endpoint)) > 300 {
		return "", "", "", "", "模型端点过长"
	}
	return endpoint, protocol, model, apiKey, ""
}

func (a *app) apiLLMConfigGet(w http.ResponseWriter, r *http.Request) {
	if a.requireAdmin(w, r) == nil {
		return
	}
	c, err := a.loadLLMConfig()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "读取模型配置失败")
		return
	}
	jsonResp(w, http.StatusOK, map[string]any{"data": toLLMConfigView(c)})
}

func (a *app) apiLLMConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if a.requireAdmin(w, r) == nil {
		return
	}
	var body llmConfigBody
	if err := decodeJSON(w, r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	endpoint, protocol, model, apiKey, errMsg := body.validate()
	if errMsg != "" {
		jsonError(w, http.StatusBadRequest, errMsg)
		return
	}
	cur, err := a.loadLLMConfig()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "读取模型配置失败")
		return
	}
	nextKey := cur.APIKey
	switch {
	case body.ClearAPIKey:
		nextKey = ""
	case apiKey != "":
		nextKey = apiKey
	}
	if err := a.saveLLMConfig(endpoint, protocol, model, nextKey); err != nil {
		jsonError(w, http.StatusInternalServerError, "保存模型配置失败")
		return
	}
	jsonResp(w, http.StatusOK, map[string]any{"data": toLLMConfigView(&llmConfig{
		Endpoint: endpoint, Protocol: protocol, Model: model, APIKey: nextKey,
	})})
}

// errModelNotConfigured 统一的「未配置」哨兵错误。
var errModelNotConfigured = errors.New("模型尚未配置，请先在「设置 → 模型配置」中填写端点与模型")

// llmErrorToHTTP 把模型调用失败映射为 HTTP 状态与中文提示。
// 返回 (code, message, detail)；detail 仅写日志，不下发给客户端（可能含上游回显内容）。
func llmErrorToHTTP(err error) (int, string, string) {
	if err == nil {
		return 0, "", ""
	}
	if errors.Is(err, errModelNotConfigured) {
		return http.StatusServiceUnavailable, errModelNotConfigured.Error(), ""
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, "模型调用超时，请稍后重试", err.Error()
	}
	var apiErr *rosetta.APIError
	if errors.As(err, &apiErr) {
		detail := fmt.Sprintf("上游 %d %s", apiErr.StatusCode, apiErr.Message)
		if apiErr.Retryable {
			return http.StatusBadGateway, "上游模型服务繁忙，请重试", detail
		}
		return http.StatusBadGateway, "模型服务返回错误，请检查端点与模型配置", detail
	}
	if errors.Is(err, rosetta.ErrInvalidRequest) {
		return http.StatusBadGateway, "请求未被模型服务接受，请检查端点与协议配置", err.Error()
	}
	if errors.Is(err, rosetta.ErrNoEndpoint) || errors.Is(err, rosetta.ErrNoAPIKey) {
		return http.StatusServiceUnavailable, errModelNotConfigured.Error(), err.Error()
	}
	return http.StatusBadGateway, "模型调用失败", err.Error()
}
