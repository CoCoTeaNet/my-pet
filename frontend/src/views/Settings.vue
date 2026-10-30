<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NInput, NInputNumber, NSelect, NButton, NModal, NSwitch, useDialog } from 'naive-ui'
import api from '@/api'
import type { AppSettings, NotificationLog, UserInfo } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { getServerUrl, isNative, setServerUrl } from '@/platform'
import { isNotifyEnabled, setNotifyEnabled } from '@/services/notifications'
import {
  formatRelative,
  passwordStrength,
  passwordValid,
} from '@/utils/time'

const auth = useAuthStore()
const ui = useUiStore()
const dialog = useDialog()
const router = useRouter()

const roleText = computed(() => (auth.user?.role === 'admin' ? '管理员' : '成员'))

function logout() {
  auth.clear()
  router.replace('/login')
}

/* ---------- App（仅原生环境） ---------- */
const serverAddr = ref(getServerUrl())
const notifyOn = ref(isNotifyEnabled())
const notifyBusy = ref(false)

async function onNotifyChange(v: boolean) {
  if (notifyBusy.value) return
  notifyBusy.value = true
  try {
    const count = await setNotifyEnabled(v)
    notifyOn.value = isNotifyEnabled()
    ui.showBanner(notifyOn.value ? `已开启，${count} 条提醒将推送` : '已关闭通知')
  } catch (err) {
    notifyOn.value = false
    dialog.error({ title: '开启失败', content: (err as Error).message, positiveText: '知道了' })
  } finally {
    notifyBusy.value = false
  }
}

function saveServer() {
  const prev = getServerUrl()
  const next = setServerUrl(serverAddr.value)
  if (!next) return
  serverAddr.value = next
  if (next === prev) {
    ui.showBanner('已保存')
    return
  }
  // 服务器换了，原 token 不再适用
  auth.clear()
  router.replace('/login')
}

/* ---------- 修改密码 ---------- */
const pwd = reactive({ old: '', next: '', confirm: '' })
const pwdErrors = reactive({ old: '', next: '', confirm: '' })
const pwdSubmitting = ref(false)

const strength = computed(() => passwordStrength(pwd.next))
const strengthLabel = computed(() => ['', '弱', '中', '强'][strength.value])
const strengthClass = computed(() => ['', 'strength-weak', 'strength-mid', 'strength-strong'][strength.value])

function strengthTextFor(pw: string): string {
  return ['', '弱', '中', '强'][passwordStrength(pw)]
}

function strengthClassFor(pw: string): string {
  return ['', 'strength-weak', 'strength-mid', 'strength-strong'][passwordStrength(pw)]
}

function validateOld() {
  pwdErrors.old = pwd.old ? '' : '请输入旧密码'
}

function validateNext() {
  pwdErrors.next = passwordValid(pwd.next) ? '' : '至少 8 位，且需包含字母和数字'
}

function validateConfirm() {
  pwdErrors.confirm = pwd.confirm === pwd.next ? '' : '两次输入的密码不一致'
}

async function changePwd() {
  validateOld()
  validateNext()
  validateConfirm()
  if (pwdErrors.old || pwdErrors.next || pwdErrors.confirm || pwdSubmitting.value) return
  pwdSubmitting.value = true
  try {
    await api.put('/auth/password', { old_password: pwd.old, new_password: pwd.next })
    ui.showBanner('已保存')
    pwd.old = ''
    pwd.next = ''
    pwd.confirm = ''
  } catch (err) {
    dialog.error({ title: '修改失败', content: (err as Error).message, positiveText: '知道了' })
  } finally {
    pwdSubmitting.value = false
  }
}

/* ---------- 管理员：设置 / 用户 / 通知 ---------- */
const isAdmin = computed(() => auth.isAdmin)
const settings = ref<AppSettings | null>(null)
const notifications = ref<NotificationLog[]>([])
const users = ref<UserInfo[]>([])
const settingsSubmitting = ref(false)
const testing = ref(false)
const backingUp = ref(false)

async function loadSettings() {
  const { data } = await api.get('/settings')
  settings.value = data
}

async function loadNotifications() {
  const { data } = await api.get('/settings/notifications')
  notifications.value = data.items || []
}

async function loadUsers() {
  const { data } = await api.get('/users')
  users.value = data.items || []
}

