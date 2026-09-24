<script setup lang="ts">
import { onMounted, onUnmounted, reactive, ref } from 'vue'
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

/* 质量叙事：品牌区字幕轮播（功能本体即说明书） */
const NARRATIONS = ['督导锚点评分 · 评价有依据', '录音转写参考 · 过程可回看', '督导 × AI 双源融合综合分', 'AI 提优建议 · 让改进有落点']
const narrationIndex = ref(0)
let narrationTimer: ReturnType<typeof setInterval> | undefined

onMounted(() => {
  narrationTimer = setInterval(() => {
    narrationIndex.value = (narrationIndex.value + 1) % NARRATIONS.length
  }, 2600)
})

onUnmounted(() => {
  if (narrationTimer) clearInterval(narrationTimer)
})

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

        <!-- 品牌图形：五维质量雷达（产品功能本体的自证） -->
        <svg
          class="login__radar"
          viewBox="0 0 200 200"
          role="img"
          aria-label="五维教学质量雷达示意"
        >
          <g class="login__radar-grid">
            <polygon points="100,20 176.1,75.3 147,164.7 53,164.7 23.9,75.3" />
            <polygon points="100,36 160.9,80.2 137.6,151.8 62.4,151.8 39.1,80.2" />
            <polygon points="100,52 145.7,85.2 128.2,138.8 71.8,138.8 54.4,85.2" />
            <polygon points="100,68 130.4,90.1 118.8,125.9 81.2,125.9 69.6,90.1" />
            <polygon points="100,84 115.2,95.1 109.4,112.9 90.6,112.9 84.8,95.1" />
          </g>
          <g class="login__radar-axes">
            <line x1="100" y1="100" x2="100" y2="20" />
            <line x1="100" y1="100" x2="176.1" y2="75.3" />
            <line x1="100" y1="100" x2="147" y2="164.7" />
            <line x1="100" y1="100" x2="53" y2="164.7" />
            <line x1="100" y1="100" x2="23.9" y2="75.3" />
          </g>
          <polygon class="login__radar-shape" points="100,32 159.4,80.7 143.3,159.5 67.1,145.3 49.8,83.7" />
          <g class="login__radar-dots">
            <circle cx="100" cy="32" r="3" />
            <circle cx="159.4" cy="80.7" r="3" />
            <circle cx="143.3" cy="159.5" r="3" />
            <circle cx="67.1" cy="145.3" r="3" />
            <circle cx="49.8" cy="83.7" r="3" />
          </g>
        </svg>

        <div class="login__narration" aria-live="polite">
          <transition name="narration" mode="out-in">
            <span :key="narrationIndex" class="login__narration-item">
              {{ NARRATIONS[narrationIndex] }}
            </span>
          </transition>
        </div>

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
    background:
      radial-gradient(720px 480px at 72% 18%, rgba(47, 179, 68, 0.16), transparent 60%),
      radial-gradient(560px 420px at 18% 82%, rgba(139, 92, 246, 0.18), transparent 65%),
      var(--color-ink);
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
    color: #ffffff;
    background: var(--gradient-ai);
    border-radius: var(--radius-lg);
  }

  &__logo-text {
    font-size: var(--font-size-3xl);
    font-weight: 600;
  }

  &__slogan {
    margin-top: var(--spacing-6);
    font-size: 34px;
    color: var(--color-bg-card);
  }

  &__radar {
    display: block;
    width: min(280px, 100%);
    height: auto;
    margin: var(--spacing-8) auto 0;

    &-grid polygon {
      fill: none;
      stroke: rgb(255 255 255 / 14%);
      stroke-width: 1;
    }

    &-axes line {
      stroke: rgb(255 255 255 / 10%);
      stroke-width: 1;
    }

    &-shape {
      fill: rgb(47 179 68 / 26%);
      stroke: var(--color-score-5);
      stroke-width: 1.6;
      stroke-linejoin: round;
      animation: login-radar-breathe 3.6s ease-in-out infinite;
      transform-origin: center;
    }

    &-dots circle {
      fill: var(--color-score-5);
      stroke: var(--color-ink);
      stroke-width: 1;
    }
  }

  &__narration {
    height: 24px;
    margin-top: var(--spacing-5);
    overflow: hidden;
    text-align: center;

    &-item {
      display: inline-block;
      font-size: var(--font-size-base);
      letter-spacing: 1px;
      color: rgb(220 231 245 / 88%);
    }
  }

  &__roles {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-3);
    margin-top: var(--spacing-7);
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

/* 品牌雷达呼吸 + 字幕切换（reduced-motion 下降级） */
@keyframes login-radar-breathe {
  0%,
  100% {
    opacity: 0.72;
    transform: scale(0.985);
  }

  50% {
    opacity: 1;
    transform: scale(1);
  }
}

.narration-enter-active,
.narration-leave-active {
  transition: opacity 0.36s var(--ease-out-soft);
}

.narration-enter-from,
.narration-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .login__radar-shape {
    animation: none;
  }

  .narration-enter-active,
  .narration-leave-active {
    transition: none;
  }
}
</style>
