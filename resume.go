package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"archive/zip"

	"github.com/cn-maul/rosetta"
	pdf "github.com/ledongthuc/pdf"
)

// ---------- 上传约束 ----------

const (
	// resumeMaxBytes 上传文件大小上限。
	resumeMaxBytes = 10 << 20 // 10MB
	// resumeMaxPages 扫描件最多渲染/识别的页数。
	resumeMaxPages = 10
	// resumeMinTextRune 判定 PDF 是否具有可用文本层的最小有效字符数。
	resumeMinTextRune = 40
	// resumeMaxPromptRune 送给模型的正文字符上限（保守控制上下文）。
	resumeMaxPromptRune = 12000
	// resumeMaxPagePNG 单页渲染图像的字节上限。
	resumeMaxPagePNG = 8 << 20
	// resumeMaxOutputTokens 模型输出上限。
	resumeMaxOutputTokens = 4096
)

// ---------- 抽取结果契约 ----------

// extractField 模型返回的单字段：值 + 证据 + 置信度。
type extractField struct {
	Value      json.RawMessage `json:"value"`
	Evidence   string          `json:"evidence"`
	Confidence float64         `json:"confidence"`
}

// text 把任意 JSON 值（字符串/数字/布尔/null）归一为字符串。
func (f extractField) text() string {
	raw := bytes.TrimSpace(f.Value)
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	if !json.Valid(raw) {
		return ""
	}
	return string(raw) // 数字或布尔字面量
}

// num 解析整数（容忍「28岁」「28.0」这类写法）。
func (f extractField) num() (int, bool) {
	s := strings.TrimSpace(f.text())
	if s == "" {
		return 0, false
	}
	s = strings.TrimSuffix(s, "岁")
	s = strings.TrimSuffix(s, "周岁")
	if i := strings.IndexAny(s, ".。"); i > 0 {
		s = s[:i]
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// extractResult 模型的完整输出。
type extractResult struct {
	Fields  map[string]extractField `json:"fields"`
	Summary string                  `json:"summary"`
}

// fieldMeta 前端高亮所需的字段元信息。
type fieldMeta struct {
	Evidence   string  `json:"evidence"`
	Confidence float64 `json:"confidence"`
	Level      string  `json:"level"` // ok | warn | error
}

// resumeWarning 闸门产生的告警。
type resumeWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Level   string `json:"level"` // warn | error
}

// draftPayload 草稿的 payload 结构（存 JSON 文本，读时原样回传）。
type draftPayload struct {
	Fields   teacherBody          `json:"fields"`
	Meta     map[string]fieldMeta `json:"meta"`
	Warnings []resumeWarning      `json:"warnings"`
	Summary  string               `json:"summary"`
}

// hasWarning 该字段是否已被闸门标记过（用于避免重复告警）。
func (p draftPayload) hasWarning(field string) bool {
	for _, w := range p.Warnings {
		if w.Field == field {
			return true
		}
	}
	return false
}

// resumeFieldLabel 闸门兜底提示里用到的字段中文名。
var resumeFieldLabel = map[string]string{
	"name":       "姓名",
	"gender":     "性别",
	"age":        "年龄",
	"subject":    "学科",
	"has_cert":   "教师资格证",
	"phone":      "联系电话",
	"education":  "学历",
	"university": "毕业院校",
	"major":      "专业",
	"remark":     "备注",
}

// ---------- 上传与嗅探 ----------

// readResumeUpload 从 multipart 请求中读取名为 file 的部分（仅内存，不落盘）。
// 失败时已写好错误响应并返回 ok=false。
func readResumeUpload(w http.ResponseWriter, r *http.Request) (name, kind string, data []byte, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, resumeMaxBytes+(1<<20)) // 为 multipart 边界留余量
	mr, err := r.MultipartReader()
	if err != nil {
		jsonError(w, http.StatusBadRequest, "请求须为 multipart/form-data 上传")
		return "", "", nil, false
	}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			jsonError(w, http.StatusBadRequest, "读取上传内容失败")
			return "", "", nil, false
		}
		if part.FormName() != "file" {
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(part, resumeMaxBytes+1))
		_ = part.Close()
		if err != nil {
			jsonError(w, http.StatusBadRequest, "读取上传文件失败")
			return "", "", nil, false
		}
		if len(raw) == 0 {
			jsonError(w, http.StatusBadRequest, "上传文件为空")
			return "", "", nil, false
		}
		if len(raw) > resumeMaxBytes {
			jsonError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("文件超过 %dMB 上限", resumeMaxBytes>>20))
			return "", "", nil, false
		}
		name = sanitizeUploadName(part.FileName())
		kind, ok = sniffResumeKind(raw)
		if !ok {
			jsonError(w, http.StatusBadRequest, "仅支持 .docx 与 .pdf（按文件内容识别，扩展名不作数）")
			return "", "", nil, false
		}
		return name, kind, raw, true
	}
	jsonError(w, http.StatusBadRequest, "请选择要识别的简历文件")
	return "", "", nil, false
}

// sanitizeUploadName 归一化上传文件名：去掉路径、控制字符，限制长度。
func sanitizeUploadName(fn string) string {
	fn = filepath.Base(strings.ReplaceAll(fn, "\\", "/"))
	fn = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		if r == '/' || r == '\\' || r == ':' {
			return -1
		}
		return r
	}, fn)
	fn = strings.TrimSpace(fn)
	if fn == "" || fn == "." || fn == ".." {
		return "resume"
	}
	if len([]rune(fn)) > 120 {
		r := []rune(fn)
		fn = string(r[:120])
	}
	return fn
}

