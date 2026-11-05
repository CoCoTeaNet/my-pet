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
		auth.PUT("/auth/featured-pet", h.SetFeaturedPet)
		auth.GET("/pets", h.ListPets)
		auth.GET("/pets/:id", h.GetPet)
		auth.POST("/pets", h.CreatePet)
		auth.PUT("/pets/:id", h.UpdatePet)
		auth.DELETE("/pets/:id", h.DeletePet)
		auth.POST("/pets/:id/feeds", h.CreateFeed)
		auth.GET("/pets/:id/feeds", h.ListFeeds)
		auth.POST("/pets/:id/reminders", h.CreateReminder)
		auth.GET("/pets/:id/reminders", h.ListReminders)
		auth.POST("/pets/:id/logs", h.CreateLog)
		auth.GET("/pets/:id/logs", h.ListLogs)
		auth.POST("/reminders/:id/complete", h.CompleteReminder)
		auth.PUT("/reminders/:id", h.UpdateReminder)
		auth.DELETE("/reminders/:id", h.DeleteReminder)
		auth.DELETE("/feeds/:id", h.DeleteFeed)
		auth.DELETE("/logs/:id", h.DeleteLog)
	}
	admin := auth.Group("", middleware.RequireAdmin())
	{
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
	u := &model.User{ID: newID(), Username: "u-" + role + string(rune(time.Now().UnixNano()%1000)), Role: role}
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

func TestPetDatesAndFeaturedPet(t *testing.T) {
	h, r := setupTest(t)
	u := &model.User{ID: newID(), Username: "featured-user", Role: "member"}
	if err := h.DB.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	token := tokenFor(t, u)

	code, body := doReq(t, r, "POST", "/api/pets",
		`{"name":"团子","birth_date":"2024-03-02","adoption_date":"2024-05-01"}`, token)
	if code != 200 || !strings.Contains(body, `"adoption_date":"2024-05-01"`) {
		t.Fatalf("create with dates: %d %s", code, body)
	}
	var pet struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &pet)

	code, body = doReq(t, r, "GET", "/api/pets/"+pet.ID, "", token)
	if code != 200 || !strings.Contains(body, `"birth_date":"2024-03-02"`) ||
		!strings.Contains(body, `"adoption_date":"2024-05-01"`) {
		t.Fatalf("get dates: %d %s", code, body)
	}

	code, body = doReq(t, r, "PUT", "/api/auth/featured-pet", `{"pet_id":"`+pet.ID+`"}`, token)
	if code != 200 || !strings.Contains(body, pet.ID) {
		t.Fatalf("set featured: %d %s", code, body)
	}
	var saved model.User
	h.DB.First(&saved, "id = ?", u.ID)
	if saved.FeaturedPetID != pet.ID {
		t.Fatalf("featured not saved: %q", saved.FeaturedPetID)
	}

	other := &model.User{ID: newID(), Username: "featured-other", Role: "member"}
	h.DB.Create(other)
	if code, _ = doReq(t, r, "PUT", "/api/auth/featured-pet", `{"pet_id":"`+pet.ID+`"}`, tokenFor(t, other)); code != 404 {
		t.Fatalf("other user's pet should be 404, got %d", code)
	}

	doReq(t, r, "DELETE", "/api/pets/"+pet.ID, "", token)
	h.DB.First(&saved, "id = ?", u.ID)
	if saved.FeaturedPetID != "" {
		t.Fatalf("featured should be cleared after delete: %q", saved.FeaturedPetID)
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

	// 成员可以新增并管理自己的宠物
	code, body := doReq(t, r, "POST", "/api/pets", `{"name":"成员的猫"}`, member)
	if code != 200 {
		t.Fatalf("member create pet: %d %s", code, body)
	}
	var own struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &own)

	_, body = doReq(t, r, "POST", "/api/pets", `{"name":"管理员的狗"}`, admin)
	var others struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &others)

	// 成员访问管理员的宠物一律按不存在处理
	denied := []struct{ m, p, b string }{
		{"GET", "/api/pets/" + others.ID, ""},
		{"PUT", "/api/pets/" + others.ID, `{"name":"改名"}`},
		{"DELETE", "/api/pets/" + others.ID, ""},
		{"GET", "/api/pets/" + others.ID + "/feeds", ""},
		{"POST", "/api/pets/" + others.ID + "/feeds", `{"food_type":"dry","amount":50}`},
		{"POST", "/api/pets/" + others.ID + "/logs", `{"content":"x"}`},
		{"POST", "/api/pets/" + others.ID + "/reminders", `{"type":"驱虫","interval_days":30,"next_due_at":"2026-11-01"}`},
	}
	for _, tc := range denied {
		if code, body := doReq(t, r, tc.m, tc.p, tc.b, member); code != 404 {
			t.Errorf("member %s %s = %d, want 404 (%s)", tc.m, tc.p, code, body)
		}
	}

	if code, body := doReq(t, r, "POST", "/api/pets/"+own.ID+"/feeds", `{"food_type":"dry","amount":50}`, member); code != 200 {
		t.Errorf("member feed own pet: %d %s", code, body)
	}
	if code, body := doReq(t, r, "PUT", "/api/pets/"+own.ID, `{"name":"新名字"}`, member); code != 200 {
		t.Errorf("member update own pet: %d %s", code, body)
	}

	// 管理员专属接口仍然 403
	for _, p := range []string{"/api/users", "/api/settings"} {
		if code, body := doReq(t, r, "GET", p, "", member); code != 403 {
			t.Errorf("member GET %s = %d, want 403 (%s)", p, code, body)
		}
	}
}

