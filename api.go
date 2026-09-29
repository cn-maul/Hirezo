package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ---------- 通用响应 ----------

func jsonResp(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	jsonResp(w, code, map[string]any{
		"error": map[string]any{"code": code, "message": msg},
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("请求体为空")
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB 上限
	return json.NewDecoder(r.Body).Decode(v)
}

func parseID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

// pagination 从查询参数解析分页；非法值回退默认，size 封顶 100。
func pagination(q url.Values) (page, size int) {
	page, _ = strconv.Atoi(q.Get("page"))
	size, _ = strconv.Atoi(q.Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

// buildTeacherQuery 从查询参数构造教师列表条件。
// 非法的 gender/hasCert/order 返回 ok=false。
// order 是唯一进入 SQL 拼接的值，此处做白名单校验（防注入）。
func buildTeacherQuery(q url.Values, page, size int) (teacherQuery, bool) {
	gender := q.Get("gender")
	if gender != "" && gender != "male" && gender != "female" {
		return teacherQuery{}, false
	}
	hasCert := -1 // -1 = 不筛
	if v := q.Get("hasCert"); v != "" {
		switch v {
		case "1", "true":
			hasCert = 1
		case "0", "false":
			hasCert = 0
		default:
			return teacherQuery{}, false
		}
	}
	order := q.Get("order")
	if order == "" {
		order = "desc"
	}
	if !strings.EqualFold(order, "asc") && !strings.EqualFold(order, "desc") {
		return teacherQuery{}, false
	}
	return teacherQuery{
		keyword:   strings.TrimSpace(q.Get("keyword")),
		subject:   strings.TrimSpace(q.Get("subject")),
		gender:    gender,
		education: strings.TrimSpace(q.Get("education")),
		hasCert:   hasCert,
		page:      page,
		size:      size,
		order:     order,
	}, true
}

// --------------------------------------------------------------------
// 健康检查
// --------------------------------------------------------------------

func (a *app) apiHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": true}})
}

// --------------------------------------------------------------------
// 教师列表（分页 + 姓名/学科/性别/学历/证书筛选）
// --------------------------------------------------------------------

func (a *app) apiTeacherList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, size := pagination(q)
	tq, ok := buildTeacherQuery(q, page, size)
	if !ok {
		jsonError(w, http.StatusBadRequest, "查询参数非法（gender/hasCert/order）")
		return
	}
	items, total, err := listTeachers(a.db, tq)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	if items == nil {
		items = []Teacher{}
	}
	jsonResp(w, http.StatusOK, map[string]any{
		"items": items, "total": total, "page": page, "size": size,
	})
}

// teacherBody 新增 / 编辑共用的请求体。
type teacherBody struct {
	Name       string `json:"name"`
	Gender     string `json:"gender"`
	Age        int    `json:"age"`
	Subject    string `json:"subject"`
	HasCert    int    `json:"has_cert"`
	Phone      string `json:"phone"`
	Education  string `json:"education"`
	University string `json:"university"`
	Major      string `json:"major"`
	Remark     string `json:"remark"`
}

// validateTeacher 校验并归一化教师字段；失败返回中文错误信息（空串表示通过）。
func (a *app) validateTeacher(b teacherBody) (teacherInput, string) {
	in := teacherInput{
		name:       strings.TrimSpace(b.Name),
		gender:     strings.TrimSpace(b.Gender),
		age:        b.Age,
		subject:    strings.TrimSpace(b.Subject),
		hasCert:    b.HasCert,
		phone:      strings.TrimSpace(b.Phone),
		education:  strings.TrimSpace(b.Education),
		university: strings.TrimSpace(b.University),
		major:      strings.TrimSpace(b.Major),
		remark:     strings.TrimSpace(b.Remark),
	}
	if in.name == "" {
		return in, "姓名不能为空"
	}
	if len([]rune(in.name)) > 50 {
		return in, "姓名过长（最多 50 个字符）"
	}
	if in.gender != "male" && in.gender != "female" {
		return in, "性别只能是男或女"
	}
	if in.age != 0 && (in.age < 18 || in.age > 100) {
		return in, "年龄须在 18-100 之间"
	}
	if in.subject != "" && !validDictValue(a.db, dictKindSubject, in.subject) {
		return in, "学科不存在或已失效"
	}
	if in.hasCert != 0 && in.hasCert != 1 {
		return in, "教师资格证取值非法"
	}
	if in.phone != "" && !validPhone(in.phone) {
		return in, "请输入正确的 11 位手机号"
	}
	if in.education != "" && !validDictValue(a.db, dictKindEducation, in.education) {
		return in, "学历不存在或已失效"
	}
	if len([]rune(in.university)) > 100 {
		return in, "毕业院校过长（最多 100 个字符）"
	}
	if len([]rune(in.major)) > 100 {
		return in, "专业过长（最多 100 个字符）"
	}
	if len([]rune(in.remark)) > 500 {
		return in, "备注过长（最多 500 个字符）"
	}
	return in, ""
}

func (a *app) apiTeacherCreate(w http.ResponseWriter, r *http.Request) {
	var body teacherBody
	if err := decodeJSON(w, r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	in, errMsg := a.validateTeacher(body)
	if errMsg != "" {
		jsonError(w, http.StatusBadRequest, errMsg)
		return
	}
	id, err := createTeacher(a.db, in)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "创建失败")
		return
	}
	t, _ := getTeacher(a.db, id)
	jsonResp(w, http.StatusCreated, map[string]any{"data": t})
}

// apiTeacherByID 单条教师：GET 读取 / PUT 编辑 / DELETE 删除。
func (a *app) apiTeacherByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		t, err := getTeacher(a.db, id)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "查询失败")
			return
		}
		if t == nil {
			http.NotFound(w, r)
			return
		}
		jsonResp(w, http.StatusOK, map[string]any{"data": t})
	case http.MethodPut:
		a.apiTeacherUpdate(w, r, id)
	case http.MethodDelete:
		t, err := getTeacher(a.db, id)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "查询失败")
			return
		}
		if t == nil {
			http.NotFound(w, r)
			return
		}
		if err := deleteTeacher(a.db, id); err != nil {
			jsonError(w, http.StatusInternalServerError, "删除失败")
			return
		}
		jsonResp(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": true}})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) apiTeacherUpdate(w http.ResponseWriter, r *http.Request, id int64) {
	t, err := getTeacher(a.db, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	if t == nil {
		http.NotFound(w, r)
		return
	}
	var body teacherBody
	if err := decodeJSON(w, r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	in, errMsg := a.validateTeacher(body)
	if errMsg != "" {
		jsonError(w, http.StatusBadRequest, errMsg)
		return
	}
	if err := updateTeacher(a.db, id, in); err != nil {
		jsonError(w, http.StatusInternalServerError, "保存失败")
		return
	}
	t, _ = getTeacher(a.db, id)
	jsonResp(w, http.StatusOK, map[string]any{"data": t})
}

const maxBatchIDs = 500

// apiTeacherBatchDelete POST /api/teachers/batch-delete —— 批量硬删除。
func (a *app) apiTeacherBatchDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if len(body.IDs) == 0 || len(body.IDs) > maxBatchIDs {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("ids 数量须为 1-%d", maxBatchIDs))
		return
	}
	for _, id := range body.IDs {
		if id <= 0 {
			jsonError(w, http.StatusBadRequest, "ids 含非法值")
			return
		}
	}
	n, err := batchDeleteTeachers(a.db, body.IDs)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "批量删除失败")
		return
	}
	jsonResp(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": true, "deleted": n}})
}

