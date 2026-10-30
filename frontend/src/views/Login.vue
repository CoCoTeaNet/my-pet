<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NInput, NButton } from 'naive-ui'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const form = reactive({ username: '', password: '' })
const usernameError = ref('')
const passwordError = ref('')
const loginError = ref('')
const submitting = ref(false)

function validateUsername() {
  usernameError.value = form.username.trim() ? '' : '请输入用户名'
}

function validatePassword() {
  passwordError.value = form.password ? '' : '请输入密码'
}

async function submit() {
  validateUsername()
  validatePassword()
  loginError.value = ''
  if (usernameError.value || passwordError.value || submitting.value) return
  submitting.value = true
  try {
    const { data } = await api.post('/auth/login', {
      username: form.username.trim(),
      password: form.password,
    })
    auth.setAuth(data.token, data.user)
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
</style>
