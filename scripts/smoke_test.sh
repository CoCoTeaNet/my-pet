#!/bin/bash
# 冒烟测试脚本（自测用，可删除）
BASE="http://localhost:18080/api"
PASS=0; FAIL=0
ck() {
  if echo "$3" | grep -qF "$2"; then PASS=$((PASS+1)); echo "PASS: $1";
  else FAIL=$((FAIL+1)); echo "FAIL: $1 | 期望含[$2] 实际: $3"; fi
}
TOKEN=""
AUTH() { [ -n "$TOKEN" ] && echo "-H|Authorization: Bearer $TOKEN"; }
# req METHOD PATH [JSON] [TOKEN]
req() {
  local m=$1 p=$2 d=$3 t=$4
  if [ -n "$d" ]; then
    printf '%s' "$d" | curl -s -X "$m" -H "Content-Type: application/json" \
      ${t:+-H "Authorization: Bearer $t"} --data-binary @- "$BASE$p"
  else
    curl -s -X "$m" ${t:+-H "Authorization: Bearer $t"} "$BASE$p"
  fi
}
J() { local m=$1 p=$2 d=$3; req "$m" "$p" "$d" "$TOKEN"; }

# 1. 登录
R=$(req POST /auth/login '{"username":"admin","password":"admin123"}')
ck "管理员登录" '"role":"admin"' "$R"
TOKEN=$(echo "$R" | sed 's/.*"token":"\([^"]*\)".*/\1/')
R=$(J GET /auth/me); ck "获取当前用户" '"username":"admin"' "$R"

# 2. 宠物 CRUD
R=$(J POST /pets '{"name":""}'); ck "创建宠物-名字为空被拒" "名字不能为空" "$R"
R=$(J POST /pets '{"name":"豆豆","breed":"柴犬","gender":"male","birth_date":"2022-05-01","weight":12.5,"notes":"活泼"}')
ck "创建宠物" '"name":"豆豆"' "$R"
PET=$(echo "$R" | sed 's/.*"id":"\([^"]*\)".*/\1/')
R=$(J GET /pets/$PET); ck "获取宠物详情(含统计)" '"feed_count":0' "$R"
R=$(J PUT /pets/$PET '{"name":"豆豆","breed":"柴犬","gender":"male","birth_date":"2022-05-01","weight":13,"notes":""}')
ck "更新宠物" '"weight":13' "$R"
R=$(J GET /pets); ck "宠物列表" '"total":1' "$R"

# 3. 头像
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB\x60\x82' > /tmp/test.png
R=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" -F "file=@/tmp/test.png" $BASE/pets/$PET/avatar)
ck "上传头像" '"avatar_url":"/api/files/avatars/' "$R"
AV=$(echo "$R" | sed 's/.*"avatar_url":"\([^"]*\)".*/\1/')
R=$(curl -s -o /dev/null -w "%{http_code}:%{content_type}" "http://localhost:18080$AV")
ck "访问头像文件" "200:image/png" "$R"

# 4. 喂养
R=$(J POST /pets/$PET/feeds '{"food_type":"snack","amount":50}'); ck "喂养-非法类型被拒" "干粮、湿粮或零食" "$R"
R=$(J POST /pets/$PET/feeds '{"food_type":"dry","amount":0}'); ck "喂养-分量为0被拒" "大于 0" "$R"
R=$(J POST /pets/$PET/feeds '{"food_type":"dry","amount":80}'); ck "创建喂养记录" '"food_type":"dry"' "$R"
FEED=$(echo "$R" | sed 's/.*"id":"\([^"]*\)".*/\1/')
R=$(J POST /pets/$PET/feeds '{"food_type":"wet","amount":60}')
R=$(J GET /pets/$PET/feeds); ck "喂养列表按日期分组" '{"date"' "$R"
R=$(J DELETE /feeds/$FEED); ck "删除喂养记录" '"message":"ok"' "$R"

# 5. 提醒
TOMORROW=$(date -d "+1 day" +%Y-%m-%d)
R=$(J POST /pets/$PET/reminders "{\"type\":\"驱虫\",\"interval_days\":30,\"next_due_at\":\"$TOMORROW\",\"notes\":\"体外驱虫\"}")
ck "创建提醒" '"type":"驱虫"' "$R"
REM=$(echo "$R" | sed 's/.*"id":"\([^"]*\)".*/\1/')
R=$(J GET /pets/$PET/reminders); ck "提醒列表" '"items":' "$R"
R=$(J POST /reminders/$REM/complete); ck "完成提醒(周期推进)" '"last_triggered_at"' "$R"
R=$(J PUT /reminders/$REM "{\"type\":\"疫苗\",\"interval_days\":90,\"next_due_at\":\"$TOMORROW\",\"notes\":\"\"}")
ck "编辑提醒" '"type":"疫苗"' "$R"

# 6. 日志
R=$(J POST /pets/$PET/logs '{"content":""}'); ck "日志-空内容被拒" "不能为空" "$R"
R=$(J POST /pets/$PET/logs '{"content":"今天遛弯很开心","log_type":"behavior"}')
ck "创建日志" '"log_type":"behavior"' "$R"
LOG=$(echo "$R" | sed 's/.*"id":"\([^"]*\)".*/\1/')
R=$(J GET /pets/$PET/logs?page=1\&page_size=50); ck "日志列表" '"total":1' "$R"

