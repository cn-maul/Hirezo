package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// happyExtract 完整可入库的抽取桩结果。
func happyExtract(t *testing.T) *extractResult {
	t.Helper()
	return &extractResult{
		Fields: map[string]extractField{
			"name":     rawField(t, "张三", "张三 男 28岁", 0.99),
			"gender":   rawField(t, "男", "张三 男", 0.98),
			"age":      rawField(t, "28", "28岁", 0.95),
			"subject":  rawField(t, "数学", "数学教师", 0.9),
			"phone":    rawField(t, "13800138000", "电话：13800138000", 0.95),
		},
		Summary: "中学数学教师",
	}
}

// TestResumeCommitStoresFile 端到端：识别→草稿临时文件→commit 归档→详情带 resume_file。
func TestResumeCommitStoresFile(t *testing.T) {
	a, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	configureLLM(t, a)
	a.extractHook = stubExtract(happyExtract(t), nil)

	docx := buildDocx(t, sampleDocxXML)
	resp := uploadResume(t, srv, cookie, "resume.docx", docx)
	requireStatus(t, resp, http.StatusCreated)
	did := draftID(t, resp)

	// 草稿临时文件应已落盘
	tmp := filepath.Join(a.resumeDraftDir, int64String(did)+".docx")
	if _, err := os.Stat(tmp); err != nil {
		t.Fatalf("草稿原件未保存: %v", err)
	}

	commitBody := map[string]any{"fields": map[string]any{
		"name": "张三", "gender": "male", "age": 28,
		"subject": "数学", "has_cert": 0, "phone": "13800138000",
	}}
	resp = req(t, srv, http.MethodPost, "/api/resume/drafts/"+int64String(did)+"/commit", commitBody, cookie)
	requireStatus(t, resp, http.StatusCreated)
	var created struct {
		Data Teacher `json:"data"`
	}
	decodeBody(t, resp, &created)
	wantFile := int64String(created.Data.ID) + ".docx"
	if created.Data.ResumeFile != wantFile {
		t.Fatalf("resume_file = %q，期望 %q", created.Data.ResumeFile, wantFile)
	}
	// 永久目录有文件且内容一致；临时目录已清空
	perm := filepath.Join(a.resumeDir, wantFile)
	b, err := os.ReadFile(perm)
	if err != nil {
		t.Fatalf("归档文件不存在: %v", err)
	}
	if string(b) != string(docx) {
		t.Fatal("归档内容与上传内容不一致")
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatal("commit 后临时文件应被移走")
	}
}

// TestResumeFileDownload 下载鉴权、内容与安全头。
func TestResumeFileDownload(t *testing.T) {
	a, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	configureLLM(t, a)
	a.extractHook = stubExtract(happyExtract(t), nil)

	docx := buildDocx(t, sampleDocxXML)
	resp := uploadResume(t, srv, cookie, "resume.docx", docx)
	requireStatus(t, resp, http.StatusCreated)
	did := draftID(t, resp)
	resp = req(t, srv, http.MethodPost, "/api/resume/drafts/"+int64String(did)+"/commit", map[string]any{"fields": map[string]any{
		"name": "张三", "gender": "male", "age": 28, "subject": "数学", "phone": "13800138000",
	}}, cookie)
	requireStatus(t, resp, http.StatusCreated)
	var created struct {
		Data Teacher `json:"data"`
	}
	decodeBody(t, resp, &created)
	idPath := "/api/teachers/" + int64String(created.Data.ID) + "/resume"

	// 未登录 → 401
	if resp := req(t, srv, http.MethodGet, idPath, nil, nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("未登录下载 status = %d，期望 401", resp.StatusCode)
	}

	// 登录下载 → 200 + 内容一致
	resp = req(t, srv, http.MethodGet, idPath, nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	if ct := resp.Header.Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); cd == "" || !strings.Contains(cd, "filename*=UTF-8''") {
		t.Errorf("Content-Disposition = %q", cd)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取下载内容失败: %v", err)
	}
	if string(b) != string(docx) {
		t.Fatal("下载内容与上传内容不一致")
	}

	// 手工创建的教师无简历 → 404
	resp = req(t, srv, http.MethodPost, "/api/teachers", map[string]any{"name": "李四", "gender": "female"}, cookie)
	requireStatus(t, resp, http.StatusCreated)
	var p2 struct {
		Data Teacher `json:"data"`
	}
	decodeBody(t, resp, &p2)
	if resp := req(t, srv, http.MethodGet, "/api/teachers/"+int64String(p2.Data.ID)+"/resume", nil, cookie); resp.StatusCode != http.StatusNotFound {
		t.Errorf("无简历下载 status = %d，期望 404", resp.StatusCode)
	}
}

