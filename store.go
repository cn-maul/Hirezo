package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ---------- Rate Limiter ----------

// rateLimiter 基于IP的简单限流器：每分钟最多N次请求。
type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// maxRateKeys 触发全表清扫的 key 数阈值：防止大量一次性 key
// （如伪造 X-Forwarded-For 的请求）导致内存无界增长。
const maxRateKeys = 4096

// maxRateKeyEntries 单个 key 允许保留的时间戳数量上限：
// 即便限流已打满且请求持续涌入，也不会让该 key 的无用时间戳无限堆积。
const maxRateKeyEntries = 256

// rateSweepInterval 后台定期全表清扫间隔：过期 key 不再依赖请求触发，内存上限更可控。
const rateSweepInterval = time.Minute

// allow 检查给定key在时间窗口内的请求是否超过限制。
func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	// 清理过期记录（顺带截断超长时间戳，防止单 key 无界增长）
	timestamps := r.requests[key]
	valid := timestamps[:0]
	for _, t := range timestamps {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}
	if len(valid) > maxRateKeyEntries {
		valid = valid[len(valid)-maxRateKeyEntries:]
	}
	r.requests[key] = valid

	if len(valid) >= r.limit {
		return false
	}
	r.requests[key] = append(r.requests[key], now)
	return true
}

// startSweep 启动后台周期清扫 goroutine：固定间隔清理过期 key，
// 全表 key 数峰值受限于 窗口内请求速率 × 地址数，内存不再随请求量线性增长。
func (r *rateLimiter) startSweep(stop <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(rateSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				r.sweep()
			}
		}
	}()
}

// sweep 全表清扫一次过期 key。
func (r *rateLimiter) sweep() {
	r.mu.Lock()
	defer r.mu.Unlock()
	windowStart := time.Now().Add(-r.window)
	for k, ts := range r.requests {
		if len(ts) == 0 || !ts[len(ts)-1].After(windowStart) {
			delete(r.requests, k)
		}
	}
}

// 限流器实例挂在 app 上（见 app.loginLimiter），
// 每个应用实例独立计数，测试可按实例替换而互不干扰。

// ---------- 数据模型 ----------

// 字典类型白名单：subject（学科）/ education（学历）。
// 同时是 teachers 表对应列名的唯一合法来源（列名拼接只允许来自这里）。
const (
	dictKindSubject   = "subject"
	dictKindEducation = "education"
)

// teacherDictColumn 返回字典类型对应的 teachers 表列名。
func teacherDictColumn(kind string) (string, bool) {
	switch kind {
	case dictKindSubject:
		return "subject", true
	case dictKindEducation:
		return "education", true
	}
	return "", false
}

func validDictKind(kind string) bool {
	_, ok := teacherDictColumn(kind)
	return ok
}

// Dictionary 字典项（学科 / 学历共用一张表，按 kind 区分）。
type Dictionary struct {
	ID      int64  `json:"id"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Color   string `json:"color"`
	Sort    int    `json:"sort"`
	Enabled int    `json:"enabled"` // 1 启用 0 停用
}

// Teacher 一条人员（教师）记录。
type Teacher struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Gender     string `json:"gender"`   // male / female
	Age        int    `json:"age"`      // 0 = 未填
	Subject    string `json:"subject"`  // → dictionaries(kind='subject').name
	HasCert    int    `json:"has_cert"` // 教师资格证 0/1
	Phone      string `json:"phone"`
	Education  string `json:"education"`  // → dictionaries(kind='education').name
	University string `json:"university"` // 毕业院校
	Major      string `json:"major"`      // 专业
	Remark     string `json:"remark"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// User 系统用户（单管理员场景仅 admin 一行）。
type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"` // admin / operator
	CreatedAt   string `json:"created_at"`
}

// ---------- 字典默认数据 ----------

type dictSeed struct {
	name  string
	color string
}

// defaultSubjects 学科种子（顺序即初始 sort）。
var defaultSubjects = []dictSeed{
	{"语文", "#ef4444"},
	{"数学", "#3b82f6"},
	{"英语", "#8b5cf6"},
	{"物理", "#06b6d4"},
	{"化学", "#10b981"},
	{"生物", "#84cc16"},
	{"政治", "#f97316"},
	{"历史", "#a16207"},
	{"地理", "#14b8a6"},
	{"音乐", "#ec4899"},
	{"体育", "#f59e0b"},
	{"美术", "#6366f1"},
	{"信息技术", "#64748b"},
}

