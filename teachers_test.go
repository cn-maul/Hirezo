package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// createTeacherVia 通过 API 创建教师并返回 id。
func createTeacherVia(t *testing.T, srv *httptest.Server, cookie *http.Cookie, body map[string]any) int64 {
	t.Helper()
	resp := req(t, srv, http.MethodPost, "/api/teachers", body, cookie)
	requireStatus(t, resp, http.StatusCreated)
	var payload struct {
		Data Teacher `json:"data"`
	}
	decodeBody(t, resp, &payload)
	return payload.Data.ID
}

func baseTeacher(name string) map[string]any {
	return map[string]any{
		"name":       name,
		"gender":     "female",
		"age":        30,
		"subject":    "语文",
		"has_cert":   1,
		"phone":      "13800138000",
		"education":  "本科",
		"university": "北京师范大学",
		"major":      "汉语言文学",
		"remark":     "",
	}
}

func TestTeacherCRUD(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)

	// 新增
	id := createTeacherVia(t, srv, c, baseTeacher("张三"))

	// 读取
	resp := req(t, srv, http.MethodGet, "/api/teachers/"+itoa(id), nil, c)
	requireStatus(t, resp, http.StatusOK)
	var got struct {
		Data Teacher `json:"data"`
	}
	decodeBody(t, resp, &got)
	if got.Data.Name != "张三" || got.Data.Subject != "语文" || got.Data.HasCert != 1 {
		t.Fatalf("详情不符: %+v", got.Data)
	}

	// 编辑
	upd := baseTeacher("李四")
	upd["gender"] = "male"
	upd["age"] = 42
	upd["subject"] = "数学"
	upd["has_cert"] = 0
	upd["phone"] = ""
	resp = req(t, srv, http.MethodPut, "/api/teachers/"+itoa(id), upd, c)
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &got)
	if got.Data.Name != "李四" || got.Data.Gender != "male" || got.Data.Subject != "数学" || got.Data.HasCert != 0 {
		t.Fatalf("编辑后不符: %+v", got.Data)
	}

	// 删除
	resp = req(t, srv, http.MethodDelete, "/api/teachers/"+itoa(id), nil, c)
	requireStatus(t, resp, http.StatusOK)
	resp = req(t, srv, http.MethodGet, "/api/teachers/"+itoa(id), nil, c)
	requireStatus(t, resp, http.StatusNotFound)
}

func TestTeacherValidation(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)

	cases := []struct {
		name string
		mut  func(m map[string]any)
		want string
	}{
		{"空姓名", func(m map[string]any) { m["name"] = "  " }, "姓名不能为空"},
		{"非法性别", func(m map[string]any) { m["gender"] = "x" }, "性别只能是男或女"},
		{"年龄过小", func(m map[string]any) { m["age"] = 10 }, "年龄须在 18-100 之间"},
		{"年龄过大", func(m map[string]any) { m["age"] = 150 }, "年龄须在 18-100 之间"},
		{"未知学科", func(m map[string]any) { m["subject"] = "不存在的学科" }, "学科不存在或已失效"},
		{"非法证书", func(m map[string]any) { m["has_cert"] = 5 }, "教师资格证取值非法"},
		{"非法手机号", func(m map[string]any) { m["phone"] = "12345" }, "请输入正确的 11 位手机号"},
		{"未知学历", func(m map[string]any) { m["education"] = "博士后" }, "学历不存在或已失效"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := baseTeacher("测试")
			tc.mut(body)
			resp := req(t, srv, http.MethodPost, "/api/teachers", body, c)
			requireStatus(t, resp, http.StatusBadRequest)
			if msg := errorOf(t, resp); msg != tc.want {
				t.Fatalf("错误信息 = %q，期望 %q", msg, tc.want)
			}
		})
	}

	// 超长姓名（51 个字符）
	long := baseTeacher("测试")
	long["name"] = repeatRune('王', 51)
	resp := req(t, srv, http.MethodPost, "/api/teachers", long, c)
	requireStatus(t, resp, http.StatusBadRequest)
	if msg := errorOf(t, resp); msg != "姓名过长（最多 50 个字符）" {
		t.Fatalf("错误信息 = %q", msg)
	}

	// 可选字段留空是合法的
	optional := map[string]any{"name": "无信息", "gender": "male", "age": 0, "has_cert": 0}
	createTeacherVia(t, srv, c, optional)
}

