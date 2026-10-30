import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'net.cocoteanet.mypet',
  appName: '我的宠物',
  webDir: 'dist',
  server: {
    // NAS 上是 http 服务，用 http scheme 避免 https 页面请求 http 接口被混合内容拦截
    androidScheme: 'http',
    cleartext: true,
  },
}

export default config
