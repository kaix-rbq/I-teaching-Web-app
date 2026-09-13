import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import AppLayout from '@/layouts/AppLayout.vue'
import BlankLayout from '@/layouts/BlankLayout.vue'
import { setupGuards } from './guards'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    component: BlankLayout,
    children: [
      {
        path: '',
        name: 'login',
        component: () => import('@/views/LoginView.vue'),
        meta: { title: '登录', public: true }
      }
    ]
  },
  {
    path: '/',
    component: AppLayout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/DashboardView.vue'),
        meta: { title: '工作台' }
      },
      {
        path: 'courses',
        name: 'course-list',
        component: () => import('@/views/CourseListView.vue'),
        meta: { title: '课程列表' }
      },
      {
        path: 'courses/new',
        name: 'course-new',
        component: () => import('@/views/CourseFormView.vue'),
        meta: { title: '新增课程', roles: ['director'] }
      },
      {
        path: 'courses/:id',
        name: 'course-detail',
        component: () => import('@/views/CourseDetailView.vue'),
        meta: { title: '课程详情' }
      },
      {
        path: 'courses/:id/edit',
        name: 'course-edit',
        component: () => import('@/views/CourseFormView.vue'),
        meta: { title: '编辑课程', roles: ['director'] }
      },
      {
        path: 'supervision',
        name: 'supervision',
        component: () => import('@/views/SupervisionView.vue'),
        meta: { title: '督导总览', roles: ['supervisor'] }
      },
      {
        path: 'profile',
        name: 'profile',
        component: () => import('@/views/ProfileView.vue'),
        meta: { title: '个人中心' }
      }
    ]
  },
  {
    path: '/403',
    component: BlankLayout,
    children: [
      {
        path: '',
        name: 'forbidden',
        component: () => import('@/views/error/ForbiddenView.vue'),
        meta: { title: '无权访问', public: true }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    component: BlankLayout,
    children: [
      {
        path: '',
        name: 'not-found',
        component: () => import('@/views/error/NotFoundView.vue'),
        meta: { title: '页面不存在', public: true }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

setupGuards(router)

export default router
