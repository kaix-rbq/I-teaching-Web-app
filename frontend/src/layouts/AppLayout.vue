<script setup lang="ts">
import { computed, markRaw, onMounted, ref, type Component as VueComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowDown,
  Collection,
  DataAnalysis,
  Expand,
  Fold,
  Odometer,
  SwitchButton,
  User
} from '@element-plus/icons-vue'
import RoleTag from '@/components/common/RoleTag.vue'
import { useAuthStore } from '@/stores/auth'
import { useDictStore } from '@/stores/dict'

interface MenuItem {
  index: string
  label: string
  icon: VueComponent
}

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const dict = useDictStore()

const collapsed = ref(false)

const menuItems = computed<MenuItem[]>(() => {
  const role = auth.user?.role
  const courseLabel =
    role === 'director' ? '课程管理' : role === 'supervisor' ? '全校课程' : '我的课程'

  const items: MenuItem[] = [
    { index: 'dashboard', label: '工作台', icon: markRaw(Odometer) },
    { index: 'course-list', label: courseLabel, icon: markRaw(Collection) }
  ]

  if (role === 'supervisor') {
    items.push({ index: 'supervision', label: '听评课管理', icon: markRaw(DataAnalysis) })
  }

  items.push({ index: 'profile', label: '个人中心', icon: markRaw(User) })
  return items
})

const activeMenu = computed(() => {
  const name = String(route.name ?? '')
  if (
    name === 'course-detail' ||
    name === 'course-new' ||
    name === 'course-edit' ||
    name === 'session-evaluation'
  ) {
    return 'course-list'
  }
  return name
})

const breadcrumbs = computed(() =>
  route.matched
    .filter((record) => record.meta.title)
    .map((record) => ({ title: record.meta.title as string, name: record.name as string }))
)

const avatarText = computed(() => auth.user?.name?.slice(0, 1) ?? '爱')

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
})
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
    background-color: var(--color-bg-card);
    border-right: 1px solid var(--color-divider);
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
    border-bottom: 1px solid var(--color-divider);
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
    color: var(--color-bg-card);
    background-color: var(--color-primary);
    border-radius: var(--radius-md);
  }

  &__logo-text {
    font-size: var(--font-size-xl);
    font-weight: 600;
    white-space: nowrap;
    color: var(--color-text-primary);
  }

  &__menu {
    flex: 1;
    padding: var(--spacing-2) 0;
    overflow-y: auto;
    border-right: none;

    :deep(.el-menu-item) {
      height: 40px;
      margin: 2px var(--spacing-2);
      line-height: 40px;
      border-radius: var(--radius-md);

      &.is-active {
        font-weight: 500;
        color: var(--color-primary);
        background-color: var(--color-primary-bg);

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
  }

  &__collapse {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    height: 48px;
    padding: 0 var(--spacing-4);
    overflow: hidden;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
    cursor: pointer;
    background: transparent;
    border: none;
    border-top: 1px solid var(--color-divider);
    white-space: nowrap;

    &:hover {
      color: var(--color-primary);
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