// sniffResumeKind 按魔数判定文件类型，扩展名仅作展示。
func sniffResumeKind(data []byte) (string, bool) {
	switch {
	case len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04:
		return "docx", true
	case len(data) >= 5 && string(data[:5]) == "%PDF-":
		return "pdf", true
	}
	return "", false
}

// ---------- docx 解析 ----------

// docxText 抽取 docx 正文：段落逐行，表格渲染为「值1 | 值2」行。
func docxText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("文件不是有效的 docx（zip 结构无法读取）: %w", err)
	}
	var doc *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return "", errors.New("文件不是有效的 docx（缺少 word/document.xml）")
	}
	rc, err := doc.Open()
	if err != nil {
		return "", fmt.Errorf("读取 docx 正文失败: %w", err)
	}
	defer rc.Close()
	return parseDocumentXML(rc)
}

// parseDocumentXML 用状态机把 word/document.xml 还原为纯文本：
// 段落 → 行；表格单元格 → 「 | 」分隔；表格行 → 行。
func parseDocumentXML(r io.Reader) (string, error) {
	dec := xml.NewDecoder(r)
	var out, para, cell, row strings.Builder
	inCell := false

	flushPara := func() {
		s := strings.TrimSpace(para.String())
		para.Reset()
		if s == "" {
			return
		}
		if inCell {
			if cell.Len() > 0 {
				cell.WriteByte(' ')
			}
			cell.WriteString(s)
			return
		}
		out.WriteString(s)
		out.WriteByte('\n')
	}
	flushCell := func() {
		s := strings.TrimSpace(cell.String())
		cell.Reset()
		if s == "" {
			return
		}
		if row.Len() > 0 {
			row.WriteString(" | ")
		}
		row.WriteString(s)
	}
	flushRow := func() {
		s := strings.TrimSpace(row.String())
		row.Reset()
		inCell = false
		if s == "" {
			return
		}
		out.WriteString(s)
		out.WriteByte('\n')
	}

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("解析 docx 失败: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tab":
				para.WriteByte(' ')
			case "br", "cr":
				para.WriteByte('\n')
			case "tr":
				row.Reset()
			case "tc":
				inCell = true
				cell.Reset()
			case "t":
				var s string
				if err := dec.DecodeElement(&s, &t); err != nil {
					return "", fmt.Errorf("解析 docx 文本失败: %w", err)
				}
				para.WriteString(s)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				flushPara()
			case "tc":
				flushCell()
			case "tr":
				flushRow()
			}
		}
	}
	flushPara()
	return strings.TrimSpace(collapseBlank(out.String())), nil
}

// collapseBlank 把连续空行压成一个，去掉行尾空白。
func collapseBlank(s string) string {
	lines := strings.Split(s, "\n")
	out := lines[:0]
	prevBlank := false
	for _, ln := range lines {
		ln = strings.TrimRight(ln, " \t\r")
		blank := strings.TrimSpace(ln) == ""
		if blank && prevBlank {
			continue
		}
		out = append(out, ln)
		prevBlank = blank
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// ---------- pdf 解析 ----------

// pdfText 抽取 PDF 文本层；同时返回总页数。空返回表示是扫描件（无文本层）。
func pdfText(data []byte, maxPages int) (text string, pages int, err error) {
	defer func() {
		if e := recover(); e != nil {
			text, pages, err = "", 0, fmt.Errorf("解析 PDF 失败: %v", e)
		}
	}()
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", 0, fmt.Errorf("打开 PDF 失败（文件可能已损坏）: %w", err)
	}
	total := r.NumPage()
	if total < 1 {
		return "", 0, errors.New("PDF 没有页面")
	}
	pages = total
	limit := total
	if limit > maxPages {
		limit = maxPages
	}
	var b strings.Builder
	for i := 1; i <= limit; i++ {
		p := r.Page(i)
		s, e := p.GetPlainText(nil)
		if e != nil {
			continue // 个别页损坏时跳过，由后续「无文本层」判定兜底
		}
		b.WriteString(s)
		b.WriteByte('\n')
	}
	return b.String(), pages, nil
}

// meaningfulRune 统计去空白后的有效字符数。
func meaningfulRune(s string) int {
	var n int
	for _, r := range s {
		if !strings.ContainsRune(" \t\r\n\f　", r) {
			n++
		}
	}
	return n
}

// pdftoppmPath 返回系统中的 pdftoppm 路径（缺失返回 error）。
func pdftoppmPath() (string, error) {
	p, err := exec.LookPath("pdftoppm")
	if err != nil {
		return "", errors.New("服务器缺少 pdftoppm（poppler-utils），无法识别扫描件 PDF，请安装后重试")
	}
	return p, nil
}

// renderPDFPages 用 pdftoppm 把 PDF 前 maxPages 页渲染为 PNG，返回 base64。
// 输入与输出都放在请求级临时目录，函数返回前整体删除。
func renderPDFPages(ctx context.Context, data []byte, maxPages int) (b64 []string, err error) {
	bin, err := pdftoppmPath()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "hirezo-pdf-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(src, data, 0o600); err != nil {
		return nil, fmt.Errorf("写入临时文件失败: %w", err)
	}
	prefix := filepath.Join(dir, "page")
	// pdftoppm 的参数顺序是 PDF-file 在前、输出 root 在后，反了会把 root 当输入去打开
	cmd := exec.CommandContext(ctx, bin, "-png", "-r", "150",
		"-f", "1", "-l", strconv.Itoa(maxPages), src, prefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("渲染 PDF 页面失败: %v %s", err, msg)
	}

	matches, _ := filepath.Glob(prefix + "-*.png")
	if len(matches) == 0 {
		return nil, errors.New("渲染 PDF 页面失败：未生成任何图像")
	}
	sort.Slice(matches, func(i, j int) bool {
		return pageSortKey(matches[i]) < pageSortKey(matches[j])
	})
	if len(matches) > maxPages {
		matches = matches[:maxPages]
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		raw, err := os.ReadFile(m)
		if err != nil {
			return nil, fmt.Errorf("读取渲染结果失败: %w", err)
		}
		if len(raw) > resumeMaxPagePNG {
			return nil, fmt.Errorf("第 %s 页渲染图像过大（>%dMB），请拆分文件后重试", pageLabel(m), resumeMaxPagePNG>>20)
		}
		out = append(out, base64.StdEncoding.EncodeToString(raw))
	}
	return out, nil
}

