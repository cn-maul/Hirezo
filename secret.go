package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// secretPrefix 标记已加密的 API Key 存储值；无前缀的值视为历史明文
// （读取兼容：原样返回；下次保存时自动升级为密文）。
const secretPrefix = "enc:v1:"

// secretKeyFile 未显式指定密钥文件时使用的默认文件名（生成在数据库同目录）。
const secretKeyFile = ".hirezo-secret"

// SecretKey 应用加密密钥（32 字节，AES-256-GCM 使用）。
type SecretKey struct {
	key    [32]byte
	loaded bool
}

// secretKeyPath 解析密钥文件路径，优先级：-secret-key flag > HIREZO_SECRET_FILE 环境变量 > 数据库同目录默认文件。
func secretKeyPath(flagPath, dbPath string) string {
	path := strings.TrimSpace(flagPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("HIREZO_SECRET_FILE"))
	}
	if path == "" {
		path = filepath.Join(filepath.Dir(dbPath), secretKeyFile)
	}
	return path
}

// LoadSecretKey 加载或生成应用加密密钥（权限 0600）。
// 密钥文件丢失会导致已加密的 API Key 无法解密 —— 部署时请与数据库一起备份。
func LoadSecretKey(flagPath, dbPath string) (*SecretKey, error) {
	path := secretKeyPath(flagPath, dbPath)
	raw, err := os.ReadFile(path)
	if err == nil {
		k, err := parseSecretKeyFile(raw)
		if err != nil {
			return nil, fmt.Errorf("密钥文件 %s 无效: %w", path, err)
		}
		return k, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("读取密钥文件 %s 失败: %w", path, err)
	}

	k := &SecretKey{loaded: true}
	if _, err := rand.Read(k.key[:]); err != nil {
		return nil, fmt.Errorf("生成加密密钥失败: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("创建密钥目录 %s 失败: %w", dir, err)
	}
	enc := base64.StdEncoding.EncodeToString(k.key[:])
	if err := os.WriteFile(path, []byte(enc+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("写入密钥文件 %s 失败: %w", path, err)
	}
	return k, nil
}

// parseSecretKeyFile 解析密钥文件：base64 编码的 32 字节（容忍换行/空白）。
func parseSecretKeyFile(raw []byte) (*SecretKey, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("密钥文件须为 base64 编码的 32 字节内容: %w", err)
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("密钥长度须为 32 字节，实际 %d", len(b))
	}
	var k SecretKey
	copy(k.key[:], b)
	k.loaded = true
	return &k, nil
}

// Encrypt 加密明文，返回带 secretPrefix 前缀的密文；空输入返回空串（不加密）。
func (k *SecretKey) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	block, err := aes.NewCipher(k.key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return secretPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 解密 Encrypt 的产物；无前缀的历史明文原样返回。
func (k *SecretKey) Decrypt(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, secretPrefix) {
		// 历史明文（旧版本写入）：读回兼容
		return stored, nil
	}
	if !k.loaded {
		return "", errors.New("应用加密密钥未加载，无法解密 API Key")
	}
	sealed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, secretPrefix))
	if err != nil {
		return "", fmt.Errorf("API Key 密文格式损坏: %w", err)
	}
	block, err := aes.NewCipher(k.key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(sealed) < gcm.NonceSize()+gcm.Overhead() {
		return "", errors.New("API Key 密文不完整")
	}
	nonce := sealed[:gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, sealed[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("API Key 解密失败（密钥与写入时不一致？请检查密钥文件）")
	}
	return string(plain), nil
}

// ---- app 级封装：未配置密钥（测试/兼容）时明文透传 ----

// encryptLLMKey 加密 API Key 后落库；app.secret 为 nil 时原样返回（测试模式）。
func (a *app) encryptLLMKey(plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	if a.secret == nil {
		return plain, nil
	}
	return a.secret.Encrypt(plain)
}

// decryptLLMKey 还原存储的 API Key；历史明文直接透传，密文解密失败返回错误。
func (a *app) decryptLLMKey(stored string) (string, error) {
	if stored == "" || !strings.HasPrefix(stored, secretPrefix) {
		return stored, nil
	}
	if a.secret == nil {
		return "", errors.New("API Key 为加密存储，但进程未配置解密密钥（-secret-key）")
	}
	return a.secret.Decrypt(stored)
}
