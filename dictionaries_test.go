package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

// dictOf 读取指定类型的字典列表。
func dictOf(t *testing.T, srv *httptest.Server, c *http.Cookie, kind string) []Dictionary {
	t.Helper()
	resp := req(t, srv, http.MethodGet, "/api/dictionaries/"+kind, nil, c)
	requireStatus(t, resp, http.StatusOK)
	var payload struct {
		Data []Dictionary `json:"data"`
	}
	decodeBody(t, resp, &payload)
	return payload.Data
}

func TestDictionaryListRequiresAuth(t *testing.T) {
	_, srv := newTestApp(t)
	resp := req(t, srv, http.MethodGet, "/api/dictionaries/subject", nil, nil)
	requireStatus(t, resp, http.StatusUnauthorized)
}

func TestDictionaryUnknownKind(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)
	resp := req(t, srv, http.MethodGet, "/api/dictionaries/nope", nil, c)
	requireStatus(t, resp, http.StatusNotFound)
	resp = req(t, srv, http.MethodPost, "/api/dictionaries/nope",
		map[string]any{"name": "地理"}, c)
	requireStatus(t, resp, http.StatusNotFound)
}

func TestDictionaryCreateUpdateDelete(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)

	// 新增
	resp := req(t, srv, http.MethodPost, "/api/dictionaries/subject",
		map[string]any{"name": "天文", "color": "#ef4444", "sort": 99, "enabled": 1}, c)
	requireStatus(t, resp, http.StatusCreated)
	var created struct {
		Data Dictionary `json:"data"`
	}
	decodeBody(t, resp, &created)
	if created.Data.Name != "天文" || created.Data.Enabled != 1 {
		t.Fatalf("新增结果不符: %+v", created.Data)
	}
	id := created.Data.ID

	// 重名拒绝
	resp = req(t, srv, http.MethodPost, "/api/dictionaries/subject",
		map[string]any{"name": "天文", "sort": 1, "enabled": 1}, c)
	requireStatus(t, resp, http.StatusBadRequest)

	// 编辑
	resp = req(t, srv, http.MethodPut, "/api/dictionaries/subject/"+itoa(id),
		map[string]any{"name": "天文学", "color": "#22c55e", "sort": 5, "enabled": 0}, c)
	requireStatus(t, resp, http.StatusOK)
	var updated struct {
		Data Dictionary `json:"data"`
	}
	decodeBody(t, resp, &updated)
	if updated.Data.Name != "天文学" || updated.Data.Enabled != 0 {
		t.Fatalf("编辑结果不符: %+v", updated.Data)
	}

	// 删除
	resp = req(t, srv, http.MethodDelete, "/api/dictionaries/subject/"+itoa(id), nil, c)
	requireStatus(t, resp, http.StatusOK)
	for _, d := range dictOf(t, srv, c, "subject") {
		if d.ID == id {
			t.Fatal("删除后仍存在")
		}
	}
}

func TestDictionaryValidation(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)

	cases := []struct {
		name string
		body map[string]any
		want string
	}{
		{"空名称", map[string]any{"name": "   "}, "名称不能为空"},
		{"超长名称", map[string]any{"name": repeatRune('名', 33)}, "名称过长（最多 32 个字符）"},
		{"非法颜色", map[string]any{"name": "地理", "color": "red"}, "颜色格式须为 #RRGGBB"},
		{"非法 enabled", map[string]any{"name": "地理", "enabled": 7}, "enabled 只能为 0 或 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := req(t, srv, http.MethodPost, "/api/dictionaries/subject", tc.body, c)
			requireStatus(t, resp, http.StatusBadRequest)
			if msg := errorOf(t, resp); msg != tc.want {
				t.Fatalf("错误信息 = %q，期望 %q", msg, tc.want)
			}
		})
	}
}

func repeatRune(r rune, n int) string {
	s := make([]rune, 0, n)
	for i := 0; i < n; i++ {
		s = append(s, r)
	}
	return string(s)
}