func TestTeacherListFilters(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)

	createTeacherVia(t, srv, c, baseTeacher("甲老师"))
	b := baseTeacher("乙老师")
	b["gender"] = "male"
	b["subject"] = "数学"
	b["has_cert"] = 0
	b["education"] = "硕士"
	createTeacherVia(t, srv, c, b)
	c3 := baseTeacher("丙老师")
	c3["subject"] = "英语"
	createTeacherVia(t, srv, c, c3)

	type listResp struct {
		Items []Teacher `json:"items"`
		Total int       `json:"total"`
		Page  int       `json:"page"`
		Size  int       `json:"size"`
	}
	get := func(query string) listResp {
		t.Helper()
		resp := req(t, srv, http.MethodGet, "/api/teachers"+query, nil, c)
		requireStatus(t, resp, http.StatusOK)
		var out listResp
		decodeBody(t, resp, &out)
		return out
	}

	if got := get(""); got.Total != 3 {
		t.Fatalf("全量 total = %d，期望 3", got.Total)
	}
	if got := get("?keyword=乙"); got.Total != 1 || got.Items[0].Name != "乙老师" {
		t.Fatalf("关键字筛选异常: %+v", got)
	}
	if got := get("?gender=female"); got.Total != 2 {
		t.Fatalf("性别筛选 total = %d，期望 2", got.Total)
	}
	if got := get("?subject=数学"); got.Total != 1 || got.Items[0].Name != "乙老师" {
		t.Fatalf("学科筛选异常: %+v", got)
	}
	if got := get("?hasCert=1"); got.Total != 2 {
		t.Fatalf("证书筛选 total = %d，期望 2", got.Total)
	}
	if got := get("?education=硕士"); got.Total != 1 {
		t.Fatalf("学历筛选 total = %d，期望 1", got.Total)
	}
	// 通配符须按字面量处理（% 在查询串中写作 %25）
	if got := get("?keyword=%25"); got.Total != 0 {
		t.Fatalf("LIKE 通配符应被转义，total = %d", got.Total)
	}

	// 分页
	if got := get("?page=2&size=2"); got.Total != 3 || len(got.Items) != 1 {
		t.Fatalf("分页异常: total=%d items=%d", got.Total, len(got.Items))
	}

	// 非法参数
	for _, q := range []string{"?gender=x", "?hasCert=maybe", "?order=;DROP TABLE"} {
		resp := req(t, srv, http.MethodGet, "/api/teachers"+q, nil, c)
		requireStatus(t, resp, http.StatusBadRequest)
	}
}

func TestTeacherBatchDelete(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)
	id1 := createTeacherVia(t, srv, c, baseTeacher("甲"))
	id2 := createTeacherVia(t, srv, c, baseTeacher("乙"))

	resp := req(t, srv, http.MethodPost, "/api/teachers/batch-delete",
		map[string]any{"ids": []int64{id1, id2}}, c)
	requireStatus(t, resp, http.StatusOK)
	var payload struct {
		Data struct {
			Deleted int64 `json:"deleted"`
		} `json:"data"`
	}
	decodeBody(t, resp, &payload)
	if payload.Data.Deleted != 2 {
		t.Fatalf("deleted = %d，期望 2", payload.Data.Deleted)
	}

	// 空列表与超量列表均拒绝
	resp = req(t, srv, http.MethodPost, "/api/teachers/batch-delete", map[string]any{"ids": []int64{}}, c)
	requireStatus(t, resp, http.StatusBadRequest)
	resp = req(t, srv, http.MethodPost, "/api/teachers/batch-delete",
		map[string]any{"ids": make([]int64, 501)}, c)
	requireStatus(t, resp, http.StatusBadRequest)
}

func TestTeacherExportXLSX(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)
	createTeacherVia(t, srv, c, baseTeacher("导出老师"))

	resp := req(t, srv, http.MethodGet, "/api/teachers/export?subject=语文", nil, c)
	requireStatus(t, resp, http.StatusOK)
	ct := resp.Header.Get("Content-Type")
	if ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if disp := resp.Header.Get("Content-Disposition"); !strings.Contains(disp, "teachers-") {
		t.Fatalf("Content-Disposition = %q", disp)
	}
	// xlsx 本质是 zip，应以 PK 开头
	head := make([]byte, 2)
	if _, err := resp.Body.Read(head); err != nil {
		t.Fatalf("读取导出内容失败: %v", err)
	}
	if string(head) != "PK" {
		t.Fatalf("导出内容不是 xlsx（zip）格式: %q", head)
	}
}