async function saveSettings() {
  if (!settings.value || settingsSubmitting.value) return
  settingsSubmitting.value = true
  try {
    const body = {
      webhook_url: settings.value.webhook_url || '',
      reminder_interval_minutes: settings.value.reminder_interval_minutes || 1,
      backup_path: settings.value.backup_path || '',
    }
    await api.put('/settings', body)
    ui.showBanner('已保存')
  } catch (err) {
    dialog.error({ title: '保存失败', content: (err as Error).message, positiveText: '知道了' })
  } finally {
    settingsSubmitting.value = false
  }
}

async function testWebhook() {
  if (testing.value) return
  testing.value = true
  try {
    await api.post('/settings/webhook/test')
    ui.showBanner('测试消息已发送')
  } catch (err) {
    dialog.error({ title: '测试失败', content: (err as Error).message, positiveText: '知道了' })
  } finally {
    testing.value = false
  }
}

async function backup() {
  if (backingUp.value) return
  backingUp.value = true
  try {
    const res = await api.post('/backup', null, { responseType: 'blob' })
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    const cd: string = res.headers['content-disposition'] || ''
    const m = cd.match(/filename\*?=(?:UTF-8''|")?([^";]+)/i)
    a.download = m ? decodeURIComponent(m[1]) : `my-pet-backup-${new Date().toISOString().slice(0, 10)}.db`
    a.click()
    URL.revokeObjectURL(url)
    ui.showBanner('备份已开始下载')
    await loadSettings()
  } catch (err) {
    dialog.error({ title: '备份失败', content: (err as Error).message, positiveText: '知道了' })
  } finally {
    backingUp.value = false
  }
}

/* ---------- 添加用户 ---------- */
const showUserModal = ref(false)
const userForm = reactive({ username: '', password: '', role: 'member' as 'admin' | 'member' })
const userErrors = reactive({ username: '', password: '' })
const userFormError = ref('')
const userSubmitting = ref(false)

const roleOptions = [
  { label: '成员', value: 'member' },
  { label: '管理员', value: 'admin' },
]

function openUserModal() {
  userForm.username = ''
  userForm.password = ''
  userForm.role = 'member'
  userErrors.username = ''
  userErrors.password = ''
  userFormError.value = ''
  showUserModal.value = true
}

function validateNewUsername() {
  userErrors.username = userForm.username.trim() ? '' : '请输入用户名'
}

function validateNewPassword() {
  userErrors.password = passwordValid(userForm.password) ? '' : '至少 8 位，且需包含字母和数字'
}

async function addUser() {
  validateNewUsername()
  validateNewPassword()
  if (userErrors.username || userErrors.password || userSubmitting.value) return
  userSubmitting.value = true
  userFormError.value = ''
  try {
    await api.post('/users', {
      username: userForm.username.trim(),
      password: userForm.password,
      role: userForm.role,
    })
    showUserModal.value = false
    ui.showBanner('已保存')
    await loadUsers()
  } catch (err) {
    userFormError.value = (err as Error).message
  } finally {
    userSubmitting.value = false
  }
}

function confirmDeleteUser(u: UserInfo) {
  dialog.warning({
    title: '删除用户',
    content: `确定删除用户"${u.username}"吗？删除后该账号无法登录。`,
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.delete(`/users/${u.id}`)
        ui.showBanner('已删除')
        await loadUsers()
      } catch (err) {
        dialog.error({ title: '删除失败', content: (err as Error).message, positiveText: '知道了' })
      }
    },
  })
}

onMounted(async () => {
  if (!isAdmin.value) return
  try {
    await Promise.all([loadSettings(), loadNotifications(), loadUsers()])
  } catch (err) {
    dialog.error({ title: '加载失败', content: (err as Error).message, positiveText: '知道了' })
  }
})
</script>

