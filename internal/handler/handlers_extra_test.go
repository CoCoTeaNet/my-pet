package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"my-pet/internal/middleware"
	"my-pet/internal/model"
)

const testPNG = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89"

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	g, err := openTestDB()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func buildFullEngine(h *Handler) *gin.Engine {
	r := gin.New()
	api := r.Group("/api")
	api.POST("/auth/login", h.Login)
	api.GET("/files/avatars/:name", h.ServeAvatar)
	auth := api.Group("", middleware.RequireAuth())
	{
		auth.GET("/auth/me", h.Me)
		auth.PUT("/auth/password", h.ChangePassword)
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
		admin.POST("/pets/:id/avatar", h.UploadAvatar)
		admin.DELETE("/feeds/:id", h.DeleteFeed)
		admin.DELETE("/logs/:id", h.DeleteLog)
		admin.PUT("/reminders/:id", h.UpdateReminder)
		admin.DELETE("/reminders/:id", h.DeleteReminder)
		admin.POST("/users", h.CreateUser)
		admin.GET("/users", h.ListUsers)
		admin.DELETE("/users/:id", h.DeleteUser)
		admin.GET("/settings", h.GetSettings)
		admin.PUT("/settings", h.UpdateSettings)
		admin.POST("/settings/webhook/test", h.TestWebhook)
		admin.GET("/settings/notifications", h.ListNotifications)
		admin.POST("/backup", h.Backup)
	}
	return r
}

func setupFull(t *testing.T) (*Handler, *gin.Engine, *model.User, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := &Handler{DB: newTestDB(t)}

	tmp := t.TempDir()
	oldAlerts, oldAvatar, oldDB := alertsFile, avatarDir, dbFilePath
	alertsFile = filepath.Join(tmp, "alerts.log")
	avatarDir = filepath.Join(tmp, "avatars")
	dbFilePath = filepath.Join(tmp, "test.db")
	if err := os.WriteFile(dbFilePath, []byte("SQLite format 3\x00padding"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		alertsFile, avatarDir, dbFilePath = oldAlerts, oldAvatar, oldDB
	})

	u := &model.User{Username: "tester", Role: "admin"}
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass1234"), bcrypt.DefaultCost)
	u.PasswordHash = string(hash)
	if err := h.DB.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	token := tokenFor(t, u)
	return h, buildFullEngine(h), u, token
}

func TestLoginFlow(t *testing.T) {
	h, r, u, _ := setupFull(t)
	_ = h
	_ = u

	code, body := doReq(t, r, "POST", "/api/auth/login", `{"username":"nouser","password":"pass1234"}`, "")
	if code != 401 || !strings.Contains(body, "还可尝试 4 次") {
		t.Fatalf("unknown user: %d %s", code, body)
	}
	for i := 0; i < 4; i++ {
		doReq(t, r, "POST", "/api/auth/login", `{"username":"nouser","password":"x"}`, "")
	}
	code, body = doReq(t, r, "POST", "/api/auth/login", `{"username":"nouser","password":"x"}`, "")
	if code != http.StatusTooManyRequests || !strings.Contains(body, "账号已锁定") {
		t.Fatalf("locked: %d %s", code, body)
	}
	// 锁定期间即使正确密码也被拒
	clearFails("nouser")

	code, body = doReq(t, r, "POST", "/api/auth/login", `{"username":"tester","password":"pass1234"}`, "")
	if code != 200 || !strings.Contains(body, `"token"`) {
		t.Fatalf("login ok: %d %s", code, body)
	}
	code, body = doReq(t, r, "POST", "/api/auth/login", `{"username":"tester","password":"bad"}`, "")
	if code != 401 || !strings.Contains(body, "还可尝试 4 次") {
		t.Fatalf("wrong pw: %d %s", code, body)
	}
	code, body = doReq(t, r, "POST", "/api/auth/login", `{"username":"tester","password":""}`, "")
	if code != 400 {
		t.Fatalf("empty fields: %d %s", code, body)
	}
}