func pageSortKey(path string) int {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	i := strings.LastIndexByte(base, '-')
	if i < 0 {
		return 0
	}
	n, _ := strconv.Atoi(base[i+1:])
	return n
}

func pageLabel(path string) string {
	k := pageSortKey(path)
	if k == 0 {
		return "?"
	}
	return strconv.Itoa(k)
}

// ---------- prompt ----------

// resumeExtractInput 传给抽取函数的已解析内容。
type resumeExtractInput struct {
	FileName  string
	Kind      string   // docx | pdf
	Text      string   // 归一化正文（docx / 文本层 PDF）
	Pages     []string // base64 PNG（扫描件）
	PageCount int      // PDF 总页数（docx 为 0）
}

// resumeSystemPrompt 构造系统提示：字段契约 + 字典白名单 + 反幻觉要求。
func (a *app) resumeSystemPrompt() string {
	subjects := strings.Join(a.dictNames(dictKindSubject), "、")
	educations := strings.Join(a.dictNames(dictKindEducation), "、")
	return strings.TrimSpace(`
你是教师简历信息抽取引擎。根据用户提供的简历内容抽取字段，只输出一个 JSON 对象。
禁止输出 Markdown 代码块、解释文字或 JSON 之外的任何内容。

字段（全部必填，简历中没有时给空值）：
- name      姓名，字符串
- gender    只能是 "male"（男）或 "female"（女），无法判断给 ""
- age       18-100 的整数，未知给 0
- subject   任教学科，只能从下方列表选一个，否则给 ""
- has_cert  是否持有教师资格证，0 或 1，未知给 0
- phone     中国大陆 11 位手机号，未知给 ""
- education 学历，只能从下方列表选一个，否则给 ""
- university 毕业院校，未知给 ""
- major     专业，未知给 ""
- remark    职称 / 教龄 / 任教经历等补充信息，未知给 ""

可用学科列表：` + subjects + `
可用学历列表：` + educations + `

输出格式（严格遵守，字段一个都不能少）：
{"fields":{"name":{"value":"","evidence":"","confidence":0},"gender":{"value":"","evidence":"","confidence":0},"age":{"value":"","evidence":"","confidence":0},"subject":{"value":"","evidence":"","confidence":0},"has_cert":{"value":"","evidence":"","confidence":0},"phone":{"value":"","evidence":"","confidence":0},"education":{"value":"","evidence":"","confidence":0},"university":{"value":"","evidence":"","confidence":0},"major":{"value":"","evidence":"","confidence":0},"remark":{"value":"","evidence":"","confidence":0}},"summary":""}

规则：
1. 每个字段格式为 {"value":..., "evidence":"简历中对应的原句片段", "confidence":0~1 的数字}。
2. evidence 必须逐字摘自简历内容，禁止编造、改写或拼接；找不到时 evidence 留空、confidence 给 0。
3. 简历中没有的信息一律给空值（age、has_cert 给 0），不要根据常识推测。
4. summary 用不超过 20 字概括候选人。`)
}

// buildResumeMessage 组装用户消息：有页图走多模态，否则走纯文本。
func buildResumeMessage(in resumeExtractInput) rosetta.Message {
	if len(in.Pages) == 0 {
		return rosetta.User("简历内容：\n" + truncateRunes(in.Text, resumeMaxPromptRune))
	}
	blocks := []rosetta.Block{{
		Type: rosetta.BlockText,
		Text: "这是简历的页面图片（共 " + strconv.Itoa(len(in.Pages)) + " 页），请抽取字段。",
	}}
	for i, p := range in.Pages {
		blocks = append(blocks, rosetta.Block{
			Type:     rosetta.BlockImage,
			ImageURL: "data:image/png;base64," + p,
			Text:     "第 " + strconv.Itoa(i+1) + " 页",
		})
	}
	return rosetta.Message{Role: rosetta.RoleUser, Blocks: blocks}
}

// truncateRunes 按字符截断（超长正文控制上下文）。
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "\n…（正文已截断）"
}

// ---------- 抽取（模型调用） ----------

// extractHook 仅供测试注入的抽取桩；nil 时走真实模型调用。
type extractHookFunc func(ctx context.Context, in resumeExtractInput) (*extractResult, error)

