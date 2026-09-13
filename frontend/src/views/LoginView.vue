<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { ROLES } from '@/constants'
import type { Role } from '@/types/user'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  username: '',
  password: ''
})

const rules: FormRules<typeof form> = {
  username: [{ required: true, message: '请输入账号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const showDemo = import.meta.env.DEV

async function redirectAfterLogin(): Promise<void> {
  const redirect = route.query.redirect
  if (typeof redirect === 'string' && redirect) {
    await router.replace(redirect)
  } else {
    await router.replace({ name: 'dashboard' })
  }
}

async function handleLogin(): Promise<void> {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await auth.login({ username: form.username, password: form.password })
    ElMessage.success('登录成功')
    await redirectAfterLogin()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    loading.value = false
  }
}

async function handleDemoLogin(role: Role): Promise<void> {
  loading.value = true
  try {
    await auth.login({ username: role, password: '123456' })
    ElMessage.success('已进入演示账号')
    await redirectAfterLogin()
  } catch {
    // 错误提示由 api/http.ts 拦截器统一处理
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login">
    <section class="login__brand">
      <div class="login__brand-inner">
        <div class="login__logo">
          <span class="login__logo-mark">爱</span>
          <span class="login__logo-text">爱教学</span>
        </div>
        <h1 class="login__slogan">让教学质量持续可测</h1>
        <p class="login__desc">
          面向高校的教学质量全链路数字化管理平台，覆盖课程查询、课堂评估与教学改进。
        </p>
        <ul class="login__roles">
          <li v-for="role in ROLES" :key="role.value" class="login__role">
            <span class="login__role-dot" :style="{ backgroundColor: role.color }" />
            <span>{{ role.label }} · {{ role.value }}</span>
          </li>
        </ul>
      </div>
    </section>

    <section class="login__form-side">
      <div class="login__form-box">
        <h2 class="login__welcome">欢迎登录</h2>
        <p class="login__form-tip">请使用统一账号按角色登录</p>

        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          size="large"
          @keyup.enter="handleLogin"
        >
          <el-form-item prop="username">
            <el-input v-model="form.username" placeholder="请输入账号" clearable />
          </el-form-item>
          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              show-password
            />
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              class="login__submit"
              :loading="loading"
              @click="handleLogin"
            >
              登录
            </el-button>
          </el-form-item>
        </el-form>

        <template v-if="showDemo">
          <el-divider>演示角色快捷入口</el-divider>
          <div class="login__demo">
            <el-button
              v-for="role in ROLES"
              :key="role.value"
              :disabled="loading"
              @click="handleDemoLogin(role.value)"
            >
              以{{ role.shortLabel }}身份体验
            </el-button>
          </div>
          <p class="login__demo-tip">演示账号密码统一为 123456</p>
        </template>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.login {
  display: flex;
  min-height: 100vh;

  &__brand {
    display: flex;
    flex: 0 0 60%;
    align-items: center;
    justify-content: center;
    padding: var(--spacing-8);
    color: var(--color-bg-card);
    background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-supervisor) 100%);
  }

  &__brand-inner {
    max-width: 480px;
  }

  &__logo {
    display: flex;
    align-items: center;
    gap: var(--spacing-3);
  }

  &__logo-mark {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    font-size: var(--font-size-2xl);
    font-weight: 600;
    color: var(--color-primary);
    background-color: var(--color-bg-card);
    border-radius: var(--radius-lg);
  }

  &__logo-text {
    font-size: var(--font-size-3xl);
    font-weight: 600;
  }

  &__slogan {
    margin-top: var(--spacing-8);
    font-size: 34px;
    color: var(--color-bg-card);
  }

  &__desc {
    margin-top: var(--spacing-4);
    font-size: var(--font-size-lg);
    line-height: 1.7;
    color: rgb(255 255 255 / 82%);
  }

  &__roles {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
    margin-top: var(--spacing-8);
  }

  &__role {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    font-size: var(--font-size-base);
    color: rgb(255 255 255 / 90%);
  }

  &__role-dot {
    width: 8px;
    height: 8px;
    background-color: var(--color-bg-card);
    border-radius: 50%;
  }

  &__form-side {
    display: flex;
    flex: 0 0 40%;
    align-items: center;
    justify-content: center;
    padding: var(--spacing-8);
    background-color: var(--color-bg-card);
  }

  &__form-box {
    width: 100%;
    max-width: 360px;
  }

  &__welcome {
    font-size: var(--font-size-3xl);
    color: var(--color-text-primary);
  }

  &__form-tip {
    margin: var(--spacing-2) 0 var(--spacing-6);
    font-size: var(--font-size-base);
    color: var(--color-text-tertiary);
  }

  &__submit {
    width: 100%;
  }

  &__demo {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-2);

    .el-button {
      flex: 1;
      margin-left: 0;
    }
  }

  &__demo-tip {
    margin-top: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
    text-align: center;
  }
}
</style>