<template>
  <div>
    <!-- 个人信息 -->
    <div class="card section anim-item" style="--i: 0">
      <div class="section-title">个人信息</div>
      <div class="info-row">
        <span class="info-label">用户名</span>
        <span class="info-value">{{ auth.user?.username }}</span>
      </div>
      <div class="info-row">
        <span class="info-label">角色</span>
        <span class="info-value">{{ roleText }}</span>
      </div>
      <n-button quaternary class="section-btn logout-btn" @click="logout">退出登录</n-button>
    </div>

    <!-- App 设置（仅安卓 App） -->
    <div v-if="isNative" class="card section anim-item" style="--i: 1">
      <div class="section-title">App 设置</div>
      <div class="field-item">
        <label class="field-label">服务器地址</label>
        <div class="server-row">
          <n-input v-model:value="serverAddr" placeholder="如 192.168.1.10:8080" />
          <n-button size="small" @click="saveServer">保存</n-button>
        </div>
      </div>
      <div class="switch-row">
        <div>
          <div class="switch-label">到期提醒通知</div>
          <div class="switch-tip">按健康提醒的到期时间推送本机通知</div>
        </div>
        <n-switch :value="notifyOn" :loading="notifyBusy" @update:value="onNotifyChange" />
      </div>
    </div>

    <!-- 修改密码 -->
    <div class="card section anim-item" style="--i: 1">
      <div class="section-title">修改密码</div>
      <div class="field-item">
        <label class="field-label">旧密码</label>
        <n-input v-model:value="pwd.old" type="password" show-password-on="click" placeholder="请输入旧密码" @blur="validateOld" />
        <div v-if="pwdErrors.old" class="field-error">{{ pwdErrors.old }}</div>
      </div>
      <div class="field-item">
        <label class="field-label">新密码</label>
        <n-input v-model:value="pwd.next" type="password" show-password-on="click" placeholder="至少 8 位，包含字母和数字" @blur="validateNext" />
        <div v-if="strength" class="strength-row">
          <span class="strength-bar" :class="strengthClass"></span>
          <span class="strength-label" :class="strengthClass">{{ strengthLabel }}</span>
        </div>
        <div v-if="pwdErrors.next" class="field-error">{{ pwdErrors.next }}</div>
      </div>
      <div class="field-item">
        <label class="field-label">确认新密码</label>
        <n-input v-model:value="pwd.confirm" type="password" show-password-on="click" placeholder="再次输入新密码" @blur="validateConfirm" />
        <div v-if="pwdErrors.confirm" class="field-error">{{ pwdErrors.confirm }}</div>
      </div>
      <n-button type="primary" :loading="pwdSubmitting" class="section-btn" @click="changePwd">保存</n-button>
    </div>

    <template v-if="isAdmin">
      <!-- 用户管理 -->
      <div class="card section anim-item" style="--i: 2">
        <div class="section-head">
          <div class="section-title">用户管理</div>
          <n-button size="small" @click="openUserModal">添加用户</n-button>
        </div>
        <div v-for="u in users" :key="u.id" class="user-row">
          <span class="info-value">{{ u.username }}</span>
          <span class="user-role">{{ u.role === 'admin' ? '管理员' : '成员' }}</span>
          <button
            v-if="u.id !== auth.user?.id"
            class="text-btn danger"
            @click="confirmDeleteUser(u)"
          >
            删除
          </button>
        </div>
      </div>

      <!-- 通知设置 -->
      <div class="card section anim-item" style="--i: 3">
        <div class="section-title">通知设置</div>
        <div class="field-item">
          <label class="field-label">Webhook 地址</label>
          <n-input v-if="settings" v-model:value="settings.webhook_url" placeholder="https://example.com/webhook" />
        </div>
        <n-button :loading="testing" class="section-btn" @click="testWebhook">发送测试消息</n-button>

        <div v-if="notifications.length" class="notif-list">
          <div class="notif-title">最近的通知</div>
          <div v-for="(n, i) in notifications" :key="i" class="notif-row">
            <span class="notif-dot" :class="{ fail: !n.success }"></span>
            <span class="notif-time">{{ formatRelative(n.time, true) }}</span>
            <span class="notif-msg">{{ n.message }}</span>
          </div>
        </div>
      </div>

      <!-- 系统配置 -->
      <div class="card section anim-item" style="--i: 4">
        <div class="section-title">系统配置</div>
        <div class="field-item">
          <label class="field-label">提醒检查间隔（分钟）</label>
          <n-input-number v-if="settings" v-model:value="settings.reminder_interval_minutes" :min="1" :max="1440" style="width: 100%" />
        </div>
        <div class="field-item">
          <label class="field-label">备份路径</label>
          <n-input v-if="settings" v-model:value="settings.backup_path" placeholder="如：./data/backups" />
        </div>
        <div class="backup-row">
          <n-button type="primary" :loading="backingUp" @click="backup">立即备份</n-button>
          <span class="backup-time">
            上次备份：{{ settings?.last_backup_at ? formatRelative(settings.last_backup_at, true) : '从未备份' }}
          </span>
        </div>
        <n-button :loading="settingsSubmitting" class="section-btn" @click="saveSettings">保存配置</n-button>
      </div>

      <!-- 添加用户弹窗 -->
      <n-modal
        v-model:show="showUserModal"
        preset="card"
        title="添加用户"
        :style="{ width: 'calc(100vw - 32px)', maxWidth: '420px', borderRadius: '12px' }"
        :bordered="false"
      >
        <div class="field-item">
          <label class="field-label">用户名</label>
          <n-input v-model:value="userForm.username" placeholder="请输入用户名" :maxlength="30" @blur="validateNewUsername" />
          <div v-if="userErrors.username" class="field-error">{{ userErrors.username }}</div>
        </div>
        <div class="field-item">
          <label class="field-label">密码</label>
          <n-input v-model:value="userForm.password" type="password" show-password-on="click" placeholder="至少 8 位，包含字母和数字" @blur="validateNewPassword" />
          <div v-if="passwordStrength(userForm.password)" class="strength-row">
            <span class="strength-bar" :class="strengthClassFor(userForm.password)"></span>
            <span class="strength-label" :class="strengthClassFor(userForm.password)">{{ strengthTextFor(userForm.password) }}</span>
          </div>
          <div v-if="userErrors.password" class="field-error">{{ userErrors.password }}</div>
        </div>
        <div class="field-item">
          <label class="field-label">角色</label>
          <n-select v-model:value="userForm.role" :options="roleOptions" />
        </div>
        <div v-if="userFormError" class="field-error">{{ userFormError }}</div>
        <n-button type="primary" block :loading="userSubmitting" class="section-btn" @click="addUser">保存</n-button>
      </n-modal>
    </template>
  </div>