// runExtract 执行字段抽取：真实实现走 rosetta，测试可替换 app.extractHook。
func (a *app) runExtract(ctx context.Context, in resumeExtractInput) (*extractResult, error) {
	if a.extractHook != nil {
		return a.extractHook(ctx, in)
	}
	return a.llmExtract(ctx, in)
}

// llmExtract 调用大模型抽取字段并解析为结构化结果。
func (a *app) llmExtract(ctx context.Context, in resumeExtractInput) (*extractResult, error) {
	cfg, err := a.loadLLMConfig()
	if err != nil {
		return nil, fmt.Errorf("读取模型配置失败: %w", err)
	}
	if !cfg.ready() {
		return nil, errModelNotConfigured
	}
	client, err := cfg.client()
	if err != nil {
		return nil, fmt.Errorf("初始化模型客户端失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, llmTimeout)
	defer cancel()

	resp, err := client.Chat(ctx, &rosetta.ChatRequest{
		Model:           strings.TrimSpace(cfg.Model),
		System:          a.resumeSystemPrompt(),
		Messages:        []rosetta.Message{buildResumeMessage(in)},
		MaxOutputTokens: resumeMaxOutputTokens,
		Temperature:     rosetta.Float(0),
	})
	if err != nil {
		return nil, err
	}
	ex, err := parseExtractJSON(resp.Text())
	if err != nil {
		return nil, err
	}
	return ex, nil
}

// parseExtractJSON 容错解析模型输出：剥离代码围栏、取首个 { 到末个 }。
func parseExtractJSON(s string) (*extractResult, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("模型没有返回内容")
	}
	// 剥离 ```json ... ``` 围栏
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[i+3:]
		s = strings.TrimPrefix(s, "json")
		if j := strings.Index(s, "```"); j >= 0 {
			s = s[:j]
		}
	}
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end < start {
		return nil, fmt.Errorf("模型输出不是 JSON 对象: %s", truncateRunes(s, 200))
	}
	raw := []byte(s[start : end+1])
	if !json.Valid(raw) {
		return nil, fmt.Errorf("模型输出的 JSON 无法解析: %s", truncateRunes(s, 200))
	}
	ex := &extractResult{Fields: map[string]extractField{}}
	if err := json.Unmarshal(raw, ex); err != nil {
		return nil, fmt.Errorf("解析模型输出失败: %w", err)
	}
	if ex.Fields == nil {
		ex.Fields = map[string]extractField{}
	}
	return ex, nil
}

// ---------- 字段闸门 ----------

// educationAlias 学历的无歧义写法映射（有歧义的写法一律不猜）。
var educationAlias = map[string]string{
	"大学专科":  "大专",
	"大专学历":  "大专",
	"专科学历":  "大专",
	"大学本科":  "本科",
	"本科学历":  "本科",
	"硕士研究生": "硕士",
	"硕士学历":  "硕士",
	"博士研究生": "博士",
	"博士学历":  "博士",
	"中专学历":  "中专",
}

// normalizeDictValue 把模型返回的学科/学历归一到字典值。
// 返回 (匹配值, 无法匹配的提示)；匹配不上时匹配值为空串。
func (a *app) normalizeDictValue(kind, raw string) (string, string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ""
	}
	// 去掉空白与常见后缀
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(" \t\r\n　", r) {
			return -1
		}
		return r
	}, s)
	s = strings.TrimSuffix(s, "学科")
	s = strings.TrimSuffix(s, "科目")
	s = strings.TrimSuffix(s, "老师")
	s = strings.TrimSuffix(s, "教师")
	s = strings.TrimSuffix(s, "学历")

	if kind == dictKindSubject {
		// 常见简写
		if v, ok := subjectAlias[s]; ok {
			s = v
		}
	} else {
		if v, ok := educationAlias[s]; ok {
			s = v
		}
	}
	for _, d := range a.dictNames(kind) {
		if d == s {
			return s, ""
		}
	}
	return "", fmt.Sprintf("%s「%s」不在字典中，请下拉选择", dictKindLabel(kind), strings.TrimSpace(raw))
}

// subjectAlias 学科的无歧义简写映射。
var subjectAlias = map[string]string{
	"信息技术课": "信息技术",
	"计算机":   "信息技术",
	"计算机课":  "信息技术",
	"信息":    "信息技术",
}

// dictNames 读取字典名列表；出错返回空（只影响 prompt 完整度，闸门侧仍会兜底）。
func (a *app) dictNames(kind string) []string {
	ds, err := allDictionaries(a.db, kind)
	if err != nil {
		log.Printf("[简历] 读取 %s 字典失败: %v", kind, err)
		return nil
	}
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}

