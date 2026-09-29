package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"archive/zip"
)

// ---------- 工具 ----------

// rawField 把任意值转成模型返回的 extractField。
func rawField(t *testing.T, v any, evidence string, confidence float64) extractField {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("序列化字段值失败: %v", err)
	}
	return extractField{Value: b, Evidence: evidence, Confidence: confidence}
}

// buildDocx 生成一个只含 word/document.xml 的最小 docx（zip）。
func buildDocx(t *testing.T, documentXML string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("创建 zip 条目失败: %v", err)
	}
	if _, err := io.WriteString(f, documentXML); err != nil {
		t.Fatalf("写入 document.xml 失败: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("关闭 zip 失败: %v", err)
	}
	return buf.Bytes()
}

// sampleDocxXML 段落 + 表格的典型简历版式。
const sampleDocxXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
<w:p><w:r><w:t>张三 男 28岁 数学教师</w:t></w:r></w:p>
<w:p><w:r><w:t>电话：13800138000</w:t></w:r></w:p>
<w:tbl>
 <w:tr>
  <w:tc><w:p><w:r><w:t>毕业院校</w:t></w:r></w:p></w:tc>
  <w:tc><w:p><w:r><w:t>北京大学</w:t></w:r></w:p></w:tc>
 </w:tr>
 <w:tr>
  <w:tc><w:p><w:r><w:t>专业</w:t></w:r></w:p></w:tc>
  <w:tc><w:p><w:r><w:t>数学与应用数学</w:t></w:r></w:p></w:tc>
 </w:tr>