func TestPetIsolation(t *testing.T) {
	_, r := setupTest(t)
	admin := loginAs(t, "admin")
	member := loginAs(t, "member")

	_, body := doReq(t, r, "POST", "/api/pets", `{"name":"管理员的狗"}`, admin)
	var adminPet struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &adminPet)
	doReq(t, r, "POST", "/api/pets", `{"name":"成员的猫"}`, member)

	code, body := doReq(t, r, "GET", "/api/pets", "", admin)
	if code != 200 || strings.Contains(body, "成员的猫") {
		t.Fatalf("admin list should only contain own pet: %d %s", code, body)
	}
	var resp struct {
		Items []model.Pet `json:"items"`
		Total int         `json:"total"`
	}
	json.Unmarshal([]byte(body), &resp)
	if resp.Total != 1 || len(resp.Items) != 1 || resp.Items[0].ID != adminPet.ID {
		t.Fatalf("admin list wrong: %+v", resp)
	}

	code, body = doReq(t, r, "GET", "/api/pets", "", member)
	if code != 200 || strings.Contains(body, "管理员的狗") || !strings.Contains(body, "成员的猫") {
		t.Fatalf("member list should only contain own pet: %d %s", code, body)
	}

	// 跨用户操作子记录同样被拦截
	var rem struct {
		ID string `json:"id"`
	}
	_, body = doReq(t, r, "POST", "/api/pets/"+adminPet.ID+"/reminders", `{"type":"疫苗","interval_days":30,"next_due_at":"2026-11-01"}`, admin)
	json.Unmarshal([]byte(body), &rem)
	if code, _ := doReq(t, r, "POST", "/api/reminders/"+rem.ID+"/complete", "", member); code != 404 {
		t.Errorf("member complete others reminder = %d, want 404", code)
	}
	if code, _ := doReq(t, r, "PUT", "/api/reminders/"+rem.ID, `{"type":"x","interval_days":1,"next_due_at":"2026-11-01"}`, member); code != 404 {
		t.Errorf("member update others reminder = %d, want 404", code)
	}
	if code, _ := doReq(t, r, "DELETE", "/api/reminders/"+rem.ID, "", member); code != 404 {
		t.Errorf("member delete others reminder = %d, want 404", code)
	}
	if code, _ := doReq(t, r, "POST", "/api/reminders/"+rem.ID+"/complete", "", admin); code != 200 {
		t.Error("owner complete reminder failed")
	}
}

func TestDeleteUserCascadesPets(t *testing.T) {
	h, r := setupTest(t)
	adminUser := &model.User{ID: newID(), Username: "theadmin", Role: "admin"}
	if err := hashPassword(adminUser, "x1234567"); err != nil {
		t.Fatal(err)
	}
	h.DB.Create(adminUser)
	adminToken := tokenFor(t, adminUser)

	_, body := doReq(t, r, "POST", "/api/users", `{"username":"spouse","password":"abc12345","role":"member"}`, adminToken)
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &created)
	memberToken := tokenFor(t, &model.User{ID: created.ID, Username: "spouse", Role: "member"})

	_, body = doReq(t, r, "POST", "/api/pets", `{"name":"成员的猫"}`, memberToken)
	var pet struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &pet)
	doReq(t, r, "POST", "/api/pets/"+pet.ID+"/feeds", `{"food_type":"dry","amount":20}`, memberToken)

	if code, _ := doReq(t, r, "DELETE", "/api/users/"+created.ID, "", adminToken); code != 200 {
		t.Fatal("delete user failed")
	}
	var pets, feeds int64
	h.DB.Model(&model.Pet{}).Where("owner_id = ?", created.ID).Count(&pets)
	h.DB.Model(&model.FeedRecord{}).Where("pet_id = ?", pet.ID).Count(&feeds)
	if pets != 0 || feeds != 0 {
		t.Fatalf("pets/feeds not cascaded: pets=%d feeds=%d", pets, feeds)
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
	var pet struct {
		ID string `json:"id"`
	}
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