// --------------------------------------------------------------------
// 字典 CRUD（subject / education）
// --------------------------------------------------------------------

// apiDictionaryList GET /api/dictionaries/{kind}
func (a *app) apiDictionaryList(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if !validDictKind(kind) {
		http.NotFound(w, r)
		return
	}
	ds, err := allDictionaries(a.db, kind)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	if ds == nil {
		ds = []Dictionary{}
	}
	jsonResp(w, http.StatusOK, map[string]any{"data": ds})
}

// apiDictionaryCreate POST /api/dictionaries/{kind}
func (a *app) apiDictionaryCreate(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if !validDictKind(kind) {
		http.NotFound(w, r)
		return
	}
	var body struct {
		Name    string `json:"name"`
		Color   string `json:"color"`
		Sort    int    `json:"sort"`
		Enabled *int   `json:"enabled"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		jsonError(w, http.StatusBadRequest, "名称不能为空")
		return
	}
	if len([]rune(name)) > 32 {
		jsonError(w, http.StatusBadRequest, "名称过长（最多 32 个字符）")
		return
	}
	color := strings.TrimSpace(body.Color)
	if color == "" {
		color = "#2563eb"
	}
	if !validColor(color) {
		jsonError(w, http.StatusBadRequest, "颜色格式须为 #RRGGBB")
		return
	}
	enabled := 1
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	if enabled != 0 && enabled != 1 {
		jsonError(w, http.StatusBadRequest, "enabled 只能为 0 或 1")
		return
	}
	if c, _ := getDictionaryByName(a.db, kind, name); c != nil {
		jsonError(w, http.StatusBadRequest, "该类型下名称已存在")
		return
	}
	if _, err := a.db.Exec(
		"INSERT INTO dictionaries (kind, name, color, sort, enabled) VALUES (?, ?, ?, ?, ?)",
		kind, name, color, body.Sort, enabled); err != nil {
		jsonError(w, http.StatusInternalServerError, "创建失败")
		return
	}
	c, _ := getDictionaryByName(a.db, kind, name)
	jsonResp(w, http.StatusCreated, map[string]any{"data": c})
}

// apiDictionaryByID PUT / DELETE /api/dictionaries/{kind}/{id}
func (a *app) apiDictionaryByID(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if !validDictKind(kind) {
		http.NotFound(w, r)
		return
	}
	id, ok := parseID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		a.apiDictionaryUpdate(w, r, kind, id)
	case http.MethodDelete:
		a.apiDictionaryDelete(w, r, kind, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// apiDictionaryUpdate 部分更新：仅修改请求体中提供的字段，未提供字段保持不变。
// 名称变更时在同一事务内级联更新 teachers 表引用，避免产生孤儿数据。
func (a *app) apiDictionaryUpdate(w http.ResponseWriter, r *http.Request, kind string, id int64) {
	var body struct {
		Name    *string `json:"name"`
		Color   *string `json:"color"`
		Sort    *int    `json:"sort"`
		Enabled *int    `json:"enabled"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		jsonError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	c, err := getDictionaryByID(a.db, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	if c == nil || c.Kind != kind {
		http.NotFound(w, r)
		return
	}

	name := c.Name
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
		if name == "" {
			jsonError(w, http.StatusBadRequest, "名称不能为空")
			return
		}
		if len([]rune(name)) > 32 {
			jsonError(w, http.StatusBadRequest, "名称过长（最多 32 个字符）")
			return
		}
		if other, _ := getDictionaryByName(a.db, kind, name); other != nil && other.ID != id {
			jsonError(w, http.StatusBadRequest, "该类型下名称已存在")
			return
		}
	}
	color := c.Color
	if body.Color != nil {
		color = strings.TrimSpace(*body.Color)
		if !validColor(color) {
			jsonError(w, http.StatusBadRequest, "颜色格式须为 #RRGGBB")
			return
		}
	}
	sort := c.Sort
	if body.Sort != nil {
		sort = *body.Sort
	}
	enabled := c.Enabled
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	tx, err := a.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "更新失败")
		return
	}
	if name != c.Name {
		// 级联改名：列名来自白名单函数，值走占位符
		col, _ := teacherDictColumn(kind)
		if _, err := tx.Exec("UPDATE teachers SET "+col+" = ? WHERE "+col+" = ?", name, c.Name); err != nil {
			_ = tx.Rollback()
			jsonError(w, http.StatusInternalServerError, "更新失败")
			return
		}
	}
	if _, err := tx.Exec("UPDATE dictionaries SET name=?, color=?, sort=?, enabled=? WHERE id=?",
		name, color, sort, enabled, id); err != nil {
		_ = tx.Rollback()
		jsonError(w, http.StatusInternalServerError, "更新失败")
		return
	}
	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, "更新失败")
		return
	}
	c, _ = getDictionaryByID(a.db, id)
	jsonResp(w, http.StatusOK, map[string]any{"data": c})
}