func TestChangePasswordAndRelogin(t *testing.T) {
	_, r, _, token := setupFull(t)
	if code, body := doReq(t, r, "PUT", "/api/auth/password", `{"old_password":"bad","new_password":"newpass99"}`, token); code != 400 || !strings.Contains(body, "原密码不正确") {
		t.Fatalf("wrong old: %d %s", code, body)
	}
	if code, body := doReq(t, r, "PUT", "/api/auth/password", `{"old_password":"pass1234","new_password":"12345678"}`, token); code != 400 {
		t.Fatalf("weak new: %d %s", code, body)
	}
	if code, _ := doReq(t, r, "PUT", "/api/auth/password", `{"old_password":"pass1234","new_password":"newpass99"}`, token); code != 200 {
		t.Fatal("change failed")
	}
	if code, _ := doReq(t, r, "GET", "/api/auth/me", "", token); code != 200 {
		t.Fatal("me failed")
	}
}

func TestAvatarUploadAndServe(t *testing.T) {
	_, r, _, token := setupFull(t)
	_, body := doReq(t, r, "POST", "/api/pets", `{"name":"豆豆"}`, token)
	var pet struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &pet)

	// 非法扩展名
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "evil.exe")
	fw.Write([]byte("MZ"))
	mw.Close()
	req := httptest.NewRequest("POST", "/api/pets/"+pet.ID+"/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "仅支持") {
		t.Fatalf("bad ext: %d %s", w.Code, w.Body.String())
	}

	// 合法 PNG
	upload := func(tok string) (*httptest.ResponseRecorder, string) {
		var b bytes.Buffer
		m := multipart.NewWriter(&b)
		f, _ := m.CreateFormFile("file", "a.png")
		f.Write([]byte(testPNG))
		m.Close()
		rq := httptest.NewRequest("POST", "/api/pets/"+pet.ID+"/avatar", &b)
		rq.Header.Set("Content-Type", m.FormDataContentType())
		rq.Header.Set("Authorization", "Bearer "+tok)
		wr := httptest.NewRecorder()
		r.ServeHTTP(wr, rq)
		return wr, b.String()
	}
	w, _ = upload(token)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "/api/files/avatars/") {
		t.Fatalf("upload: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		AvatarURL string `json:"avatar_url"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// 成员不可上传头像
	member := loginAs(t, "member")
	w, _ = upload(member)
	if w.Code != 403 {
		t.Fatalf("member avatar: %d", w.Code)
	}

	// 静态访问
	req = httptest.NewRequest("GET", resp.AvatarURL, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Type"), "image/png") {
		t.Fatalf("serve avatar: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	// 路径穿越
	req = httptest.NewRequest("GET", "/api/files/avatars/..%2F..%2Ftest.db", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == 200 {
		t.Fatal("path traversal must fail")
	}
}

func TestNotificationsAndBackup(t *testing.T) {
	_, r, _, token := setupFull(t)

	if code, body := doReq(t, r, "PUT", "/api/settings", `{"webhook_url":"","reminder_interval_minutes":2,"backup_path":"./data/backups"}`, token); code != 200 {
		t.Fatalf("settings: %d %s", code, body)
	}
	AppendAlert("测试消息一", true)
	AppendAlert("测试消息二", false)
	code, body := doReq(t, r, "GET", "/api/settings/notifications", "", token)
	if code != 200 || !strings.Contains(body, "测试消息一") {
		t.Fatalf("notifications: %d %s", code, body)
	}
	// Webhook 未配置时测试按钮报具体原因
	if code, body := doReq(t, r, "POST", "/api/settings/webhook/test", "", token); code != 400 || !strings.Contains(body, "请先填写") {
		t.Fatalf("test webhook no url: %d %s", code, body)
	}

	tmp := t.TempDir()
	code, body = doReq(t, r, "PUT", "/api/settings", `{"webhook_url":"","reminder_interval_minutes":1,"backup_path":"`+filepath.ToSlash(tmp)+`"}`, token)
	if code != 200 {
		t.Fatalf("set backup path: %d %s", code, body)
	}
	req := httptest.NewRequest("POST", "/api/backup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Disposition"), "my-pet-backup-") {
		t.Fatalf("backup: %d %s %s", w.Code, w.Body.String(), w.Header().Get("Content-Disposition"))
	}
	if !strings.HasPrefix(string(w.Body.Bytes()), "SQLite format 3") {
		t.Fatal("backup content should be the db file copy")
	}
}