// defaultEducations 学历种子。
var defaultEducations = []dictSeed{
	{"中专", "#94a3b8"},
	{"大专", "#0ea5e9"},
	{"本科", "#2563eb"},
	{"硕士", "#7c3aed"},
	{"博士", "#db2777"},
}

// ---------- 应用 ----------

// app 应用依赖：数据库、会话、请求限流、API Key 加密密钥。
type app struct {
	db           *sql.DB
	auth         *authStore
	trustProxy   bool         // -trust-proxy：是否信任反向代理头（XFF/X-Real-IP）
	loginLimiter *rateLimiter // /api/login 限流：同IP每分钟10次，缓解密码爆破
	secret       *SecretKey   // API Key 加密密钥（AES-256-GCM）；测试/未配置时为 nil
	// siteNameMu 保护 siteName 缓存：SPA 回退注入站点名时不再查库，设置变更时失效。
	siteNameMu     sync.Mutex
	siteNameLoaded bool
	siteNameVal    string
	// extractHook 简历字段抽取的替换实现；nil 走真实模型调用（仅测试注入）。
	extractHook extractHookFunc
}

// siteName 读取缓存的站点名；未命中缓存时查库并填充（首次请求 / 设置变更后）。
// 站点名为空也缓存（loaded 标记），避免每次 SPA 回退都触发一次 DB 查询。
func (a *app) siteName() string {
	a.siteNameMu.Lock()
	loaded, v := a.siteNameLoaded, a.siteNameVal
	a.siteNameMu.Unlock()
	if loaded {
		return v
	}

	name, err := getSetting(a.db, "site_name")
	if err != nil {
		return ""
	}
	a.siteNameMu.Lock()
	a.siteNameLoaded = true
	a.siteNameVal = name
	a.siteNameMu.Unlock()
	return name
}

// invalidateSiteName 设置变更后调用：清除缓存，下次请求重新读取。
func (a *app) invalidateSiteName() {
	a.siteNameMu.Lock()
	a.siteNameLoaded = false
	a.siteNameVal = ""
	a.siteNameMu.Unlock()
}

func nowStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// ---------- 数据库打开与迁移 ----------

// openDB 打开 SQLite 数据库；单人使用固定单连接，避免偶发 database is locked。
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("设置 %s 失败: %w", pragma, err)
		}
	}
	return db, nil
}