// gateExtract 对模型输出做本地校验：格式闸门 + 字典白名单 + 反幻觉检查。
// 清空的字段一定会附带 warning，绝不静默丢数据。
func (a *app) gateExtract(ex *extractResult, rawText string) draftPayload {
	p := draftPayload{
		Fields:   teacherBody{},
		Meta:     map[string]fieldMeta{},
		Warnings: []resumeWarning{},
	}
	if ex == nil {
		ex = &extractResult{Fields: map[string]extractField{}}
	}
	get := func(name string) extractField {
		if f, ok := ex.Fields[name]; ok {
			return f
		}
		return extractField{}
	}
	meta := func(name string, f extractField, level string) {
		p.Meta[name] = fieldMeta{
			Evidence:   strings.TrimSpace(f.Evidence),
			Confidence: f.Confidence,
			Level:      level,
		}
	}
	warn := func(field, msg, level string) {
		p.Warnings = append(p.Warnings, resumeWarning{Field: field, Message: msg, Level: level})
	}

	// 正文压平（去空白），用于反幻觉比对
	flat := flattenRunes(rawText)

	// ---- name ----
	f := get("name")
	name := strings.TrimSpace(f.text())
	switch {
	case name == "":
		meta("name", f, "error")
		warn("name", "未能识别姓名，请手动填写", "error")
	case len([]rune(name)) > 50:
		meta("name", f, "error")
		warn("name", "姓名过长（最多 50 个字符）", "error")
	default:
		p.Fields.Name = name
		meta("name", f, "ok")
	}

	// ---- gender ----
	f = get("gender")
	raw := strings.TrimSpace(f.text())
	switch raw {
	case "男", "男性", "male", "man", "m", "M":
		p.Fields.Gender = "male"
		meta("gender", f, "ok")
	case "女", "女性", "female", "woman", "f", "F":
		p.Fields.Gender = "female"
		meta("gender", f, "ok")
	default:
		meta("gender", f, "error")
		warn("gender", "性别无法识别，请选择男或女", "error")
	}

	// ---- age ----
	f = get("age")
	ageEv := strings.TrimSpace(f.Evidence)
	switch n, ok := f.num(); {
	case ok && n >= 18 && n <= 100:
		p.Fields.Age = n
		meta("age", f, "ok")
	case ok && n == 0 && ageEv == "":
		// 契约里「未知给 0」：不算错误
		meta("age", f, "ok")
	case ok && n == 0:
		// 有证据却给 0 → 内容识别到了但没写进取值
		meta("age", f, "warn")
		warn("age", "识别到年龄但未能写入取值，请手动填写", "warn")
	case ok:
		meta("age", f, "error")
		warn("age", "年龄超出 18-100 范围，请重新填写", "error")
	case strings.TrimSpace(f.text()) != "":
		meta("age", f, "error")
		warn("age", "年龄格式无法识别，请重新填写", "error")
	default:
		meta("age", f, "ok")
	}

	// ---- subject ----
	f = get("subject")
	subjectEv := strings.TrimSpace(f.Evidence)
	if v, msg := a.normalizeDictValue(dictKindSubject, f.text()); v != "" {
		p.Fields.Subject = v
		if subjectEv != "" && !valueInEvidence(v, subjectEv) {
			meta("subject", f, "warn")
			warn("subject", "取值与识别依据不一致，请核对学科", "warn")
		} else {
			meta("subject", f, "ok")
		}
	} else if msg != "" {
		meta("subject", f, "warn")
		warn("subject", msg, "warn")
	} else if subjectEv != "" {
		// 模型按规则「不在列表给空值」清空了取值，但证据仍在 → 必须提示人工下拉选择
		meta("subject", f, "warn")
		warn("subject", dictKindLabel(dictKindSubject)+"不在字典中，请下拉选择", "warn")
	} else {
		meta("subject", f, "ok")
	}

	// ---- has_cert ----
	f = get("has_cert")
	certRaw := strings.TrimSpace(f.text())
	cert, certOK := parseCert(certRaw)
	switch {
	case certOK:
		p.Fields.HasCert = cert
		level := "ok"
		if cert == 0 && strings.TrimSpace(f.Evidence) != "" && looksPositive(f.Evidence) {
			level = "warn"
			warn("has_cert", "证据与取值不一致，请核对是否持有教师资格证", "warn")
		}
		meta("has_cert", f, level)
	case certRaw == "" && strings.TrimSpace(f.Evidence) == "":
		meta("has_cert", f, "ok") // 简历未提及，默认无
	default:
		meta("has_cert", f, "warn")
		warn("has_cert", "教师资格证信息无法识别，请手动选择", "warn")
	}

	// ---- phone ----
	f = get("phone")
	phone := normalizePhone(f.text())
	phoneEv := strings.TrimSpace(f.Evidence)
	switch {
	case phone == "" && phoneEv != "":
		// 取值为空但简历里有联系方式：写不进去就一定要提醒（不静默丢数据）
		if strings.ContainsAny(phoneEv, "0123456789") {
			meta("phone", f, "error")
			warn("phone", "手机号格式不正确，请重新填写", "error")
		} else {
			meta("phone", f, "warn")
			warn("phone", "未能识别手机号，请手动填写", "warn")
		}
	case phone == "":
		meta("phone", f, "ok")
	case !validPhone(phone):
		meta("phone", f, "error")
		warn("phone", "手机号格式不正确，请重新填写", "error")
	case flat != "" && !strings.Contains(flat, phone):
		// 反幻觉：正文里根本没出现该号码
		meta("phone", f, "error")
		warn("phone", "正文中未找到该手机号，请核对（疑似识别错误）", "error")
	default:
		p.Fields.Phone = phone
		meta("phone", f, "ok")
	}

	// ---- education ----
	f = get("education")
	eduEv := strings.TrimSpace(f.Evidence)
	if v, msg := a.normalizeDictValue(dictKindEducation, f.text()); v != "" {
		p.Fields.Education = v
		if eduEv != "" && !valueInEvidence(v, eduEv) {
			meta("education", f, "warn")
			warn("education", "取值与识别依据不一致，请核对学历", "warn")
		} else {
			meta("education", f, "ok")
		}
	} else if msg != "" {
		meta("education", f, "warn")
		warn("education", msg, "warn")
	} else if eduEv != "" {
		meta("education", f, "warn")
		warn("education", dictKindLabel(dictKindEducation)+"不在字典中，请下拉选择", "warn")
	} else {
		meta("education", f, "ok")
	}

	// ---- university / major ----
	f = get("university")
	if v := strings.TrimSpace(f.text()); v != "" {
		if len([]rune(v)) > 100 {
			p.Fields.University = string([]rune(v)[:100])
			meta("university", f, "warn")
			warn("university", "毕业院校过长，已截断至 100 字符", "warn")
		} else {
			p.Fields.University = v
			meta("university", f, "ok")
		}
	} else {
		meta("university", f, "ok")
	}

	f = get("major")
	if v := strings.TrimSpace(f.text()); v != "" {
		if len([]rune(v)) > 100 {
			p.Fields.Major = string([]rune(v)[:100])
			meta("major", f, "warn")
			warn("major", "专业过长，已截断至 100 字符", "warn")
		} else {
			p.Fields.Major = v
			meta("major", f, "ok")
		}
	} else {
		meta("major", f, "ok")
	}

	// ---- remark ----
	f = get("remark")
	if v := strings.TrimSpace(f.text()); v != "" {
		if len([]rune(v)) > 500 {
			p.Fields.Remark = string([]rune(v)[:500])
			meta("remark", f, "warn")
			warn("remark", "备注过长，已截断至 500 字符", "warn")
		} else {
			p.Fields.Remark = v
			meta("remark", f, "ok")
		}
	} else {
		meta("remark", f, "ok")
	}

	// ---- 兜底：模型给了证据却没给取值（院校、专业、备注、年龄等）----
	// 这类「识别到了但没写进去」的数据若不提醒就会被静默丢掉。
	for _, fieldName := range []string{
		"name", "gender", "age", "subject", "has_cert",
		"phone", "education", "university", "major", "remark",
	} {
		if p.hasWarning(fieldName) {
			continue
		}
		g := get(fieldName)
		if g.text() != "" || strings.TrimSpace(g.Evidence) == "" {
			continue
		}
		p.Meta[fieldName] = fieldMeta{
			Evidence:   strings.TrimSpace(g.Evidence),
			Confidence: g.Confidence,
			Level:      "warn",
		}
		warn(fieldName, fmt.Sprintf("识别到%s相关内容，但未能写入取值，请手动补全", resumeFieldLabel[fieldName]), "warn")
	}

	p.Summary = strings.TrimSpace(ex.Summary)
	return p
}

