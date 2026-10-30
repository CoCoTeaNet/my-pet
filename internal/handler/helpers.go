package handler

import (
	"bytes"
	"io"
	"strconv"
	"time"
)

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

var cst = time.FixedZone("Asia/Shanghai", 8*3600)

var dbFilePath = "./data/my-pet.db"

func SetDBPath(p string) { dbFilePath = p }

// parseFlexibleTime 支持 RFC3339、"2006-01-02 15:04"、"2006-01-02"（按早上 9 点）三种格式
func parseFlexibleTime(s string) (time.Time, bool) {
	s = stringsTrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04", s, cst); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("2006-01-02", s, cst); err == nil {
		return t.Add(9 * time.Hour), true
	}
	return time.Time{}, false
}

func stringsTrimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