// initDB 创建基础表（幂等）。
func initDB(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	username     TEXT NOT NULL UNIQUE,
	password     TEXT NOT NULL,
	display_name TEXT NOT NULL,
	role         TEXT NOT NULL DEFAULT 'operator',
	created_at   TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS dictionaries (
	id      INTEGER PRIMARY KEY AUTOINCREMENT,
	kind    TEXT    NOT NULL,
	name    TEXT    NOT NULL,
	color   TEXT    NOT NULL DEFAULT '#2563eb',
	sort    INTEGER NOT NULL DEFAULT 0,
	enabled INTEGER NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_dictionaries_kind_name ON dictionaries(kind, name);
CREATE TABLE IF NOT EXISTS teachers (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	name         TEXT    NOT NULL,
	gender       TEXT    NOT NULL DEFAULT 'male',
	age          INTEGER NOT NULL DEFAULT 0,
	subject      TEXT    NOT NULL DEFAULT '',
	has_cert     INTEGER NOT NULL DEFAULT 0,
	phone        TEXT    NOT NULL DEFAULT '',
	education    TEXT    NOT NULL DEFAULT '',
	university   TEXT    NOT NULL DEFAULT '',
	major        TEXT    NOT NULL DEFAULT '',
	remark       TEXT    NOT NULL DEFAULT '',
	created_at   TEXT    NOT NULL,
	updated_at   TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_teachers_subject ON teachers(subject);
CREATE INDEX IF NOT EXISTS idx_teachers_education ON teachers(education);
CREATE INDEX IF NOT EXISTS idx_teachers_name ON teachers(name);
CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS resume_drafts (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	file_name  TEXT    NOT NULL,
	file_type  TEXT    NOT NULL,
	file_size  INTEGER NOT NULL DEFAULT 0,
	page_count INTEGER NOT NULL DEFAULT 0,
	raw_text   TEXT    NOT NULL,
	payload    TEXT    NOT NULL,
	created_at TEXT    NOT NULL
);`)
	return err
}

// hasColumn 检查某表是否已存在指定列。
func hasColumn(db *sql.DB, table, col string) (bool, error) {
	var n int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name=?", table), col).Scan(&n)
	return n > 0, err
}

// migrateDB 幂等迁移：初始化字典种子、升级明文密码。
// 列结构演进沿用 hasColumn 探测模式（当前无待补列）。
func migrateDB(db *sql.DB) error {
	// 字典种子：仅当 dictionaries 表为空时写入（用户改过不重置）
	var seedCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM dictionaries").Scan(&seedCount); err != nil {
		return err
	}
	if seedCount == 0 {
		if err := seedDictionaries(db); err != nil {
			return err
		}
	}

	// 密码哈希迁移：旧版本明文密码在启动时一次性升级为 bcrypt 哈希
	if err := migratePlaintextPasswords(db); err != nil {
		return err
	}

	return nil
}

// seedDictionaries 写入学科与学历初始字典。
func seedDictionaries(db *sql.DB) error {
	total := len(defaultSubjects) + len(defaultEducations)
	log.Printf("[迁移] 初始化字典种子（%d 条）", total)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	rollback := func(e error) error {
		_ = tx.Rollback()
		return e
	}
	for i, s := range defaultSubjects {
		if _, err := tx.Exec(
			"INSERT INTO dictionaries (kind, name, color, sort, enabled) VALUES (?, ?, ?, ?, 1)",
			dictKindSubject, s.name, s.color, i); err != nil {
			return rollback(fmt.Errorf("初始化学科 %s 失败: %w", s.name, err))
		}
	}
	for i, s := range defaultEducations {
		if _, err := tx.Exec(
			"INSERT INTO dictionaries (kind, name, color, sort, enabled) VALUES (?, ?, ?, ?, 1)",
			dictKindEducation, s.name, s.color, i); err != nil {
			return rollback(fmt.Errorf("初始化学历 %s 失败: %w", s.name, err))
		}
	}
	return tx.Commit()
}

// migratePlaintextPasswords 将 users 表中残留的明文密码升级为 bcrypt。
// 判定标准：bcrypt 哈希以 $2 开头，其余视为明文（幂等：已迁移的行不再匹配）。
func migratePlaintextPasswords(db *sql.DB) error {
	rows, err := db.Query("SELECT id, password FROM users WHERE password NOT LIKE '$2%'")
	if err != nil {
		return err
	}
	type pair struct {
		id int64
		pw string
	}
	var legacy []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.id, &p.pw); err != nil {
			rows.Close()
			return err
		}
		legacy = append(legacy, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, p := range legacy {
		hash, err := hashPassword(p.pw)
		if err != nil {
			log.Printf("[迁移] 用户 %d 密码哈希失败，跳过: %v", p.id, err)
			continue
		}
		if _, err := db.Exec("UPDATE users SET password = ? WHERE id = ?", hash, p.id); err != nil {
			return fmt.Errorf("升级用户 %d 密码哈希失败: %w", p.id, err)
		}
	}
	if len(legacy) > 0 {
		log.Printf("[迁移] 已将 %d 个明文密码升级为 bcrypt 哈希", len(legacy))
	}
	return nil
}

// ---------- 字典数据访问 ----------

// allDictionaries 返回某类型的全部字典项，按 sort、id 排序。
func allDictionaries(db *sql.DB, kind string) ([]Dictionary, error) {
	rows, err := db.Query(
		"SELECT id, kind, name, color, sort, enabled FROM dictionaries WHERE kind = ? ORDER BY sort ASC, id ASC",
		kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ds []Dictionary
	for rows.Next() {
		var d Dictionary
		if err := rows.Scan(&d.ID, &d.Kind, &d.Name, &d.Color, &d.Sort, &d.Enabled); err != nil {
			return nil, err
		}
		ds = append(ds, d)
	}
	return ds, rows.Err()
}

func getDictionaryByName(db *sql.DB, kind, name string) (*Dictionary, error) {
	d := &Dictionary{}
	err := db.QueryRow(
		"SELECT id, kind, name, color, sort, enabled FROM dictionaries WHERE kind = ? AND name = ?",
		kind, name).Scan(&d.ID, &d.Kind, &d.Name, &d.Color, &d.Sort, &d.Enabled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func getDictionaryByID(db *sql.DB, id int64) (*Dictionary, error) {
	d := &Dictionary{}
	err := db.QueryRow(
		"SELECT id, kind, name, color, sort, enabled FROM dictionaries WHERE id = ?", id).
		Scan(&d.ID, &d.Kind, &d.Name, &d.Color, &d.Sort, &d.Enabled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// validDictValue 判断字典值是否存在（不校验启用状态：
// 已停用的历史值仍允许教师记录保留与编辑）。
func validDictValue(db *sql.DB, kind, name string) bool {
	if name == "" {
		return false
	}
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM dictionaries WHERE kind = ? AND name = ?", kind, name).Scan(&n)
	return n > 0
}

// countTeachersUsingDict 统计引用某字典值的教师数（删除保护用）。
func countTeachersUsingDict(db *sql.DB, kind, name string) (int, error) {
	col, ok := teacherDictColumn(kind)
	if !ok {
		return 0, fmt.Errorf("未知字典类型: %s", kind)
	}
	// 列名来自白名单函数，值走占位符
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM teachers WHERE "+col+" = ?", name).Scan(&n)
	return n, err
}

// ---------- 教师数据访问 ----------

const teacherCols = "id, name, gender, age, subject, has_cert, phone, education, university, major, remark, created_at, updated_at"

// teacherQuery 教师列表查询条件。
// Gender 为空表示不筛；HasCert 为 -1 表示不筛、0/1 表示精确匹配。
// Order 仅允许 asc/desc（进入 SQL 拼接，必须走白名单）。
type teacherQuery struct {
	keyword   string
	subject   string
	gender    string
	education string
	hasCert   int
	page      int
	size      int
	order     string
}

// teacherWhere 拼接 WHERE 子句与参数（列表与导出共用同一口径）。
func teacherWhere(q teacherQuery) (string, []any) {
	where := []string{}
	args := []any{}
	if q.subject != "" {
		where = append(where, "subject = ?")
		args = append(args, q.subject)
	}
	if q.gender != "" {
		where = append(where, "gender = ?")
		args = append(args, q.gender)
	}
	if q.education != "" {
		where = append(where, "education = ?")
		args = append(args, q.education)
	}
	if q.hasCert >= 0 {
		where = append(where, "has_cert = ?")
		args = append(args, q.hasCert)
	}
	if q.keyword != "" {
		where = append(where, "name LIKE ? ESCAPE '/'")
		args = append(args, "%"+escapeLike(q.keyword)+"%")
	}
	if len(where) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(where, " AND "), args
}

// listTeachers 按查询条件分页返回教师与总数。
func listTeachers(db *sql.DB, q teacherQuery) ([]Teacher, int, error) {
	cond, args := teacherWhere(q)

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM teachers "+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (q.page - 1) * q.size
	rows, err := db.Query("SELECT "+teacherCols+" FROM teachers "+cond+" ORDER BY id "+strings.ToUpper(q.order)+" LIMIT ? OFFSET ?",
		append(args, q.size, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	ts, err := scanTeachers(rows)
	return ts, total, err
}

// listTeachersForExport 导出用全量查询（沿用列表筛选口径，无分页上限）。
func listTeachersForExport(db *sql.DB, q teacherQuery) ([]Teacher, error) {
	cond, args := teacherWhere(q)
	rows, err := db.Query("SELECT "+teacherCols+" FROM teachers "+cond+" ORDER BY id "+strings.ToUpper(q.order), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTeachers(rows)
}

func scanTeachers(rows *sql.Rows) ([]Teacher, error) {
	var ts []Teacher
	for rows.Next() {
		var t Teacher
		if err := rows.Scan(&t.ID, &t.Name, &t.Gender, &t.Age, &t.Subject, &t.HasCert, &t.Phone,
			&t.Education, &t.University, &t.Major, &t.Remark, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, rows.Err()
}

// getTeacher 查询单条教师；不存在时返回 (nil, nil)。
func getTeacher(db *sql.DB, id int64) (*Teacher, error) {
	t := &Teacher{}
	err := db.QueryRow("SELECT "+teacherCols+" FROM teachers WHERE id = ?", id).
		Scan(&t.ID, &t.Name, &t.Gender, &t.Age, &t.Subject, &t.HasCert, &t.Phone,
			&t.Education, &t.University, &t.Major, &t.Remark, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// teacherInput 新增/编辑共用的落库字段（已通过校验）。
type teacherInput struct {
	name       string
	gender     string
	age        int
	subject    string
	hasCert    int
	phone      string
	education  string
	university string
	major      string
	remark     string
}

func createTeacher(db *sql.DB, in teacherInput) (int64, error) {
	now := nowStr()
	res, err := db.Exec(
		`INSERT INTO teachers (name, gender, age, subject, has_cert, phone, education, university, major, remark, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.name, in.gender, in.age, in.subject, in.hasCert, in.phone,
		in.education, in.university, in.major, in.remark, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateTeacher(db *sql.DB, id int64, in teacherInput) error {
	_, err := db.Exec(
		`UPDATE teachers SET name=?, gender=?, age=?, subject=?, has_cert=?, phone=?,
		 education=?, university=?, major=?, remark=?, updated_at=? WHERE id=?`,
		in.name, in.gender, in.age, in.subject, in.hasCert, in.phone,
		in.education, in.university, in.major, in.remark, nowStr(), id)
	return err
}

func deleteTeacher(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM teachers WHERE id = ?", id)
	return err
}

// batchDeleteTeachers 事务批量删除，返回实际删除条数。
func batchDeleteTeachers(db *sql.DB, ids []int64) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	res, err := tx.Exec("DELETE FROM teachers WHERE id IN ("+ph+")", args...)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, tx.Commit()
}

// escapeLike 转义 LIKE 通配符。
func escapeLike(s string) string {
	r := strings.NewReplacer("/", "//", "%", "/%", "_", "/_")
	return r.Replace(s)
}

// ---------- 设置 ----------

func getSetting(db *sql.DB, key string) (string, error) {
	var v string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func setSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

func getAllSettings(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

// ---------- 简历识别草稿 ----------

// ResumeDraft 简历识别的中间结果：只存归一化文本与抽取 JSON，不存原始文件。
type ResumeDraft struct {
	ID        int64  `json:"id"`
	FileName  string `json:"file_name"`
	FileType  string `json:"file_type"` // 'docx' | 'pdf'
	FileSize  int64  `json:"file_size"`
	PageCount int    `json:"page_count"`
	RawText   string `json:"raw_text,omitempty"`
	Payload   string `json:"payload"` // 抽取结果 JSON（fields + evidence + confidence + warnings）
	CreatedAt string `json:"created_at"`
}

const resumeDraftCols = "id, file_name, file_type, file_size, page_count, raw_text, payload, created_at"

func scanResumeDraft(scanner interface{ Scan(...any) error }) (*ResumeDraft, error) {
	d := &ResumeDraft{}
	err := scanner.Scan(&d.ID, &d.FileName, &d.FileType, &d.FileSize, &d.PageCount,
		&d.RawText, &d.Payload, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

func insertResumeDraft(db *sql.DB, d *ResumeDraft) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO resume_drafts (file_name, file_type, file_size, page_count, raw_text, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		d.FileName, d.FileType, d.FileSize, d.PageCount, d.RawText, d.Payload, nowStr())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// getResumeDraft 查询单条草稿；不存在时返回 (nil, nil)。
func getResumeDraft(db *sql.DB, id int64) (*ResumeDraft, error) {
	return scanResumeDraft(db.QueryRow("SELECT "+resumeDraftCols+" FROM resume_drafts WHERE id = ?", id))
}

// listResumeDrafts 草稿列表（按 id 倒序分页），同时返回总数。
func listResumeDrafts(db *sql.DB, page, size int) ([]ResumeDraft, int, error) {
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM resume_drafts").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(
		"SELECT "+resumeDraftCols+" FROM resume_drafts ORDER BY id DESC LIMIT ? OFFSET ?",
		size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []ResumeDraft{}
	for rows.Next() {
		d, err := scanResumeDraft(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *d)
	}
	return items, total, rows.Err()
}

func deleteResumeDraft(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM resume_drafts WHERE id = ?", id)
	return err
}

// ---------- 用户数据访问 ----------

// getUserAuth 登录用单次查询：用户信息 + 密码哈希；不存在时返回 (nil, "", nil)。
func getUserAuth(db *sql.DB, username string) (*User, string, error) {
	u := &User{}
	var pw string
	err := db.QueryRow("SELECT id, username, display_name, role, created_at, password FROM users WHERE username = ?", username).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.CreatedAt, &pw)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return u, pw, nil
}

func getUserByID(db *sql.DB, id int64) (*User, error) {
	u := &User{}
	err := db.QueryRow("SELECT id, username, display_name, role, created_at FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func updateUserPassword(db *sql.DB, id int64, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}
	_, err = db.Exec("UPDATE users SET password = ? WHERE id = ?", hash, id)
	return err
}

// setupDefaultAdmin 首次启动写入 admin 账号；已存在则不覆盖密码（幂等）。
// 返回是否本次新建（用于启动日志提示初始密码）。
func setupDefaultAdmin(db *sql.DB, password string) (bool, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count); err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	hash, err := hashPassword(password)
	if err != nil {
		return false, fmt.Errorf("密码哈希失败: %w", err)
	}
	if _, err = db.Exec(
		"INSERT OR IGNORE INTO users (username, password, display_name, role, created_at) VALUES (?, ?, ?, ?, ?)",
		"admin", hash, "管理员", "admin", nowStr()); err != nil {
		return false, err
	}
	return true, nil
}
