<script setup lang="ts">
import { computed, markRaw, onMounted, ref, watch, type Component as VueComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowDown,
  Bell,
  Collection,
  EditPen,
  Expand,
  Fold,
  Medal,
  Odometer,
  SwitchButton,
  User,
  UserFilled
} from '@element-plus/icons-vue'
import RoleTag from '@/components/common/RoleTag.vue'
import { useAuthStore } from '@/stores/auth'
import { useDictStore } from '@/stores/dict'
import { useSemester } from '@/composables/useSemester'
import { SEMESTERS } from '@/constants'
import { seedDemoApi } from '@/api/demo'
import { fetchNotificationsApi } from '@/api/notification'

interface MenuItem {
  index: string
  label: string
  icon: VueComponent
}

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const dict = useDictStore()
const { semester, setSemester } = useSemester()

const collapsed = ref(false)

const menuItems = computed<MenuItem[]>(() => {
  const role = auth.user?.role

  // 教师端以「教师工作台」为首页，档案页只承载质量数据。
  if (role === 'teacher') {
    return [
      { index: 'teacher-workbench', label: '工作台', icon: markRaw(Odometer) },
      { index: 'profile-quality', label: '我的质量档案', icon: markRaw(Medal) },
      { index: 'course-list', label: '我的课程', icon: markRaw(Collection) },
      { index: 'reflections', label: '教学反思', icon: markRaw(EditPen) },
      { index: 'profile', label: '个人中心', icon: markRaw(User) }
    ]
  }

  // 督导端聚焦「听评课」核心任务：工作台、课程列表、个人中心。
  if (role === 'supervisor') {
    return [
      { index: 'dashboard', label: '工作台', icon: markRaw(Odometer) },
      { index: 'course-list', label: '课程列表', icon: markRaw(Collection) },
      { index: 'profile', label: '个人中心', icon: markRaw(User) }
    ]
  }

  return [
    { index: 'dashboard', label: '质量驾驶舱', icon: markRaw(Odometer) },
    { index: 'course-list', label: '课程库', icon: markRaw(Collection) },
    { index: 'teacher-list', label: '教师画像', icon: markRaw(UserFilled) },
    { index: 'profile', label: '个人中心', icon: markRaw(User) }
  ]
})

const activeMenu = computed(() => {
  const name = String(route.name ?? '')
  if (name === 'session-evaluation') {
    return auth.role === 'supervisor' ? 'dashboard' : 'teacher-workbench'
  }
  if (name === 'profile-quality') {
    return 'profile-quality'
  }
  if (name === 'teacher-workbench') {
    return 'teacher-workbench'
  }
  if (
    name === 'course-detail' ||
    name === 'course-new' ||
    name === 'course-edit' ||
    name === 'course-improve' ||
    name === 'supervisor-course'
  ) {
    return 'course-list'
  }
  if (name === 'teacher-detail') {
    return 'teacher-list'
  }
  return name
})

const breadcrumbs = computed(() =>
  route.matched
    .filter((record) => record.meta.title)
    .map((record) => ({ title: record.meta.title as string, name: record.name as string }))
)

const avatarText = computed(() => auth.user?.name?.slice(0, 1) ?? '爱')

// 演示模式按钮：仅 DEV 且督导/主任可见（B6/N7）。
const demoLoading = ref(false)
const showDemo = computed(() => import.meta.env.DEV && (auth.role === 'supervisor' || auth.role === 'director'))

async function handleDemoSeed(): Promise<void> {
  try {
    await ElMessageBox.confirm('将清空并载入示例转写/清单等演示数据，仅本地开发库可用。是否继续？', '载入演示数据', {
      type: 'warning',
      confirmButtonText: '确认载入'
    })
  } catch {
    return
  }
  demoLoading.value = true
  try {
    const res = await seedDemoApi()
    ElMessage.success(`${res.message}：转写 ${res.transcripts} 条`)
    window.location.reload()
  } finally {
    demoLoading.value = false
  }
}