// parseCert 把模型返回的资格证取值归一为 0/1；无法判断返回 ok=false。
func parseCert(s string) (int, bool) {
	switch strings.TrimSpace(s) {
	case "1", "true", "有", "有证", "是", "具备", "✓", "√", "yes", "y":
		return 1, true
	case "0", "false", "无", "没有", "否", "不具备", "×", "x", "no", "n":
		return 0, true
	}
	return 0, false
}

// looksPositive 判断证据文本是否在描述「持有证书」（用于一致性提示）。
func looksPositive(ev string) bool {
	ev = strings.TrimSpace(ev)
	if ev == "" {
		return false
	}
	if certNegativeEvidence.MatchString(ev) {
		return false
	}
	return certPositiveEvidence.MatchString(ev)
}

// flattenRunes 去掉所有空白，用于「取值是否出现在证据里」的比对。
func flattenRunes(s string) string {
	return strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "", "　", "").Replace(s)
}

// valueInEvidence 判断取值是否出现在识别依据里（反幻觉：证据中找不到该词 → 高度可疑）。
func valueInEvidence(value, evidence string) bool {
	v := flattenRunes(value)
	return v == "" || strings.Contains(flattenRunes(evidence), v)
}

// normalizePhone 去掉分隔符并补全常见前缀写法。
func normalizePhone(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.NewReplacer("-", "", " ", "", "\t", "", "－", "", "（", "", "）", "", "(", "", ")", "", "+86", "", "86", "").Replace(s)
	if len(s) == 12 && strings.HasPrefix(s, "0") {
		// 座机或带 0 前缀：不是手机号，交给 validPhone 判否
		return s
	}
	return s
}

// ---------- 简历原文件存储 ----------

// resumeExt 按识别类型返回落盘扩展名。
func resumeExt(kind string) string {
	if kind == "pdf" {
		return ".pdf"
	}
	return ".docx"
}

// resumeFilePath 永久库中的物理路径（resumes/{教师ID}.{ext}）。
func (a *app) resumeFilePath(file string) string {
	if a.resumeDir == "" || file == "" {
		return ""
	}
	return filepath.Join(a.resumeDir, file)
}

// draftResumePath 草稿临时文件的物理路径（drafts/{草稿ID}.{ext}）。
func (a *app) draftResumePath(draftID int64, kind string) string {
	if a.resumeDraftDir == "" {
		return ""
	}
	return filepath.Join(a.resumeDraftDir, strconv.FormatInt(draftID, 10)+resumeExt(kind))
}

// saveDraftResumeFile 识别成功后保存原件到临时目录；失败只记日志（识别结果不受影响，
// 但 commit 时无法归档原件，入库将不带简历文件）。
func (a *app) saveDraftResumeFile(draftID int64, kind string, data []byte) {
	p := a.draftResumePath(draftID, kind)
	if p == "" {
		return
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		log.Printf("[简历] 保存草稿原件失败 draft=%d: %v", draftID, err)
	}
}