func TestDictionaryRenameCascadesToTeachers(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)

	// 找到内置学科 "语文" 的 id
	var subjID int64
	for _, d := range dictOf(t, srv, c, "subject") {
		if d.Name == "语文" {
			subjID = d.ID
			break
		}
	}
	if subjID == 0 {
		t.Fatal("未找到内置学科 语文")
	}

	tid := createTeacherVia(t, srv, c, map[string]any{
		"name": "级联老师", "gender": "female", "age": 33, "subject": "语文",
		"has_cert": 1, "education": "本科",
	})

	// 改名
	resp := req(t, srv, http.MethodPut, "/api/dictionaries/subject/"+itoa(subjID),
		map[string]any{"name": "语文（改）", "sort": 1, "enabled": 1}, c)
	requireStatus(t, resp, http.StatusOK)

	// 教师引用同步更新
	resp = req(t, srv, http.MethodGet, "/api/teachers/"+itoa(tid), nil, c)
	requireStatus(t, resp, http.StatusOK)
	var payload struct {
		Data Teacher `json:"data"`
	}
	decodeBody(t, resp, &payload)
	if payload.Data.Subject != "语文（改）" {
		t.Fatalf("级联更新失败，subject = %q", payload.Data.Subject)
	}
}

func TestDictionaryDeleteRejectedWhenReferenced(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)

	// 建一个专用字典并被引用
	resp := req(t, srv, http.MethodPost, "/api/dictionaries/subject",
		map[string]any{"name": "被引用学科", "sort": 1, "enabled": 1}, c)
	requireStatus(t, resp, http.StatusCreated)
	var created struct {
		Data Dictionary `json:"data"`
	}
	decodeBody(t, resp, &created)

	tid := createTeacherVia(t, srv, c, map[string]any{
		"name": "引用者", "gender": "male", "age": 40, "subject": "被引用学科",
		"has_cert": 0, "education": "本科",
	})

	resp = req(t, srv, http.MethodDelete, "/api/dictionaries/subject/"+itoa(created.Data.ID), nil, c)
	requireStatus(t, resp, http.StatusConflict)
	if msg := errorOf(t, resp); msg == "" {
		t.Fatal("409 应携带错误信息")
	}

	// 断开引用后可删除
	resp = req(t, srv, http.MethodPost, "/api/teachers/batch-delete",
		map[string]any{"ids": []int64{tid}}, c)
	requireStatus(t, resp, http.StatusOK)
	resp = req(t, srv, http.MethodDelete, "/api/dictionaries/subject/"+itoa(created.Data.ID), nil, c)
	requireStatus(t, resp, http.StatusOK)
}

func TestDictionaryDisabledStillAllowedForExistingRecords(t *testing.T) {
	_, srv := newTestApp(t)
	c := loginAs(t, srv)

	// 停用某学科后，已有记录可继续编辑保存
	var subjID int64
	var subjName string
	for _, d := range dictOf(t, srv, c, "subject") {
		subjID, subjName = d.ID, d.Name
		break
	}
	resp := req(t, srv, http.MethodPut, "/api/dictionaries/subject/"+itoa(subjID),
		map[string]any{"name": subjName, "sort": 1, "enabled": 0}, c)
	requireStatus(t, resp, http.StatusOK)

	tid := createTeacherVia(t, srv, c, map[string]any{
		"name": "在职", "gender": "female", "age": 28, "subject": subjName,
		"has_cert": 1, "education": "本科",
	})
	upd := map[string]any{
		"name": "在职", "gender": "female", "age": 29, "subject": subjName,
		"has_cert": 1, "education": "本科", "remark": "已停用学科仍可保留",
	}
	resp = req(t, srv, http.MethodPut, "/api/teachers/"+itoa(tid), upd, c)
	requireStatus(t, resp, http.StatusOK)
}
