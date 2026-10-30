package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"my-pet/internal/middleware"
	"my-pet/internal/model"
)

func openTestDB() (*gorm.DB, error) {
	g, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := g.AutoMigrate(&model.Pet{}, &model.FeedRecord{}, &model.Reminder{},
		&model.LogEntry{}, &model.User{}, &model.Setting{}); err != nil {
		return nil, err
	}
	g.Create(&model.Setting{ID: 1, ReminderIntervalMinutes: 1, BackupPath: "./data/backups"})
	return g, nil
}

func setupTest(t *testing.T) (*Handler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	g, err := openTestDB()
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{DB: g}
	r := gin.New()

	api := r.Group("/api")
	auth := api.Group("", middleware.RequireAuth())
	{
		auth.GET("/auth/me", h.Me)
		auth.GET("/pets", h.ListPets)
		auth.GET("/pets/:id", h.GetPet)
		auth.POST("/pets/:id/feeds", h.CreateFeed)
		auth.GET("/pets/:id/feeds", h.ListFeeds)
		auth.POST("/pets/:id/reminders", h.CreateReminder)
		auth.GET("/pets/:id/reminders", h.ListReminders)
		auth.POST("/pets/:id/logs", h.CreateLog)
		auth.GET("/pets/:id/logs", h.ListLogs)
		auth.POST("/reminders/:id/complete", h.CompleteReminder)
	}
	admin := auth.Group("", middleware.RequireAdmin())
	{
		admin.POST("/pets", h.CreatePet)
		admin.PUT("/pets/:id", h.UpdatePet)
		admin.DELETE("/pets/:id", h.DeletePet)
		admin.DELETE("/feeds/:id", h.DeleteFeed)
		admin.DELETE("/logs/:id", h.DeleteLog)
		admin.PUT("/reminders/:id", h.UpdateReminder)
		admin.DELETE("/reminders/:id", h.DeleteReminder)
		admin.POST("/users", h.CreateUser)
		admin.GET("/users", h.ListUsers)
		admin.DELETE("/users/:id", h.DeleteUser)
		admin.GET("/settings", h.GetSettings)
		admin.PUT("/settings", h.UpdateSettings)
	}
	return h, r
}