// commitResumeFile 把草稿临时原件移动到永久目录，返回入库文件名；无原件或移动失败返回空串。
func (a *app) commitResumeFile(draftID int64, kind string, teacherID int64) string {
	src := a.draftResumePath(draftID, kind)
	if src == "" || a.resumeDir == "" {
		return ""
	}
	if _, err := os.Stat(src); err != nil {
		return ""
	}
	file := strconv.FormatInt(teacherID, 10) + resumeExt(kind)
	dst := filepath.Join(a.resumeDir, file)
	if err := os.Rename(src, dst); err != nil {
		// 跨设备等情况退回复制
		data, rerr := os.ReadFile(src)
		if rerr != nil {
			log.Printf("[简历] 归档原件失败 draft=%d teacher=%d: %v / %v", draftID, teacherID, err, rerr)
			return ""
		}
		if werr := os.WriteFile(dst, data, 0o600); werr != nil {
			log.Printf("[简历] 归档原件写入失败 draft=%d teacher=%d: %v", draftID, teacherID, werr)
			return ""
		}
		_ = os.Remove(src)
	}
	return file
}

// removeDraftResumeFile 丢弃草稿时清理临时原件。
func (a *app) removeDraftResumeFile(draftID int64, kind string) {
	if p := a.draftResumePath(draftID, kind); p != "" {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("[简历] 清理草稿原件失败 draft=%d: %v", draftID, err)
		}
	}
}

// sweepResumeDraftFiles 启动时清理临时目录：草稿行重启后仍可查，但原件不再保留，
// 残留文件属于孤儿数据。
func (a *app) sweepResumeDraftFiles() {
	if a.resumeDraftDir == "" {
		return
	}
	entries, err := os.ReadDir(a.resumeDraftDir)
	if err != nil {
		return
	}
	var n int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(a.resumeDraftDir, e.Name())); err == nil {
			n++
		}
	}
	if n > 0 {
		log.Printf("[简历] 已清理草稿原件残留 %d 个", n)
	}
}

// removeTeacherResumeFiles 删除教师后清理其归档原件（文件残留不影响数据正确性，只记日志）。
func (a *app) removeTeacherResumeFiles(ts []*Teacher) {
	for _, t := range ts {
		p := a.resumeFilePath(t.ResumeFile)
		if p == "" {
			continue
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("[简历] 删除原件失败 teacher=%d: %v", t.ID, err)
		}
	}
}

// apiTeacherResumeDownload GET /api/teachers/{id}/resume —— 下载入库简历原件（需登录）。
func (a *app) apiTeacherResumeDownload(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		jsonError(w, http.StatusNotFound, "识别记录不存在")
		return
	}
	t, err := getTeacher(a.db, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	if t == nil {
		jsonError(w, http.StatusNotFound, "人员不存在")
		return
	}
	path := a.resumeFilePath(t.ResumeFile)
	if path == "" {
		jsonError(w, http.StatusNotFound, "该人员没有已入库的简历原件")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		jsonError(w, http.StatusNotFound, "简历文件已丢失")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		jsonError(w, http.StatusInternalServerError, "读取简历失败")
		return
	}
	kind := "docx"
	if strings.EqualFold(filepath.Ext(t.ResumeFile), ".pdf") {
		kind = "pdf"
	}
	if kind == "pdf" {
		w.Header().Set("Content-Type", "application/pdf")
	} else {
		w.Header().Set("Content-Type",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	}
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(t.Name+"-简历."+kind)))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, t.ResumeFile, st.ModTime(), f)
}

// ---------- HTTP handler ----------

// resumeView 是草稿的对外表示；payload 以对象原样回传。
type resumeView struct {
	ID        int64           `json:"id"`
	FileName  string          `json:"file_name"`
	FileType  string          `json:"file_type"`
	FileSize  int64           `json:"file_size"`
	PageCount int             `json:"page_count"`
	CreatedAt string          `json:"created_at"`
	Payload   json.RawMessage `json:"payload"`
	RawText   string          `json:"raw_text,omitempty"`
}

func toResumeView(d *ResumeDraft, withText bool) (resumeView, error) {
	v := resumeView{
		ID:        d.ID,
		FileName:  d.FileName,
		FileType:  d.FileType,
		FileSize:  d.FileSize,
		PageCount: d.PageCount,
		CreatedAt: d.CreatedAt,
		Payload:   json.RawMessage(d.Payload),
	}
	if withText {
		v.RawText = d.RawText
	}
	if !json.Valid(v.Payload) {
		v.Payload = json.RawMessage(`{}`)
	}
	return v, nil
}