</w:tbl>
<w:p><w:r><w:t>2018-2022 任教于实验中学</w:t></w:r></w:p>
</w:body></w:document>`

// uploadResume 以 multipart/form-data 方式上传简历文件。
func uploadResume(t *testing.T, srv *httptest.Server, cookie *http.Cookie, filename string, data []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("构造 multipart 失败: %v", err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatalf("写入 multipart 文件失败: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("关闭 multipart writer 失败: %v", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, srv.URL+"/api/resume/parse", &buf)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	if cookie != nil {
		httpReq.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("上传请求失败: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// stubExtract 装配固定抽取结果的桩。
func stubExtract(ex *extractResult, err error) extractHookFunc {
	return func(context.Context, resumeExtractInput) (*extractResult, error) {
		return ex, err
	}
}

// configureLLM 写入可用的模型配置（解析接口的 ready 检查依赖）。
func configureLLM(t *testing.T, a *app) {
	t.Helper()
	for k, v := range map[string]string{
		settingLLMEndpoint: "https://api.example.com/v1",
		settingLLMProtocol: llmDefaultProtocol,
		settingLLMModel:    "test-model",
		settingLLMAPIKey:   "sk-test-abcdef1234",
	} {
		if err := setSetting(a.db, k, v); err != nil {
			t.Fatalf("写模型配置失败: %v", err)
		}
	}
}

// draftID 解析 201 响应里的草稿 id。
func draftID(t *testing.T, resp *http.Response) int64 {
	t.Helper()
	var payload struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, resp, &payload)
	if payload.Data.ID <= 0 {
		t.Fatalf("草稿 id 非法: %d", payload.Data.ID)
	}
	return payload.Data.ID
}

// decodeLLMView 解析 {data:{...}} 信封中的模型配置。
func decodeLLMView(t *testing.T, resp *http.Response) llmConfigView {
	t.Helper()
	var payload struct {
		Data llmConfigView `json:"data"`
	}
	decodeBody(t, resp, &payload)
	return payload.Data
}

// ---------- 上传校验 ----------

func TestSniffResumeKind(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
		ok   bool
	}{
		{"zip/docx", []byte{0x50, 0x4B, 0x03, 0x04, 0, 0}, "docx", true},
		{"pdf", []byte("%PDF-1.4 rest"), "pdf", true},
		{"jpg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0}, "", false},
		{"png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}, "", false},
		{"doc(ole)", []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1}, "", false},
		{"text", []byte("个人简历"), "", false},
		{"empty", nil, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := sniffResumeKind(c.data)
			if got != c.want || ok != c.ok {
				t.Fatalf("sniff(%s) = (%q,%v)，期望 (%q,%v)", c.name, got, ok, c.want, c.ok)
			}
		})
	}
}

func TestSanitizeUploadName(t *testing.T) {
	cases := map[string]string{
		"../../etc/passwd": "passwd",
		`C:\temp\简历.docx`:  "简历.docx",
		"张三 的简历.pdf":       "张三 的简历.pdf",
		"":                 "resume",
		"..":               "resume",
		"a\x00b.txt":       "ab.txt",
	}
	for in, want := range cases {
		if got := sanitizeUploadName(in); got != want {
			t.Errorf("sanitize(%q) = %q，期望 %q", in, got, want)
		}
	}
}

// ---------- docx 解析 ----------

func TestDocxTextParagraphAndTable(t *testing.T) {
	data := buildDocx(t, sampleDocxXML)
	text, err := docxText(data)
	if err != nil {
		t.Fatalf("docxText 失败: %v", err)
	}
	lines := strings.Split(text, "\n")
	want := []string{
		"张三 男 28岁 数学教师",
		"电话：13800138000",
		"毕业院校 | 北京大学",
		"专业 | 数学与应用数学",
		"2018-2022 任教于实验中学",
	}
	if len(lines) != len(want) {
		t.Fatalf("行数 = %d，期望 %d；输出:\n%s", len(lines), len(want), text)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("第 %d 行 = %q，期望 %q", i+1, lines[i], want[i])
		}
	}
}

func TestDocxTextRejectsNonDocx(t *testing.T) {
	if _, err := docxText([]byte("PK\x03\x04 not really a zip")); err == nil {
		t.Fatal("非 docx 内容应当报错")
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	_, _ = zw.Create("other.xml")
	_ = zw.Close()
	if _, err := docxText(buf.Bytes()); err == nil {
		t.Fatal("缺少 word/document.xml 应当报错")
	}
}

// ---------- PDF ----------

// buildMinimalPDF 生成带文本层的最小合法 PDF（含 xref 偏移）。
func buildMinimalPDF(t *testing.T, text string) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	stream := fmt.Sprintf("BT /F1 12 Tf 72 720 Td (%s) Tj ET", text)
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >> stream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	offsets := make([]int, 0, len(objs))
	for i, body := range objs {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return buf.Bytes()
}

func TestPDFTextLayer(t *testing.T) {
	data := buildMinimalPDF(t, "Zhang San phone 13800138000 teacher of mathematics at No.1 middle school")
	text, pages, err := pdfText(data, resumeMaxPages)
	if err != nil {
		t.Fatalf("pdfText 失败: %v", err)
	}
	if pages != 1 {
		t.Errorf("页数 = %d，期望 1", pages)
	}
	if !strings.Contains(text, "13800138000") {
		t.Errorf("未抽取到文本层内容: %q", text)
	}
	if meaningfulRune(text) < resumeMinTextRune {
		t.Errorf("有效字符数 = %d，应被判定为有文本层", meaningfulRune(text))
	}
}

func TestPDFCorruptRejected(t *testing.T) {
	if _, _, err := pdfText([]byte("%PDF-1.4 garbage"), resumeMaxPages); err == nil {
		t.Fatal("损坏的 PDF 应当报错")
	}
}

// ---------- 模型输出解析 ----------

func TestParseExtractJSON(t *testing.T) {
	okJSON := `{"fields":{"name":{"value":"张三","evidence":"张三","confidence":0.9}},"summary":"数学教师"}`
	cases := []struct {
		name string
		in   string
		fail bool
	}{
		{"纯 JSON", okJSON, false},
		{"代码围栏", "```json\n" + okJSON + "\n```", false},
		{"前后有解释", "好的，结果如下：\n" + okJSON + "\n希望有帮助", false},
		{"空", "   ", true},
		{"非 JSON", "我觉得这个简历挺好的", true},
		{"截断的 JSON", `{"fields":{"name":`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseExtractJSON(c.in)
			if c.fail {
				if err == nil {
					t.Fatal("期望解析失败")
				}
				return
			}
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			if got.Fields["name"].text() != "张三" {
				t.Errorf("name = %q", got.Fields["name"].text())
			}
		})
	}
}

func TestExtractFieldValueCoercion(t *testing.T) {
	f := rawField(t, 28, "", 0)
	if n, ok := f.num(); !ok || n != 28 {
		t.Errorf("num(28) = (%d,%v)", n, ok)
	}
	f = rawField(t, "28岁", "", 0)
	if n, ok := f.num(); !ok || n != 28 {
		t.Errorf("num(\"28岁\") = (%d,%v)", n, ok)
	}
	f = rawField(t, "二十八", "", 0)
	if _, ok := f.num(); ok {
		t.Error("非数字应当解析失败")
	}
	f = rawField(t, nil, "", 0)
	if f.text() != "" {
		t.Errorf("null 的 text = %q", f.text())
	}
	f = rawField(t, true, "", 0)
	if f.text() != "true" {
		t.Errorf("bool 的 text = %q", f.text())
	}
}

// ---------- 字段闸门 ----------

func TestGateExtractHappyPath(t *testing.T) {
	a, _ := newTestApp(t)
	ex := &extractResult{
		Fields: map[string]extractField{
			"name":       rawField(t, " 张三 ", "张三 男 28岁", 0.99),
			"gender":     rawField(t, "男", "张三 男", 0.98),
			"age":        rawField(t, "28", "28岁", 0.95),
			"subject":    rawField(t, "数学", "数学教师", 0.9),
			"has_cert":   rawField(t, 1, "持有高级中学教师资格证", 0.9),
			"phone":      rawField(t, "13800138000", "电话：13800138000", 0.95),
			"education":  rawField(t, "大学本科", "学历：大学本科", 0.9),
			"university": rawField(t, "北京大学", "毕业于北京大学", 0.9),
			"major":      rawField(t, "数学与应用数学", "专业：数学与应用数学", 0.9),
			"remark":     rawField(t, "5 年教龄", "5 年教龄", 0.8),
		},
		Summary: "中学数学教师",
	}
	raw := "张三 男 28岁 数学教师 持有高级中学教师资格证 电话：13800138000 学历：大学本科 毕业于北京大学 专业：数学与应用数学 5 年教龄"
	p := a.gateExtract(ex, raw)

	if p.Fields.Name != "张三" {
		t.Errorf("name = %q", p.Fields.Name)
	}
	if p.Fields.Gender != "male" {
		t.Errorf("gender = %q", p.Fields.Gender)
	}
	if p.Fields.Age != 28 {
		t.Errorf("age = %d", p.Fields.Age)
	}
	if p.Fields.Subject != "数学" {
		t.Errorf("subject = %q", p.Fields.Subject)
	}
	if p.Fields.HasCert != 1 {
		t.Errorf("has_cert = %d", p.Fields.HasCert)
	}
	if p.Fields.Phone != "13800138000" {
		t.Errorf("phone = %q", p.Fields.Phone)
	}
	if p.Fields.Education != "本科" {
		t.Errorf("education = %q，期望别名归一为「本科」", p.Fields.Education)
	}
	if len(p.Warnings) != 0 {
		t.Errorf("不应有告警，实际: %+v", p.Warnings)
	}
	if p.Meta["phone"].Level != "ok" || p.Meta["phone"].Evidence == "" {
		t.Errorf("phone meta = %+v", p.Meta["phone"])
	}
	if p.Summary != "中学数学教师" {
		t.Errorf("summary = %q", p.Summary)
	}
}

func TestGateExtractRejectsHallucination(t *testing.T) {
	a, _ := newTestApp(t)
	ex := &extractResult{Fields: map[string]extractField{
		"name":   rawField(t, "李四", "", 0.9),
		"gender": rawField(t, "女", "", 0.9),
		"phone":  rawField(t, "13900112233", "电话 13900112233", 0.95), // 正文里没有
	}}
	// 正文只出现另一个号码 → 幻觉号码必须被清空
	p := a.gateExtract(ex, "李四 女 教师 电话 13800138000")

	if p.Fields.Phone != "" {
		t.Errorf("正文中不存在的手机号应被清空，实际 %q", p.Fields.Phone)
	}
	if !hasWarning(p, "phone", "error") {
		t.Errorf("应给出 phone 的 error 告警，实际 %+v", p.Warnings)
	}
	if p.Fields.Gender != "female" {
		t.Errorf("gender = %q", p.Fields.Gender)
	}
}

func TestGateExtractDictionaryWhitelist(t *testing.T) {
	a, _ := newTestApp(t)
	ex := &extractResult{Fields: map[string]extractField{
		"name":       rawField(t, "王五", "", 0.9),
		"gender":     rawField(t, "男", "", 0.9),
		"subject":    rawField(t, "体育保健与健康", "", 0.9),
		"education":  rawField(t, "博士后", "", 0.9),
		"university": rawField(t, "某某大学", "", 0.9),
	}}
	p := a.gateExtract(ex, "王五 男")

	if p.Fields.Subject != "" {
		t.Errorf("字典外的学科必须留空，实际 %q", p.Fields.Subject)
	}
	if p.Fields.Education != "" {
		t.Errorf("字典外的学历必须留空，实际 %q", p.Fields.Education)
	}
	if !hasWarning(p, "subject", "warn") {
		t.Errorf("应给出 subject 的 warn 告警，实际 %+v", p.Warnings)
	}
	if !hasWarning(p, "education", "warn") {
		t.Errorf("应给出 education 的 warn 告警，实际 %+v", p.Warnings)
	}
	// 自由文本字段不受字典限制
	if p.Fields.University != "某某大学" {
		t.Errorf("university = %q", p.Fields.University)
	}
}

func TestGateExtractFormatGates(t *testing.T) {
	a, _ := newTestApp(t)
	ex := &extractResult{Fields: map[string]extractField{
		"name":     rawField(t, "赵六", "", 0.9),
		"gender":   rawField(t, "未知", "", 0.5),
		"age":      rawField(t, "150", "", 0.5),
		"phone":    rawField(t, "12345", "", 0.5),
		"has_cert": rawField(t, "大概有", "", 0.5),
	}}
	p := a.gateExtract(ex, "赵六 未知 150岁 12345 大概有")

	if p.Fields.Gender != "" {
		t.Errorf("无法识别的性别应留空，实际 %q", p.Fields.Gender)
	}
	if p.Fields.Age != 0 {
		t.Errorf("越界年龄应清空，实际 %d", p.Fields.Age)
	}
	if p.Fields.Phone != "" {
		t.Errorf("非法手机号应清空，实际 %q", p.Fields.Phone)
	}
	if p.Fields.HasCert != 0 {
		t.Errorf("无法识别的资格证应为 0，实际 %d", p.Fields.HasCert)
	}
	for _, f := range []string{"gender", "age", "phone", "has_cert"} {
		if !hasAnyWarning(p, f) {
			t.Errorf("字段 %s 被清空但没有告警", f)
		}
	}
}

func TestGateExtractMissingFields(t *testing.T) {
	a, _ := newTestApp(t)
	p := a.gateExtract(&extractResult{}, "")
	if p.Fields.Name != "" {
		t.Errorf("空抽取结果的 name 应为空")
	}
	if !hasWarning(p, "name", "error") {
		t.Errorf("缺少姓名应给出 error 告警，实际 %+v", p.Warnings)
	}
	if p.Meta == nil {
		t.Fatal("Meta 不应为 nil")
	}
}

// 模型按契约把「不在字典 / 无法判断」的取值清空，但 evidence 仍在：
// 这种「识别到了却没写进去」的数据必须告警，不能静默丢掉。
func TestGateExtractEvidenceWithoutValue(t *testing.T) {
	a, _ := newTestApp(t)
	ex := &extractResult{Fields: map[string]extractField{
		"name":       rawField(t, "王五", "王五", 0.9),
		"gender":     rawField(t, "男", "男", 0.9),
		"subject":    rawField(t, "", "求职意向：篮球教练", 0.9),
		"education":  rawField(t, "", "博士研究生", 0.9),
		"phone":      rawField(t, "", "电话 138-0013", 0.9),
		"university": rawField(t, "", "毕业于北京大学", 0.9),
	}}
	p := a.gateExtract(ex, "王五 男 求职意向：篮球教练 博士研究生 电话 138-0013 毕业于北京大学")

	for _, tc := range []struct{ field, level string }{
		{"subject", "warn"},
		{"education", "warn"},
		{"phone", "error"},
		{"university", "warn"},
	} {
		if !hasWarning(p, tc.field, tc.level) {
			t.Errorf("%s 应给出 %s 告警，实际 %+v", tc.field, tc.level, p.Warnings)
		}
	}
	if p.Fields.Subject != "" || p.Fields.Education != "" || p.Fields.Phone != "" {
		t.Errorf("字典外/非法取值必须保持为空: %+v", p.Fields)
	}
	if p.Meta["university"].Level != "warn" {
		t.Errorf("university meta = %+v", p.Meta["university"])
	}
}

// 契约里「年龄未知给 0」：0 本身不是错误，不能误报越界。
func TestGateExtractUnknownAgeIsNotAnError(t *testing.T) {
	a, _ := newTestApp(t)
	ex := &extractResult{Fields: map[string]extractField{
		"name":   rawField(t, "李四", "李四", 0.9),
		"gender": rawField(t, "女", "女", 0.9),
		"age":    rawField(t, 0, "", 0),
	}}
	p := a.gateExtract(ex, "李四 女 语文教师")
	if p.Fields.Age != 0 {
		t.Errorf("age = %d，应保持 0", p.Fields.Age)
	}
	if hasAnyWarning(p, "age") {
		t.Errorf("年龄未知给 0 不应告警，实际 %+v", p.Warnings)
	}
	if len(p.Warnings) != 0 {
		t.Errorf("不应有任何告警，实际 %+v", p.Warnings)
	}
}

// 取值必须能在识别依据中找到（证据里根本没有该词 → 疑似幻觉，交人工核对）。
func TestGateExtractValueEvidenceMismatch(t *testing.T) {
	a, _ := newTestApp(t)
	ex := &extractResult{Fields: map[string]extractField{
		"name":      rawField(t, "王五", "王五", 0.9),
		"gender":    rawField(t, "男", "男", 0.9),
		"subject":   rawField(t, "数学", "篮球教练", 0.9),
		"education": rawField(t, "本科", "毕业于北京大学", 0.9),
	}}
	p := a.gateExtract(ex, "王五 男 篮球教练 毕业于北京大学")

	for _, f := range []string{"subject", "education"} {
		if !hasWarning(p, f, "warn") {
			t.Errorf("%s 取值与证据不一致时应告警，实际 %+v", f, p.Warnings)
		}
		if p.Meta[f].Level != "warn" {
			t.Errorf("%s meta = %+v，应为 warn", f, p.Meta[f])
		}
	}
	if p.Fields.Subject != "数学" || p.Fields.Education != "本科" {
		t.Errorf("不一致只提示不拦截，取值仍应写入: %+v", p.Fields)
	}
}

func hasWarning(p draftPayload, field, level string) bool {
	for _, w := range p.Warnings {
		if w.Field == field && w.Level == level {
			return true
		}
	}
	return false
}

func hasAnyWarning(p draftPayload, field string) bool {
	for _, w := range p.Warnings {
		if w.Field == field {
			return true
		}
	}
	return false
}

// ---------- 模型配置接口 ----------

func TestLLMConfigRoundTrip(t *testing.T) {
	a, srv := newTestApp(t)
	cookie := loginAs(t, srv)

	// 未配置时 configured=false
	resp := req(t, srv, http.MethodGet, "/api/settings/llm", nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	view := decodeLLMView(t, resp)
	if view.Configured || view.APIKeySet {
		t.Errorf("初始状态应为未配置: %+v", view)
	}

	// 写入
	resp = req(t, srv, http.MethodPut, "/api/settings/llm", map[string]any{
		"endpoint": "https://api.example.com/v1",
		"protocol": "openai-chat",
		"model":    "deepseek-chat",
		"api_key":  "sk-abcdefghijkl",
	}, cookie)
	requireStatus(t, resp, http.StatusOK)
	view = decodeLLMView(t, resp)
	if !view.Configured || !view.APIKeySet {
		t.Errorf("写入后应为已配置: %+v", view)
	}
	if view.APIKeyHint != "****ijkl" {
		t.Errorf("api_key_hint = %q，期望 ****ijkl", view.APIKeyHint)
	}

	// 回读不应出现明文密钥
	resp = req(t, srv, http.MethodGet, "/api/settings/llm", nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	view = decodeLLMView(t, resp)
	b, _ := json.Marshal(view)
	if strings.Contains(string(b), "sk-abcdefghijkl") {
		t.Fatal("模型接口回显了明文 API Key")
	}

	// api_key 留空 = 不修改
	resp = req(t, srv, http.MethodPut, "/api/settings/llm", map[string]any{
		"endpoint": "https://api.example.com/v2",
		"protocol": "openai-chat",
		"model":    "deepseek-chat",
	}, cookie)
	requireStatus(t, resp, http.StatusOK)
	cfg, err := a.loadLLMConfig()
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	if cfg.APIKey != "sk-abcdefghijkl" {
		t.Errorf("空 api_key 应保留原值，实际 %q", cfg.APIKey)
	}
	if cfg.Endpoint != "https://api.example.com/v2" {
		t.Errorf("endpoint = %q", cfg.Endpoint)
	}
}

func TestLLMConfigValidation(t *testing.T) {
	_, srv := newTestApp(t)
	cookie := loginAs(t, srv)

	bad := []map[string]any{
		{"endpoint": "", "model": "m"},
		{"endpoint": "ftp://x/v1", "model": "m"},
		{"endpoint": "https://x/v1?key=secret", "model": "m"},
		{"endpoint": "https://x/v1", "model": ""},
		{"endpoint": "https://x/v1", "model": "m", "protocol": "grpc"},
	}
	for i, body := range bad {
		resp := req(t, srv, http.MethodPut, "/api/settings/llm", body, cookie)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("用例 %d 状态码 = %d，期望 400（body=%v）", i, resp.StatusCode, body)
		}
	}
}

func TestLLMSettingsNotPublic(t *testing.T) {
	a, srv := newTestApp(t)
	if err := setSetting(a.db, settingLLMEndpoint, "https://secret.example/v1"); err != nil {
		t.Fatalf("写设置失败: %v", err)
	}
	resp := req(t, srv, http.MethodGet, "/api/settings", nil, nil) // 未登录
	requireStatus(t, resp, http.StatusOK)
	var payload struct {
		Data map[string]string `json:"data"`
	}
	decodeBody(t, resp, &payload)
	for k := range payload.Data {
		if strings.HasPrefix(k, "llm_") {
			t.Fatalf("公开设置接口泄露了模型配置键: %s", k)
		}
	}
}

// ---------- 解析与入库 ----------

func TestResumeParseRejectsUnauthenticated(t *testing.T) {
	_, srv := newTestApp(t)
	resp := uploadResume(t, srv, nil, "resume.docx", buildDocx(t, sampleDocxXML))
	requireStatus(t, resp, http.StatusUnauthorized)
}

func TestResumeParseNotConfigured(t *testing.T) {
	_, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	resp := uploadResume(t, srv, cookie, "resume.docx", buildDocx(t, sampleDocxXML))
	requireStatus(t, resp, http.StatusServiceUnavailable)
	if msg := errorOf(t, resp); !strings.Contains(msg, "模型尚未配置") {
		t.Errorf("错误信息 = %q", msg)
	}
}

func TestResumeParseRejectsBadFiles(t *testing.T) {
	_, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	for name, data := range map[string][]byte{
		"photo.jpg": {0xFF, 0xD8, 0xFF, 0xE0, 0, 0},
		"old.doc":   {0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1},
		"empty.txt": []byte("hello"),
	} {
		resp := uploadResume(t, srv, cookie, name, data)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s 状态码 = %d，期望 400", name, resp.StatusCode)
		}
	}
}

func TestResumeParseAndCommit(t *testing.T) {
	a, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	configureLLM(t, a)

	a.extractHook = stubExtract(&extractResult{
		Fields: map[string]extractField{
			"name":       rawField(t, "张三", "张三 男 28岁", 0.99),
			"gender":     rawField(t, "男", "张三 男", 0.98),
			"age":        rawField(t, "28", "28岁", 0.95),
			"subject":    rawField(t, "数学", "数学教师", 0.9),
			"has_cert":   rawField(t, 1, "持有教师资格证", 0.9),
			"phone":      rawField(t, "13800138000", "电话：13800138000", 0.95),
			"education":  rawField(t, "大学本科", "学历：大学本科", 0.9),
			"university": rawField(t, "北京大学", "毕业于北京大学", 0.9),
			"major":      rawField(t, "数学与应用数学", "专业：数学与应用数学", 0.9),
		},
		Summary: "中学数学教师",
	}, nil)

	resp := uploadResume(t, srv, cookie, "张三-简历.docx", buildDocx(t, sampleDocxXML))
	requireStatus(t, resp, http.StatusCreated)
	id := draftID(t, resp)

	// 草稿详情
	resp = req(t, srv, http.MethodGet, fmt.Sprintf("/api/resume/drafts/%d", id), nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	var detail struct {
		Data struct {
			FileName string `json:"file_name"`
			RawText  string `json:"raw_text"`
			Payload  struct {
				Fields   teacherBody          `json:"fields"`
				Warnings []resumeWarning      `json:"warnings"`
				Meta     map[string]fieldMeta `json:"meta"`
			} `json:"payload"`
		} `json:"data"`
	}
	decodeBody(t, resp, &detail)
	if detail.Data.FileName != "张三-简历.docx" {
		t.Errorf("file_name = %q", detail.Data.FileName)
	}
	if detail.Data.Payload.Fields.Name != "张三" || detail.Data.Payload.Fields.Subject != "数学" {
		t.Errorf("预填字段 = %+v", detail.Data.Payload.Fields)
	}
	if detail.Data.Payload.Fields.Education != "本科" {
		t.Errorf("education 应归一到字典值，实际 %q", detail.Data.Payload.Fields.Education)
	}
	if detail.Data.RawText == "" {
		t.Error("详情应包含 raw_text")
	}
	if detail.Data.Payload.Meta["name"].Evidence == "" {
		t.Error("meta 应包含 name 的证据")
	}

	// 人工修正后入库（把学历改成硕士）
	detail.Data.Payload.Fields.Education = "硕士"
	resp = req(t, srv, http.MethodPost, fmt.Sprintf("/api/resume/drafts/%d/commit", id),
		map[string]any{"fields": detail.Data.Payload.Fields}, cookie)
	requireStatus(t, resp, http.StatusCreated)
	var created struct {
		Data Teacher `json:"data"`
	}
	decodeBody(t, resp, &created)
	if created.Data.Name != "张三" || created.Data.Education != "硕士" {
		t.Errorf("入库结果 = %+v", created.Data)
	}

	// 草稿已删除
	resp = req(t, srv, http.MethodGet, fmt.Sprintf("/api/resume/drafts/%d", id), nil, cookie)
	requireStatus(t, resp, http.StatusNotFound)

	// 教师确实落库
	items, total, err := listTeachers(a.db, teacherQuery{page: 1, size: 20, hasCert: -1, order: "desc"})
	if err != nil {
		t.Fatalf("查询教师失败: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Name != "张三" {
		t.Fatalf("教师表 = %d 条 %+v", total, items)
	}
}

func TestResumeCommitValidation(t *testing.T) {
	a, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	configureLLM(t, a)
	a.extractHook = stubExtract(&extractResult{
		Fields: map[string]extractField{"name": rawField(t, "李四", "", 0.9)},
	}, nil)

	resp := uploadResume(t, srv, cookie, "r.docx", buildDocx(t, sampleDocxXML))
	requireStatus(t, resp, http.StatusCreated)
	id := draftID(t, resp)

	// 缺性别 → 校验失败
	resp = req(t, srv, http.MethodPost, fmt.Sprintf("/api/resume/drafts/%d/commit", id),
		map[string]any{"fields": map[string]any{"name": "李四"}}, cookie)
	requireStatus(t, resp, http.StatusBadRequest)
	if msg := errorOf(t, resp); !strings.Contains(msg, "性别") {
		t.Errorf("错误信息 = %q", msg)
	}
	// 字典外学科 → 校验失败
	resp = req(t, srv, http.MethodPost, fmt.Sprintf("/api/resume/drafts/%d/commit", id),
		map[string]any{"fields": map[string]any{"name": "李四", "gender": "male", "subject": "玄学"}}, cookie)
	requireStatus(t, resp, http.StatusBadRequest)

	// 校验失败不删草稿
	resp = req(t, srv, http.MethodGet, fmt.Sprintf("/api/resume/drafts/%d", id), nil, cookie)
	requireStatus(t, resp, http.StatusOK)
}

func TestResumeDraftListAndDelete(t *testing.T) {
	a, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	configureLLM(t, a)
	a.extractHook = stubExtract(&extractResult{
		Fields: map[string]extractField{"name": rawField(t, "王五", "", 0.9)},
	}, nil)

	var ids []int64
	for i := 0; i < 3; i++ {
		resp := uploadResume(t, srv, cookie, fmt.Sprintf("r%d.docx", i), buildDocx(t, sampleDocxXML))
		requireStatus(t, resp, http.StatusCreated)
		ids = append(ids, draftID(t, resp))
	}

	resp := req(t, srv, http.MethodGet, "/api/resume/drafts?page=1&size=2", nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	var list struct {
		Items []resumeView `json:"items"`
		Total int          `json:"total"`
	}
	decodeBody(t, resp, &list)
	if list.Total != 3 || len(list.Items) != 2 {
		t.Fatalf("列表 = %d/%d，期望 3 条取 2", len(list.Items), list.Total)
	}
	if list.Items[0].ID != ids[2] {
		t.Errorf("应按 id 倒序，首条 = %d，期望 %d", list.Items[0].ID, ids[2])
	}

	// 删除
	resp = req(t, srv, http.MethodDelete, fmt.Sprintf("/api/resume/drafts/%d", ids[0]), nil, cookie)
	requireStatus(t, resp, http.StatusOK)
	resp = req(t, srv, http.MethodGet, fmt.Sprintf("/api/resume/drafts/%d", ids[0]), nil, cookie)
	requireStatus(t, resp, http.StatusNotFound)

	// 不存在的 id
	resp = req(t, srv, http.MethodDelete, "/api/resume/drafts/99999", nil, cookie)
	requireStatus(t, resp, http.StatusNotFound)
}

func TestResumeParseExtractionError(t *testing.T) {
	a, srv := newTestApp(t)
	cookie := loginAs(t, srv)
	configureLLM(t, a)
	a.extractHook = stubExtract(nil, fmt.Errorf("boom"))

	resp := uploadResume(t, srv, cookie, "r.docx", buildDocx(t, sampleDocxXML))
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("状态码 = %d，期望 502", resp.StatusCode)
	}
}

// ---------- pdftoppm 检测 ----------

func TestPDFToppmDetection(t *testing.T) {
	// 两种结果都合法：装了就返回路径，没装就返回带说明的错误
	if p, err := pdftoppmPath(); err == nil {
		if p == "" {
			t.Error("检测到 pdftoppm 但路径为空")
		}
	} else if !strings.Contains(err.Error(), "poppler-utils") {
		t.Errorf("错误信息缺少安装指引: %v", err)
	}
}