</template>

<style scoped>
.section + .section {
  margin-top: 16px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 12px;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-head .section-title {
  margin-bottom: 0;
}

.section-head + .user-row {
  margin-top: 8px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  font-size: 14px;
}

.info-label {
  color: var(--text-sub);
}

.info-value {
  color: var(--text);
}

.strength-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}

.strength-bar {
  height: 4px;
  border-radius: 2px;
  width: 120px;
  background: #ececec;
  position: relative;
  overflow: hidden;
}

.strength-bar::after {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  border-radius: 2px;
}

.strength-weak::after {
  width: 33%;
  background: var(--danger);
}

.strength-mid::after {
  width: 66%;
  background: var(--primary);
}

.strength-strong::after {
  width: 100%;
  background: var(--success);
}

.strength-label {
  font-size: 12px;
}

.strength-label.strength-weak {
  color: var(--danger);
}

.strength-label.strength-mid {
  color: var(--primary);
}

.strength-label.strength-strong {
  color: var(--success);
}

.section-btn {
  margin-top: 16px;
}

.logout-btn {
  color: #999;
  width: 100%;
}

.server-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.server-row :deep(.n-input) {
  flex: 1;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 0 2px;
}

.switch-label {
  font-size: 14px;
  color: var(--text);
}

.switch-tip {
  font-size: 12px;
  color: var(--text-sub);
  margin-top: 2px;
}

.user-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid #f7f7f7;
  font-size: 14px;
}

.user-row:last-child {
  border-bottom: none;
}

.user-role {
  flex: 1;
  font-size: 12px;
  color: var(--text-sub);
}

.notif-list {
  margin-top: 20px;
  border-top: 1px solid #f5f5f5;
  padding-top: 12px;
}

.notif-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
}

.notif-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 12px;
}

.notif-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--success);
  flex-shrink: 0;
}

.notif-dot.fail {
  background: var(--danger);
}

.notif-time {
  color: var(--text-sub);
  flex-shrink: 0;
}

.notif-msg {
  color: var(--text-mid);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.backup-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 16px;
}

.backup-time {
  font-size: 12px;
  color: var(--text-sub);
}
</style>