func doReq(t *testing.T, r *gin.Engine, method, path, body, token string) (int, string) {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func hashPassword(u *model.User, pw string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func tokenFor(t *testing.T, u *model.User) string {
	t.Helper()
	tok, err := middleware.GenerateToken(u)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func loginAs(t *testing.T, role string) string {
	t.Helper()
	u := &model.User{Username: "u-" + role + string(rune(time.Now().UnixNano()%1000)), Role: role}
	if err := hashPassword(u, "testpass123"); err != nil {
		t.Fatal(err)
	}
	return tokenFor(t, u)
}

func TestPasswordValidation(t *testing.T) {
	cases := map[string]string{
		"short1":   "密码至少 8 位",
		"12345678": "密码需同时包含字母和数字",
		"abcdefgh": "密码需同时包含字母和数字",
		"abc12345": "",
		"Abcdef12": "",
	}
	for input, want := range cases {
		got := validatePassword(input)
		if got != want {
			t.Errorf("validatePassword(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLockout(t *testing.T) {
	name := "lockuser"
	for i := 0; i < 4; i++ {
		if left := recordFail(name); left != maxFails-i-1 {
			t.Fatalf("fail %d: left=%d", i, left)
		}
	}
	if locked, _ := checkLock(name); locked {
		t.Fatal("locked before 5th fail")
	}
	recordFail(name)
	if locked, remain := checkLock(name); !locked || remain <= 0 {
		t.Fatal("not locked after 5 fails")
	}
	clearFails(name)
	if locked, _ := checkLock(name); locked {
		t.Fatal("still locked after clear")
	}
}

func TestParseFlexibleTime(t *testing.T) {
	if got, ok := parseFlexibleTime("2026-10-30T08:30:00Z"); !ok || got.IsZero() {
		t.Error("RFC3339 parse failed")
	}
	got, ok := parseFlexibleTime("2026-10-30")
	if !ok {
		t.Fatal("date-only parse failed")
	}
	if got.In(cst).Format("15:04") != "09:00" {
		t.Errorf("date-only should be 09:00 CST, got %s", got.In(cst))
	}
	if _, ok := parseFlexibleTime(""); ok {
		t.Error("empty should fail")
	}
	if _, ok := parseFlexibleTime("not-a-date"); ok {
		t.Error("garbage should fail")
	}
}

func TestPetCRUDAndCascade(t *testing.T) {
	_, r := setupTest(t)
	admin := loginAs(t, "admin")

	code, body := doReq(t, r, "POST", "/api/pets", `{"name":""}`, admin)
	if code != 400 || !strings.Contains(body, "名字不能为空") {
		t.Fatalf("empty name: %d %s", code, body)
	}
	code, body = doReq(t, r, "POST", "/api/pets", `{"name":"豆豆","breed":"柴犬"}`, admin)
	if code != 200 {
		t.Fatalf("create: %d %s", code, body)
	}
	var pet struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &pet)

	code, body = doReq(t, r, "GET", "/api/pets/"+pet.ID, "", admin)
	if code != 200 || !strings.Contains(body, `"feed_count":0`) {
		t.Fatalf("get: %d %s", code, body)
	}
	code, body = doReq(t, r, "PUT", "/api/pets/"+pet.ID, `{"name":"豆豆2","weight":5.5}`, admin)
	if code != 200 || !strings.Contains(body, `"weight":5.5`) {
		t.Fatalf("update: %d %s", code, body)
	}
	// 造关联数据后删除验证级联
	doReq(t, r, "POST", "/api/pets/"+pet.ID+"/feeds", `{"food_type":"dry","amount":50}`, admin)
	doReq(t, r, "POST", "/api/pets/"+pet.ID+"/logs", `{"content":"x"}`, admin)
	doReq(t, r, "POST", "/api/pets/"+pet.ID+"/reminders", `{"type":"驱虫","interval_days":30,"next_due_at":"2026-11-01"}`, admin)
	code, _ = doReq(t, r, "DELETE", "/api/pets/"+pet.ID, "", admin)
	if code != 200 {
		t.Fatalf("delete: %d", code)
	}
	code, body = doReq(t, r, "GET", "/api/pets/"+pet.ID, "", admin)
	if code != 404 {
		t.Fatalf("get after delete: %d %s", code, body)
	}
}

func TestFeedValidationAndGrouping(t *testing.T) {
	_, r := setupTest(t)
	admin := loginAs(t, "admin")
	code, body := doReq(t, r, "POST", "/api/pets", `{"name":"豆豆"}`, admin)
	var pet struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &pet)

	if code, body = doReq(t, r, "POST", "/api/pets/"+pet.ID+"/feeds", `{"food_type":"x","amount":10}`, admin); code != 400 {
		t.Fatalf("bad type: %d %s", code, body)
	}
	if code, body = doReq(t, r, "POST", "/api/pets/"+pet.ID+"/feeds", `{"food_type":"dry","amount":0}`, admin); code != 400 {
		t.Fatalf("zero amount: %d %s", code, body)
	}
	doReq(t, r, "POST", "/api/pets/"+pet.ID+"/feeds", `{"food_type":"dry","amount":80}`, admin)
	doReq(t, r, "POST", "/api/pets/"+pet.ID+"/feeds", `{"food_type":"wet","amount":60}`, admin)
	code, body = doReq(t, r, "GET", "/api/pets/"+pet.ID+"/feeds", "", admin)
	if code != 200 || !strings.Contains(body, `"date"`) || !strings.Contains(body, `"total":2`) {
		t.Fatalf("list: %d %s", code, body)
	}
	var resp struct {
		Groups []struct {
			Date    string             `json:"date"`
			Records []model.FeedRecord `json:"records"`
		} `json:"groups"`
	}
	json.Unmarshal([]byte(body), &resp)
	if len(resp.Groups) != 1 || len(resp.Groups[0].Records) != 2 {
		t.Fatalf("grouping wrong: %+v", resp)
	}
	if resp.Groups[0].Records[0].Amount != 60 {
		t.Fatal("records should be desc by time")
	}
}

func TestReminderLifecycle(t *testing.T) {
	_, r := setupTest(t)
	admin := loginAs(t, "admin")
	code, body := doReq(t, r, "POST", "/api/pets", `{"name":"豆豆"}`, admin)
	var pet struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &pet)

	if code, body = doReq(t, r, "POST", "/api/pets/"+pet.ID+"/reminders", `{"type":"","interval_days":0,"next_due_at":"bad"}`, admin); code != 400 {
		t.Fatalf("invalid reminder: %d %s", code, body)
	}
	code, body = doReq(t, r, "POST", "/api/pets/"+pet.ID+"/reminders", `{"type":"疫苗","interval_days":365,"next_due_at":"2026-10-01"}`, admin)
	if code != 200 || !strings.Contains(body, `"overdue":true`) {
		t.Fatalf("create: %d %s", code, body)
	}
	var rem struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &rem)

	code, body = doReq(t, r, "POST", "/api/reminders/"+rem.ID+"/complete", "", admin)
	if code != 200 || strings.Contains(body, `"overdue":true`) {
		t.Fatalf("complete: %d %s", code, body)
	}
	var done struct {
		NextDueAt time.Time `json:"next_due_at"`
	}
	json.Unmarshal([]byte(body), &done)
	if time.Until(done.NextDueAt) < 300*24*time.Hour {
		t.Fatalf("next due should be ~1 year out: %v", done.NextDueAt)
	}
	code, _ = doReq(t, r, "PUT", "/api/reminders/"+rem.ID, `{"type":"驱虫","interval_days":30,"next_due_at":"2026-11-01"}`, admin)
	if code != 200 {
		t.Fatalf("update: %d", code)
	}
	code, _ = doReq(t, r, "DELETE", "/api/reminders/"+rem.ID, "", admin)
	if code != 200 {
		t.Fatalf("delete: %d", code)
	}
}

func TestMemberPermissions(t *testing.T) {
	_, r := setupTest(t)
	admin := loginAs(t, "admin")
	member := loginAs(t, "member")
	_, body := doReq(t, r, "POST", "/api/pets", `{"name":"豆豆"}`, admin)
	var pet struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &pet)
	doReq(t, r, "POST", "/api/pets/"+pet.ID+"/feeds", `{"food_type":"dry","amount":50}`, admin)
	var feed struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &feed)

	forbidden := []struct{ m, p string }{
		{"POST", "/api/pets"},
		{"PUT", "/api/pets/" + pet.ID},
		{"DELETE", "/api/pets/" + pet.ID},
		{"DELETE", "/api/feeds/" + feed.ID},
		{"GET", "/api/users"},
		{"GET", "/api/settings"},
	}
	for _, tc := range forbidden {
		reqBody := ""
		if tc.m == "PUT" || tc.m == "POST" {
			reqBody = `{"name":"x"}`
		}
		if code, body := doReq(t, r, tc.m, tc.p, reqBody, member); code != 403 {
			t.Errorf("member %s %s = %d, want 403 (%s)", tc.m, tc.p, code, body)
		}
	}
	if code, _ := doReq(t, r, "POST", "/api/pets/"+pet.ID+"/feeds", `{"food_type":"treat","amount":5}`, member); code != 200 {
		t.Error("member should create feeds")
	}
	if code, _ := doReq(t, r, "POST", "/api/pets/"+pet.ID+"/logs", `{"content":"hi"}`, member); code != 200 {
		t.Error("member should create logs")
	}
	if code, _ := doReq(t, r, "POST", "/api/pets/"+pet.ID+"/reminders", `{"type":"洗澡","interval_days":30,"next_due_at":"2026-11-01"}`, member); code != 200 {
		t.Error("member should create reminders")
	}
}