// apiDictionaryDelete 被教师引用时返回 409，避免历史数据悬空。
func (a *app) apiDictionaryDelete(w http.ResponseWriter, r *http.Request, kind string, id int64) {
	c, err := getDictionaryByID(a.db, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	if c == nil || c.Kind != kind {
		http.NotFound(w, r)
		return
	}
	n, err := countTeachersUsingDict(a.db, kind, c.Name)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	if n > 0 {
		jsonError(w, http.StatusConflict, fmt.Sprintf("该%s已被 %d 名教师使用，无法删除", dictKindLabel(kind), n))
		return
	}
	res, err := a.db.Exec("DELETE FROM dictionaries WHERE id=?", id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "删除失败")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.NotFound(w, r)
		return
	}
	jsonResp(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": true}})
}

// dictKindLabel 字典类型的中文名（错误文案用）。
func dictKindLabel(kind string) string {
	if kind == dictKindEducation {
		return "学历"
	}
	return "学科"
}

// --------------------------------------------------------------------
// 导出（xlsx，沿用列表筛选口径）
// --------------------------------------------------------------------

func (a *app) apiExportXLSX(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tq, ok := buildTeacherQuery(q, 1, 100)
	if !ok {
		jsonError(w, http.StatusBadRequest, "查询参数非法（gender/hasCert/order）")
		return
	}
	items, err := listTeachersForExport(a.db, tq)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "查询失败")
		return
	}

	f := excelize.NewFile()
	defer f.Close()
	const sheet = "教师"
	f.SetSheetName("Sheet1", sheet)

	headers := []any{"姓名", "性别", "年龄", "学科", "教师资格证", "联系电话", "学历", "毕业院校", "专业", "备注", "录入时间", "更新时间"}
	if err := f.SetSheetRow(sheet, "A1", &headers); err != nil {
		jsonError(w, http.StatusInternalServerError, "生成表格失败")
		return
	}
	genderLabel := map[string]string{"male": "男", "female": "女"}
	certLabel := map[int]string{0: "无", 1: "有"}
	for i, t := range items {
		row := []any{
			t.Name,
			genderLabel[t.Gender],
			ageOrDash(t.Age),
			t.Subject,
			certLabel[t.HasCert],
			t.Phone,
			t.Education,
			t.University,
			t.Major,
			t.Remark,
			t.CreatedAt,
			t.UpdatedAt,
		}
		cell := fmt.Sprintf("A%d", i+2)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			jsonError(w, http.StatusInternalServerError, "生成表格失败")
			return
		}
	}
	// 列宽（中文按约 2 字符宽估算）
	for i, wpx := range []float64{12, 7, 7, 12, 12, 15, 9, 22, 16, 30, 20, 20} {
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, col, col, wpx)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "生成表格失败")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="teachers-%s.xlsx"`, time.Now().Format("20060102")))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	_, _ = w.Write(buf.Bytes())
}

// ageOrDash 年龄 0 表示未填，导出为 “-”。
func ageOrDash(age int) any {
	if age <= 0 {
		return "-"
	}
	return age
}

// --------------------------------------------------------------------
// 工具函数
// --------------------------------------------------------------------

// clientIP 从请求中提取客户端IP。
// 仅在显式开启 -trust-proxy（部署于反向代理之后）时才信任
// X-Forwarded-For / X-Real-IP 头，否则直连客户端可伪造这些头绕过限流。
func (a *app) clientIP(r *http.Request) string {
	if a.trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// X-Forwarded-For: client, proxy1, proxy2
			if idx := strings.IndexByte(xff, ','); idx > 0 {
				return strings.TrimSpace(xff[:idx])
			}
			return xff
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}
	// 去掉端口号（兼容 IPv4 / IPv6，如 [::1]:8080 → ::1）
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// validPhone 校验 11 位大陆手机号（1 开头、第二位 3-9）。
func validPhone(p string) bool {
	if len(p) != 11 {
		return false
	}
	if p[0] != '1' || p[1] < '3' || p[1] > '9' {
		return false
	}
	for i := 2; i < 11; i++ {
		if p[i] < '0' || p[i] > '9' {
			return false
		}
	}
	return true
}

// validColor 校验十六进制颜色 #RRGGBB。
func validColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, ch := range s[1:] {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F') {
			return false
		}
	}
	return true
}