// TestTeacherDeleteRemovesResumeFile 单删与批删都清理归档文件。
func TestTeacherDeleteRemovesResumeFile(t *testing.T) {
	a, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	configureLLM(t, a)
	a.extractHook = stubExtract(happyExtract(t), nil)

	commitOne := func() int64 {
		t.Helper()
		resp := uploadResume(t, srv, cookie, "resume.docx", buildDocx(t, sampleDocxXML))
		requireStatus(t, resp, http.StatusCreated)
		did := draftID(t, resp)
		resp = req(t, srv, http.MethodPost, "/api/resume/drafts/"+int64String(did)+"/commit", map[string]any{"fields": map[string]any{
			"name": "张三", "gender": "male", "age": 28, "subject": "数学", "phone": "13800138000",
		}}, cookie)
		requireStatus(t, resp, http.StatusCreated)
		var c struct {
			Data Teacher `json:"data"`
		}
		decodeBody(t, resp, &c)
		return c.Data.ID
	}

	id1, id2 := commitOne(), commitOne()
	p1, p2 := filepath.Join(a.resumeDir, int64String(id1)+".docx"), filepath.Join(a.resumeDir, int64String(id2)+".docx")

	resp := req(t, srv, http.MethodDelete, "/api/teachers/"+int64String(id1), nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	if _, err := os.Stat(p1); !os.IsNotExist(err) {
		t.Fatal("单删后简历文件应被清理")
	}

	resp = req(t, srv, http.MethodPost, "/api/teachers/batch-delete", map[string]any{"ids": []int64{id2}}, cookie)
	requireStatus(t, resp, http.StatusOK)
	if _, err := os.Stat(p2); !os.IsNotExist(err) {
		t.Fatal("批删后简历文件应被清理")
	}
}

// TestDraftDeleteRemovesTempFile 丢弃草稿清理临时原件。
func TestDraftDeleteRemovesTempFile(t *testing.T) {
	a, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	configureLLM(t, a)
	a.extractHook = stubExtract(happyExtract(t), nil)

	resp := uploadResume(t, srv, cookie, "resume.docx", buildDocx(t, sampleDocxXML))
	requireStatus(t, resp, http.StatusCreated)
	did := draftID(t, resp)
	tmp := filepath.Join(a.resumeDraftDir, int64String(did)+".docx")
	if _, err := os.Stat(tmp); err != nil {
		t.Fatalf("草稿原件未保存: %v", err)
	}
	resp = req(t, srv, http.MethodDelete, "/api/resume/drafts/"+int64String(did), nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatal("丢弃草稿后临时文件应被删除")
	}
}

// TestResumeFileMigration 旧库（无 resume_file 列）启动迁移后列存在且默认空。
func TestResumeFileMigration(t *testing.T) {
	a, srv := newTestApp(t)
	if _, err := a.db.Exec("ALTER TABLE teachers DROP COLUMN resume_file"); err != nil {
		t.Skipf("当前 sqlite 驱动不支持 DROP COLUMN: %v", err)
	}
	if err := migrateDB(a.db); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	has, err := hasColumn(a.db, "teachers", "resume_file")
	if err != nil || !has {
		t.Fatalf("迁移后应存在 resume_file 列 (has=%v err=%v)", has, err)
	}
	_ = srv
}

// ---------- 小工具 ----------

func int64String(v int64) string { return strconv.FormatInt(v, 10) }