function goNotifications(): void {
  void router.push({ name: 'notifications' })
}

const unread = ref(0)
async function loadUnread(): Promise<void> {
  try {
    const page = await fetchNotificationsApi({ pageSize: 1 })
    unread.value = page.unread ?? 0
  } catch {
    unread.value = 0
  }
}

function handleSelect(index: string): void {
  if (index !== route.name) {
    void router.push({ name: index })
  }
}

async function handleCommand(command: string): Promise<void> {
  if (command === 'profile') {
    void router.push({ name: 'profile' })
    return
  }
  if (command === 'logout') {
    await auth.logout()
    void router.push({ name: 'login' })
  }
}

onMounted(() => {
  void dict.load()
  void loadUnread()
})

// 站内信未读数：路由切换后刷新（进入详情自动已读 → 红点收敛）。
watch(
  () => route.fullPath,
  () => {
    void loadUnread()
  }
)
</script>

<template>
  <div class="app-layout" :class="{ 'app-layout--collapsed': collapsed }">
    <aside class="app-layout__aside">
      <div class="app-layout__logo">
        <span class="app-layout__logo-mark">爱</span>
        <span v-show="!collapsed" class="app-layout__logo-text">爱教学</span>
      </div>

      <el-menu
        class="app-layout__menu"
        :default-active="activeMenu"
        :collapse="collapsed"
        :collapse-transition="false"
        @select="handleSelect"
      >
        <el-menu-item v-for="item in menuItems" :key="item.index" :index="item.index">
          <el-icon><component :is="item.icon" /></el-icon>
          <template #title>{{ item.label }}</template>
        </el-menu-item>
      </el-menu>

      <button class="app-layout__collapse" type="button" @click="collapsed = !collapsed">
        <el-icon><component :is="collapsed ? Expand : Fold" /></el-icon>
        <span v-show="!collapsed">收起菜单</span>
      </button>
    </aside>

    <div class="app-layout__main">
      <header class="app-layout__header">
        <el-breadcrumb separator="/">
          <el-breadcrumb-item v-for="item in breadcrumbs" :key="item.name">
            {{ item.title }}
          </el-breadcrumb-item>
        </el-breadcrumb>

        <div class="app-layout__header-side">
          <el-badge :value="unread" :hidden="unread === 0" :max="99">
            <button class="app-layout__bell" type="button" aria-label="站内信" @click="goNotifications">
              <el-icon><Bell /></el-icon>
            </button>
          </el-badge>
          <el-button v-if="showDemo" size="small" plain :loading="demoLoading" @click="handleDemoSeed">
            载入演示数据
          </el-button>
          <span class="app-layout__semester-label">学期口径</span>
          <el-select
            class="app-layout__semester"
            :model-value="semester"
            size="small"
            aria-label="学期选择"
            @update:model-value="setSemester"
          >
            <el-option v-for="item in SEMESTERS" :key="item" :label="item" :value="item" />
          </el-select>

          <el-dropdown trigger="click" @command="handleCommand">
            <span class="app-layout__user">
              <span class="app-layout__avatar">{{ avatarText }}</span>
              <span class="app-layout__name">{{ auth.user?.name }}</span>
              <RoleTag :role="auth.user?.role" />
              <el-icon class="app-layout__caret"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <el-icon><User /></el-icon>个人中心
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><SwitchButton /></el-icon>退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="app-layout__content">
        <div class="app-layout__inner">
          <router-view v-slot="{ Component }">
            <transition name="fade" mode="out-in">
              <component :is="Component" />
            </transition>
          </router-view>
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped lang="scss">
.app-layout {
  display: flex;
  min-height: 100vh;
  background-color: var(--color-bg-page);

  &__aside {
    position: fixed;
    top: 0;
    bottom: 0;
    left: 0;
    z-index: 10;
    display: flex;
    flex-direction: column;
    width: var(--sidebar-width);
    background-color: var(--color-ink);
    border-right: 1px solid rgba(255, 255, 255, 0.06);
    transition: width 0.2s ease;
  }

  &--collapsed &__aside {
    width: var(--sidebar-width-collapsed);
  }

  &__logo {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    height: var(--header-height);
    padding: 0 var(--spacing-4);
    overflow: hidden;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }

  &__logo-mark {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: #ffffff;
    background: var(--gradient-ai);
    border-radius: var(--radius-md);
  }

  &__logo-text {
    font-size: var(--font-size-xl);
    font-weight: 600;
    white-space: nowrap;
    color: var(--color-ink-text);
  }

  &__menu {
    flex: 1;
    padding: var(--spacing-2) 0;
    overflow-y: auto;
    border-right: none;

    /* Element 菜单变量深色化（官方支持的主题入口） */
    --el-menu-bg-color: transparent;
    --el-menu-text-color: var(--color-ink-text-dim);
    --el-menu-hover-bg-color: var(--color-ink-2);
    --el-menu-active-color: #ffffff;

    :deep(.el-menu-item) {
      height: 40px;
      margin: 2px var(--spacing-2);
      line-height: 40px;
      border-radius: var(--radius-md);

      &:hover {
        color: var(--color-ink-text);
      }

      &.is-active {
        font-weight: 500;
        color: #ffffff;
        background-color: var(--color-ink-3);

        &::before {
          position: absolute;
          top: 50%;
          left: 0;
          width: 3px;
          height: 20px;
          content: '';
          background-color: var(--color-primary);
          border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
          transform: translateY(-50%);
        }
      }
    }

    /* 深色侧栏内的滚动条弱化 */
    &::-webkit-scrollbar-thumb {
      background-color: rgba(255, 255, 255, 0.18);
    }
  }

  &__collapse {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    height: 48px;
    padding: 0 var(--spacing-4);
    overflow: hidden;
    font-size: var(--font-size-sm);
    color: var(--color-ink-text-dim);
    cursor: pointer;
    background: transparent;
    border: none;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    white-space: nowrap;

    &:hover {
      color: var(--color-ink-text);
    }
  }

  &__main {
    flex: 1;
    min-width: 0;
    margin-left: var(--sidebar-width);
    transition: margin-left 0.2s ease;
  }

  &--collapsed &__main {
    margin-left: var(--sidebar-width-collapsed);
  }

  &__header {
    position: sticky;
    top: 0;
    z-index: 9;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: var(--header-height);
    padding: 0 var(--content-padding);
    background-color: var(--color-bg-card);
    border-bottom: 1px solid var(--color-divider);
  }

  &__header-side {
    display: flex;
    align-items: center;
    gap: var(--spacing-4);
  }

  &__semester-label {
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
    white-space: nowrap;
  }

  &__semester {
    width: 132px;
  }

  &__bell {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    font-size: 16px;
    color: var(--color-text-secondary);
    cursor: pointer;
    background-color: var(--color-bg-page);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    transition: color var(--duration-fast) var(--ease-out-soft),
      border-color var(--duration-fast) var(--ease-out-soft),
      background-color var(--duration-fast) var(--ease-out-soft);

    &:hover {
      color: var(--color-primary);
      background-color: var(--color-primary-bg);
      border-color: var(--color-primary-hover);
    }

    &:focus-visible {
      outline: 2px solid var(--color-primary);
      outline-offset: 2px;
    }
  }

  &__user {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    cursor: pointer;
    outline: none;
  }

  &__avatar {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    font-size: var(--font-size-base);
    font-weight: 500;
    color: var(--color-bg-card);
    background-color: var(--color-primary);
    border-radius: 50%;
  }

  &__name {
    font-size: var(--font-size-base);
    font-weight: 500;
    color: var(--color-text-primary);
  }

  &__caret {
    color: var(--color-text-tertiary);
  }

  &__content {
    padding: var(--content-padding);
  }

  &__inner {
    max-width: var(--content-max-width);
    margin: 0 auto;
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
