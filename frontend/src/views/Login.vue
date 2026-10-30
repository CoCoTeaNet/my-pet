<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NInput, NButton } from 'naive-ui'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { isNative, setServerUrl } from '@/platform'
import { syncNotifications } from '@/services/notifications'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const form = reactive({ server: localStorage.getItem('my-pet-server') || '', username: '', password: '' })
const serverError = ref('')
const usernameError = ref('')
const passwordError = ref('')
const loginError = ref('')
const submitting = ref(false)
const testing = ref(false)
const testTip = ref('')

function validateServer() {
  if (!isNative) return
  serverError.value = form.server.trim() ? '' : '请输入服务器地址'
}

async function testConnection() {
  validateServer()
  testTip.value = ''
  if (serverError.value || testing.value) return
  testing.value = true
  setServerUrl(form.server)
  try {
    await api.get('/ping')
    testTip.value = '连接成功'
  } catch {
    testTip.value = '连接失败，请检查地址与网络'
  } finally {
    testing.value = false
  }
}

function validateUsername() {
  usernameError.value = form.username.trim() ? '' : '请输入用户名'
}

function validatePassword() {
  passwordError.value = form.password ? '' : '请输入密码'
}

async function submit() {
  validateServer()
  validateUsername()
  validatePassword()
  loginError.value = ''
  if (serverError.value || usernameError.value || passwordError.value || submitting.value) return
  submitting.value = true
  try {
    if (isNative) setServerUrl(form.server)
    const { data } = await api.post('/auth/login', {
      username: form.username.trim(),
      password: form.password,
    })
    auth.setAuth(data.token, data.user)
    syncNotifications().catch(() => {})
    const redirect = (route.query.redirect as string) || '/'
    router.replace(redirect)
  } catch (err) {
    loginError.value = (err as Error).message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card card">
      <div class="login-logo" aria-hidden="true">🐾</div>
      <h1 class="login-title">我的宠物</h1>
      <p class="login-sub">记录毛孩子的每一天</p>

      <div v-if="isNative" class="field-item">
        <label class="field-label">服务器地址</label>
        <div class="server-row">
          <n-input
            v-model:value="form.server"
            placeholder="如 192.168.1.10:8080"
            @blur="validateServer"
            @keyup.enter="submit"
          />
          <n-button quaternary size="small" :loading="testing" @click="testConnection">测试</n-button>
        </div>
        <div v-if="serverError" class="field-error">{{ serverError }}</div>
        <div v-else-if="testTip" class="server-tip" :class="{ ok: testTip === '连接成功' }">{{ testTip }}</div>
      </div>

      <div class="field-item">
        <label class="field-label">用户名</label>
        <n-input v-model:value="form.username" placeholder="请输入用户名" @blur="validateUsername" @keyup.enter="submit" />
        <div v-if="usernameError" class="field-error">{{ usernameError }}</div>
      </div>

      <div class="field-item">
        <label class="field-label">密码</label>
        <n-input
          v-model:value="form.password"
          type="password"
          show-password-on="click"
          placeholder="请输入密码"
          @blur="validatePassword"
          @keyup.enter="submit"
        />
        <div v-if="passwordError" class="field-error">{{ passwordError }}</div>
      </div>

      <div v-if="loginError" class="field-error login-error">{{ loginError }}</div>

      <n-button type="primary" block :loading="submitting" class="login-btn" @click="submit">登录</n-button>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  animation: page-in 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

@keyframes page-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.login-card {
  width: 100%;
  max-width: 380px;
  padding: 32px 24px;
}

.login-logo {
  text-align: center;
  font-size: 40px;
  line-height: 1;
}

.login-title {
  font-size: 20px;
  font-weight: 600;
  text-align: center;
  margin-top: 8px;
}

.login-sub {
  font-size: 14px;
  color: var(--text-sub);
  text-align: center;
  margin: 4px 0 24px;
}

.login-error {
  margin-top: 12px;
}

.login-btn {
  margin-top: 24px;
}

.server-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.server-row :deep(.n-input) {
  flex: 1;
}

.server-tip {
  font-size: 12px;
  color: var(--text-sub);
  margin-top: 4px;
}

.server-tip.ok {
  color: var(--success, #18a058);
}
</style>