// apiResumeParse 上传并同步解析简历：魔数校验 → 文本/页图 → 模型抽取 → 闸门 → 存草稿。
func (a *app) apiResumeParse(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	name, kind, data, ok := readResumeUpload(w, r)
	if !ok {
		return
	}
	cfg, err := a.loadLLMConfig()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "读取模型配置失败")
		return
	}
	if !cfg.ready() {
		jsonError(w, http.StatusServiceUnavailable, errModelNotConfigured.Error())
		return
	}

	in := resumeExtractInput{FileName: name, Kind: kind}
	switch kind {
	case "docx":
		text, err := docxText(data)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		in.Text = text
	case "pdf":
		text, pages, err := pdfText(data, resumeMaxPages)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		in.Text = text
		if meaningfulRune(text) < resumeMinTextRune {
			// 无可用文本层 → 按扫描件渲染页图
			pgs, err := renderPDFPages(r.Context(), data, resumeMaxPages)
			if err != nil {
				log.Printf("[简历] 渲染扫描件失败 file=%s: %v", name, err)
				jsonError(w, http.StatusInternalServerError, err.Error())
				return
			}
			in.Pages = pgs
			in.Text = "" // 扫描件不送文本，避免半截噪声
		} else {
			in.PageCount = pages
		}
		if pages > resumeMaxPages {
			log.Printf("[简历] 文件共 %d 页，仅识别前 %d 页 file=%s", pages, resumeMaxPages, name)
		}
	}
	if in.Text != "" {
		in.Text = collapseBlank(in.Text)
	}

	ex, err := a.runExtract(r.Context(), in)
	if err != nil {
		code, msg, detail := llmErrorToHTTP(err)
		if detail != "" {
			log.Printf("[简历] 抽取失败 file=%s: %s", name, detail)
		} else {
			log.Printf("[简历] 抽取失败 file=%s: %v", name, err)
		}
		jsonError(w, code, msg)
		return
	}

	payload := a.gateExtract(ex, in.Text)
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "保存识别结果失败")
		return
	}
	d := &ResumeDraft{
		FileName:  name,
		FileType:  kind,
		FileSize:  int64(len(data)),
		PageCount: in.PageCount,
		RawText:   truncateRunes(in.Text, resumeMaxPromptRune),
		Payload:   string(rawPayload),
	}
	id, err := insertResumeDraft(a.db, d)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "保存识别结果失败")
		return
	}
	d.ID = id
	a.saveDraftResumeFile(id, kind, data)
	d.CreatedAt = nowStr()
	log.Printf("[简历] 识别完成 file=%s type=%s size=%dKB pages=%d 图页=%d 抽取字段=%d 告警=%d 耗时=%s",
		name, kind, len(data)>>10, d.PageCount, len(in.Pages), len(payload.Meta), len(payload.Warnings), time.Since(start).Round(time.Millisecond))

	view, _ := toResumeView(d, true)
	jsonResp(w, http.StatusCreated, map[string]any{"data": view})
}

// apiResumeDraftList 草稿列表。
func (a *app) apiResumeDraftList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, size := pagination(q)
	items, total, err := listResumeDrafts(a.db, page, size)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询识别记录失败")
		return
	}
	views := make([]resumeView, 0, len(items))
	for i := range items {
		v, _ := toResumeView(&items[i], false)
		views = append(views, v)
	}
	jsonResp(w, http.StatusOK, map[string]any{
		"items": views, "total": total, "page": page, "size": size,
	})
}

// apiResumeDraftByID 草稿详情（GET）与丢弃（DELETE）。
func (a *app) apiResumeDraftByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		d, err := getResumeDraft(a.db, id)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "查询识别记录失败")
			return
		}
		if d == nil {
			jsonError(w, http.StatusNotFound, "识别记录不存在")
			return
		}
		v, _ := toResumeView(d, true)
		jsonResp(w, http.StatusOK, map[string]any{"data": v})
	case http.MethodDelete:
		d, err := getResumeDraft(a.db, id)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "查询识别记录失败")
			return
		}
		if d == nil {
			jsonError(w, http.StatusNotFound, "识别记录不存在")
			return
		}
		if err := deleteResumeDraft(a.db, id); err != nil {
			jsonError(w, http.StatusInternalServerError, "删除识别记录失败")
			return
		}
		a.removeDraftResumeFile(id, d.FileType)
		jsonResp(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": true}})
	default:
		jsonError(w, http.StatusMethodNotAllowed, "方法不支持")
	}
}

// apiResumeCommit 用人工修正后的字段入库，复用 validateTeacher + createTeacher。
func (a *app) apiResumeCommit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var body struct {
		Fields teacherBody `json:"fields"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	in, errMsg := a.validateTeacher(body.Fields)
	if errMsg != "" {
		jsonError(w, http.StatusBadRequest, errMsg)
		return
	}
	d, err := getResumeDraft(a.db, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询识别记录失败")
		return
	}
	if d == nil {
		jsonError(w, http.StatusNotFound, "识别记录不存在")
		return
	}
	tid, err := createTeacher(a.db, in)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "创建失败")
		return
	}
	// 归档原件：草稿临时文件移入永久目录（无临时目录/无原件时跳过，仅存文本草稿）
	if file := a.commitResumeFile(id, d.FileType, tid); file != "" {
		if err := setTeacherResumeFile(a.db, tid, file); err != nil {
			log.Printf("[简历] 写入 resume_file 失败 teacher=%d: %v", tid, err)
		}
	}
	if err := deleteResumeDraft(a.db, id); err != nil {
		log.Printf("[简历] 入库后删除草稿失败 id=%d: %v", id, err)
	}
	log.Printf("[简历] 已入库 draft=%d teacher=%d name=%s", id, tid, in.name)
	t, _ := getTeacher(a.db, tid)
	jsonResp(w, http.StatusCreated, map[string]any{"data": t})
}

// certPositiveEvidence 描述「持有教师资格证」的证据（预编译，避免每次调用编译）。
var certPositiveEvidence = regexp.MustCompile(`教师资格证|资格证|持证`)

// certNegativeEvidence 描述「未持有教师资格证」的证据。
var certNegativeEvidence = regexp.MustCompile(`无[^，。；\n]*教师资格证|没有[^，。；\n]*资格证|未取得[^，。；\n]*资格证`)
