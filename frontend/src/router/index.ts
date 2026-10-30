import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/Login.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/components/AppLayout.vue'),
      children: [
        { path: '', name: 'home', component: () => import('@/views/Home.vue'), meta: { title: '我的宠物' } },
        { path: 'pet/:id', name: 'pet-detail', component: () => import('@/views/PetDetail.vue'), meta: { title: '宠物详情' } },
        { path: 'pet/:id/feed', name: 'pet-feed', component: () => import('@/views/FeedPage.vue'), meta: { title: '喂养记录' } },
        { path: 'pet/:id/log', name: 'pet-log', component: () => import('@/views/LogPage.vue'), meta: { title: '日志' } },
        { path: 'pet/:id/reminder', name: 'pet-reminder', component: () => import('@/views/ReminderPage.vue'), meta: { title: '健康提醒' } },
        { path: 'settings', name: 'settings', component: () => import('@/views/Settings.vue'), meta: { title: '设置' } },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('my-pet-token')
  if (!to.meta.public && !token) {
    return { path: '/login', query: to.fullPath === '/' ? {} : { redirect: to.fullPath } }
  }
  if (to.path === '/login' && token) {
    return '/'
  }
})

export default router
