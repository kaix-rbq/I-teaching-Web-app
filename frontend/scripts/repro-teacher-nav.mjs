/**
 * 教师端连续导航白屏回归脚本（Task A）。
 *
 * 用法：先启动后端 :8080 与前端 :5173，再执行
 *   node scripts/repro-teacher-nav.mjs
 *
 * 脚本会登录教师账号，依次切换全部教师页面，断言 <main> 有渲染内容；
 * 任一步白屏或出现未捕获异常都会以非 0 退出。
 */
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { chromium } from 'playwright'

const BASE = process.env.REPRO_BASE_URL || 'http://127.0.0.1:5173'
const OUT_DIR = path.resolve('test-results/teacher-nav')

function chromiumExecutable() {
  if (process.env.PLAYWRIGHT_CHROMIUM_PATH) return process.env.PLAYWRIGHT_CHROMIUM_PATH
  const cache = path.join(os.homedir(), '.cache/ms-playwright')
  if (!fs.existsSync(cache)) return undefined
  const versions = fs
    .readdirSync(cache)
    .filter((name) => name.startsWith('chromium-'))
    .sort()
    .reverse()
  for (const version of versions) {
    const candidate = path.join(cache, version, 'chrome-linux64/chrome')
    if (fs.existsSync(candidate)) return candidate
  }
  return undefined
}

const executablePath = chromiumExecutable()
const browser = await chromium.launch({ executablePath, headless: true })
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
const failures = []

page.on('pageerror', (error) => failures.push(`pageerror: ${error.message}`))
page.on('console', (message) => {
  if (message.type() === 'warning' && message.text().includes('non-element root node')) {
    failures.push(`transition warning: ${message.text()}`)
  }
})

fs.mkdirSync(OUT_DIR, { recursive: true })

try {
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await page.getByPlaceholder('请输入账号').fill(process.env.REPRO_USERNAME || 'teacher')
  await page.getByPlaceholder('请输入密码').fill(process.env.REPRO_PASSWORD || '123456')
  await page.getByRole('button', { name: '登录' }).click()
  await page.waitForTimeout(1600)

  const pages = [
    ['工作台', '/workbench'],
    ['我的质量档案', '/me/quality'],
    ['我的课程', '/courses'],
    ['教学反思', '/reflections'],
    ['个人中心', '/profile'],
    ['工作台', '/workbench']
  ]

  for (const [label, expectedPath] of pages) {
    await page.locator('.el-menu-item', { hasText: label }).first().click()
    await page.waitForTimeout(900)
    const text = (await page.locator('main').innerText()).trim()
    const urlOk = new URL(page.url()).pathname === expectedPath
    const rendered = text.length > 20
    console.log(`${rendered && urlOk ? 'PASS' : 'FAIL'} ${label} -> ${new URL(page.url()).pathname} (${text.length} chars)`)
    if (!rendered || !urlOk) failures.push(`${label}: rendered=${rendered} urlOk=${urlOk}`)
    await page.screenshot({
      path: path.join(OUT_DIR, `${label}-${Date.now()}.png`),
      fullPage: true
    })
  }
} finally {
  await browser.close()
}

if (failures.length) {
  console.error('\nFAILED')
  for (const failure of failures) console.error(`- ${failure}`)
  process.exit(1)
}
console.log('\nPASS: 教师端连续导航无白屏')