# 7. 成员权限
R=$(J POST /users '{"username":"mama","password":"weak"}'); ck "创建用户-弱密码被拒" "至少 8 位" "$R"
R=$(J POST /users '{"username":"mama","password":"abc12345","role":"member"}')
ck "创建成员" '"role":"member"' "$R"
R=$(req POST /auth/login '{"username":"mama","password":"abc12345"}')
MTOKEN=$(echo "$R" | sed 's/.*"token":"\([^"]*\)".*/\1/')
M() { local m=$1 p=$2 d=$3; req "$m" "$p" "$d" "$MTOKEN"; }
[ -n "$MTOKEN" ] && echo "成员登录 OK" || { echo "成员登录失败: $R"; exit 1; }
R=$(M DELETE /pets/$PET); ck "成员删宠物被拒" "仅管理员" "$R"
R=$(M PUT /pets/$PET '{"name":"x"}'); ck "成员改宠物被拒" "仅管理员" "$R"
R=$(M DELETE /feeds/$FEED); ck "成员删喂养被拒" "仅管理员" "$R"
R=$(M DELETE /logs/$LOG); ck "成员删日志被拒" "仅管理员" "$R"
R=$(M DELETE /reminders/$REM); ck "成员删提醒被拒" "仅管理员" "$R"
R=$(M POST /pets/$PET/feeds '{"food_type":"treat","amount":10}'); ck "成员可添加喂养" '"food_type":"treat"' "$R"
R=$(M POST /pets/$PET/logs '{"content":"成员写的日志","log_type":"health"}'); ck "成员可添加日志" "成员写的日志" "$R"
R=$(M POST /reminders/$REM/complete); ck "成员可完成提醒" '"next_due_at"' "$R"
R=$(M POST /pets '{"name":"偷建"}'); ck "成员建宠物被拒" "仅管理员" "$R"
R=$(M GET /users); ck "成员看用户列表被拒" "仅管理员" "$R"
R=$(M GET /settings); ck "成员看设置被拒" "仅管理员" "$R"

# 8. 改密码
R=$(M PUT /auth/password '{"old_password":"abc12345","new_password":"short"}'); ck "改密码-弱密码被拒" "至少 8 位" "$R"
R=$(M PUT /auth/password '{"old_password":"wrong","new_password":"newpass123"}'); ck "改密码-原密码错误" "原密码不正确" "$R"
R=$(M PUT /auth/password '{"old_password":"abc12345","new_password":"newpass123"}'); ck "改密码成功" '"message":"ok"' "$R"
R=$(req POST /auth/login '{"username":"mama","password":"newpass123"}')
ck "新密码可登录" '"token"' "$R"

# 9. 登录锁定（对 mama）
for i in 1 2 3 4 5; do
  req POST /auth/login '{"username":"mama","password":"badbadbad"}' > /dev/null
done
R=$(req POST /auth/login '{"username":"mama","password":"badbadbad"}')
ck "锁定提示" "账号已锁定" "$R"
R=$(req POST /auth/login '{"username":"mama","password":"newpass123"}')
ck "锁定期间正确密码也被拒" "账号已锁定" "$R"

# 10. 设置与通知
R=$(J PUT /settings '{"webhook_url":"http://localhost:19999/hook","reminder_interval_minutes":1,"backup_path":"./data/backups"}')
ck "更新设置" '"webhook_url":"http://localhost:19999/hook"' "$R"
R=$(J POST /settings/webhook/test); ck "Webhook 测试-不可达报具体错误" "Webhook 发送失败" "$R"
R=$(J GET /settings/notifications); ck "通知日志" '"items":' "$R"

# 11. 备份
CODE=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" -o /tmp/backup.db -w "%{http_code}" $BASE/backup)
ck "备份下载" "200" "$CODE"
R=$(head -c 15 /tmp/backup.db)
ck "备份文件为 SQLite" "SQLite" "$R"

# 12. 未登录与 SPA
R=$(req GET /pets); ck "未登录被拒" "未登录" "$R"
R=$(curl -s http://localhost:18080/nonexist-route); ck "SPA fallback 返回 HTML" "<!DOCTYPE html>" "$R"

# 13. 删除宠物级联
R=$(J GET /pets/$PET); ck "删除前统计" '"feed_count":2' "$R"
R=$(J DELETE /pets/$PET); ck "删除宠物" '"message":"ok"' "$R"
R=$(J GET /pets); ck "删除后列表为空" '"total":0' "$R"

# 14. 并发写入
for i in $(seq 1 10); do
  J POST /pets "{\"name\":\"并发$i\"}" &
done
wait
R=$(J GET /pets)
TOTAL=$(echo "$R" | sed 's/.*"total":\([0-9]*\).*/\1/')
ck "并发写入 10 条全部成功" "10" "$TOTAL"

# 15. 清理并发测试数据与成员
for id in $(J GET /pets | grep -o '"id":"[^"]*"' | cut -d'"' -f4); do
  J DELETE /pets/$id > /dev/null
done
MAMA_ID=$(J GET /users | sed 's/.*"id":"\([^"]*\)","username":"mama".*/\1/')
R=$(J DELETE /users/$MAMA_ID); ck "删除成员" '"message":"ok"' "$R"
R=$(J GET /users); ck "仅剩管理员" '"items":[{"id"' "$R"

echo "===================="
echo "通过: $PASS  失败: $FAIL"
