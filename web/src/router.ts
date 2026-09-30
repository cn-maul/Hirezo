import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import Layout from '@/components/Layout.vue'
import RequireAuth from '@/components/RequireAuth.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/pages/Login.vue'),
  },
  {
    path: '/',
    component: RequireAuth,
    children: [
      {
        path: '',
        component: Layout,
        children: [
          { path: '', redirect: '/teachers' },
          {
            path: 'teachers',
            name: 'teachers',
            component: () => import('@/pages/TeacherList.vue'),
          },
          {
            path: 'teachers/:id',
            name: 'teacher-detail',
            component: () => import('@/pages/TeacherDetail.vue'),
          },
          {
            path: 'resume',
            name: 'resume',
            component: () => import('@/pages/Resume.vue'),
          },
          {
            path: 'settings',
            component: () => import('@/pages/Settings/SettingsLayout.vue'),
            children: [
              { path: '', redirect: '/settings/general' },
              {
                path: 'general',
                name: 'settings-general',
                component: () => import('@/pages/Settings/General.vue'),
              },
              {
                path: 'dicts',
                name: 'settings-dicts',
                component: () => import('@/pages/Settings/Dicts.vue'),
              },
              {
                path: 'llm',
                name: 'settings-llm',
                component: () => import('@/pages/Settings/LLM.vue'),
              },
            ],
          },
        ],
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/pages/NotFound.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router