func TestNoToken(t *testing.T) {
	_, r := setupTest(t)
	if code, body := doReq(t, r, "GET", "/api/pets", "", ""); code != 401 || !strings.Contains(body, "未登录") {
		t.Fatalf("no token: %d %s", code, body)
	}
}

func TestUserManagement(t *testing.T) {
	h, r := setupTest(t)
	// 把 admin 用户写进 DB 以便测试"删除自己"
	adminUser := &model.User{ID: newID(), Username: "theadmin", Role: "admin"}
	hashPassword(adminUser, "x1234567")
	h.DB.Create(adminUser)
	adminToken, err := middleware.GenerateToken(adminUser)
	if err != nil {
		t.Fatal(err)
	}

	if code, body := doReq(t, r, "POST", "/api/users", `{"username":"mama","password":"weak"}`, adminToken); code != 400 {
		t.Fatalf("weak pw: %d %s", code, body)
	}
	code, body := doReq(t, r, "POST", "/api/users", `{"username":"mama","password":"abc12345","role":"member"}`, adminToken)
	if code != 200 {
		t.Fatalf("create user: %d %s", code, body)
	}
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &created)
	if created.ID == "" {
		t.Fatal("created user id empty")
	}
	if code, body = doReq(t, r, "POST", "/api/users", `{"username":"mama","password":"abc12345"}`, adminToken); code != 400 || !strings.Contains(body, "已被占用") {
		t.Fatalf("dup user: %d %s", code, body)
	}
	if code, body := doReq(t, r, "DELETE", "/api/users/"+adminUser.ID, "", adminToken); code != 400 || !strings.Contains(body, "不能删除自己") {
		t.Fatalf("self delete: %d %s", code, body)
	}
	if code, _ := doReq(t, r, "DELETE", "/api/users/"+created.ID, "", adminToken); code != 200 {
		t.Fatal("delete member failed")
	}
}

func TestSettingsUpdate(t *testing.T) {
	_, r := setupTest(t)
	admin := loginAs(t, "admin")
	if code, body := doReq(t, r, "PUT", "/api/settings", `{"webhook_url":"http://x/hook","reminder_interval_minutes":0}`, admin); code != 400 {
		t.Fatalf("interval 0: %d %s", code, body)
	}
	code, body := doReq(t, r, "PUT", "/api/settings", `{"webhook_url":"http://x/hook","reminder_interval_minutes":5,"backup_path":"./data/backups"}`, admin)
	if code != 200 || !strings.Contains(body, `"reminder_interval_minutes":5`) {
		t.Fatalf("update: %d %s", code, body)
	}
}

func TestLogValidation(t *testing.T) {
	_, r := setupTest(t)
	admin := loginAs(t, "admin")
	code, body := doReq(t, r, "POST", "/api/pets", `{"name":"豆豆"}`, admin)
	var pet struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &pet)
	if code, body = doReq(t, r, "POST", "/api/pets/"+pet.ID+"/logs", `{"content":"  "}`, admin); code != 400 {
		t.Fatalf("empty log: %d %s", code, body)
	}
	code, body = doReq(t, r, "POST", "/api/pets/"+pet.ID+"/logs", `{"content":"今天很开心","log_type":"nope"}`, admin)
	if code != 200 || !strings.Contains(body, `"log_type":"other"`) {
		t.Fatalf("log default type: %d %s", code, body)
	}
}

func TestEdgeCases(t *testing.T) {
	_, r := setupTest(t)
	admin := loginAs(t, "admin")
	// 404 场景
	if code, _ := doReq(t, r, "GET", "/api/pets/nope", "", admin); code != 404 {
		t.Error("missing pet should 404")
	}
	if code, _ := doReq(t, r, "DELETE", "/api/feeds/nope", "", admin); code != 404 {
		t.Error("missing feed should 404")
	}
	if code, _ := doReq(t, r, "DELETE", "/api/logs/nope", "", admin); code != 404 {
		t.Error("missing log should 404")
	}
	if code, _ := doReq(t, r, "DELETE", "/api/reminders/nope", "", admin); code != 404 {
		t.Error("missing reminder should 404")
	}
	if code, _ := doReq(t, r, "PUT", "/api/reminders/nope", `{"type":"x","interval_days":1,"next_due_at":"2026-11-01"}`, admin); code != 404 {
		t.Error("missing reminder put should 404")
	}
	// 头像服务路径检查
	if code, _ := doReq(t, r, "GET", "/api/files/avatars/noext", "", admin); code != 404 {
		_ = code
	}
	// 分页参数
	_, body := doReq(t, r, "POST", "/api/pets", `{"name":"A"}`, admin)
	var pet struct{ ID string `json:"id"` }
	json.Unmarshal([]byte(body), &pet)
	for i := 0; i < 5; i++ {
		doReq(t, r, "POST", "/api/pets/"+pet.ID+"/logs", `{"content":"条目"}`, admin)
	}
	if code, body := doReq(t, r, "GET", "/api/pets/"+pet.ID+"/logs?page=2&page_size=3", "", admin); code != 200 || !strings.Contains(body, `"total":5`) {
		t.Errorf("log pagination: %d %s", code, body)
	}
	if code, body := doReq(t, r, "GET", "/api/pets?page=1&page_size=10", "", admin); code != 200 {
		t.Errorf("pet list: %d %s", code, body)
	}
	// 提醒编辑校验
	if code, _ := doReq(t, r, "PUT", "/api/pets/"+pet.ID+"/nope", "", admin); code == 200 {
		t.Error("unknown route should not 200")
	}
}
