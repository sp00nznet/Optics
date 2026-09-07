// Headless end-to-end smoke test: boots against a running stack, drives every
// page in a real browser and drops a screenshot of each into docs/screenshots.
//
// Usage: docker compose up -d && npm --prefix test install && npm --prefix test run smoke
import { chromium } from 'playwright'
import { mkdir, rm } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const BASE = process.env.OPTICS_URL ?? 'http://localhost:3333'
const API = process.env.OPTICS_API ?? 'http://localhost:8080'
const SHOTS = join(dirname(fileURLToPath(import.meta.url)), '..', 'docs', 'screenshots')
const USER = { username: 'demo', email: 'demo@example.com', password: 'demopassword123' }

const problems = []
let page

const api = async (path, opts) => {
  const res = await fetch(API + path, opts)
  return { status: res.status, body: await res.json().catch(() => null) }
}

// Screenshot + assert the page actually rendered the thing we came for.
async function shot(name, expect) {
  await page.waitForLoadState('networkidle')
  // Thumbnails render off the network, on a shared WebGL context - wait for the
  // loading placeholders to clear or we screenshot half-drawn grids.
  await page
    .waitForFunction(() => !document.querySelector('.animate-pulse'), null, { timeout: 30000 })
    .catch(() => problems.push(`${name}: thumbnails never finished loading`))
  if (expect) {
    try {
      await page.locator(expect).first().waitFor({ state: 'visible', timeout: 15000 })
    } catch {
      problems.push(`${name}: expected "${expect}" never appeared`)
    }
  }
  await page.screenshot({ path: join(SHOTS, `${name}.png`), fullPage: true })
  console.log(`  captured ${name}.png`)
}

const run = async () => {
  await rm(SHOTS, { recursive: true, force: true })
  await mkdir(SHOTS, { recursive: true })

  const health = await api('/health')
  if (health.status !== 200) throw new Error(`backend not up at ${API} (got ${health.status})`)

  // First registered user becomes admin; a re-run just logs in instead.
  const reg = await api('/api/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(USER),
  })
  if (![201, 409].includes(reg.status)) throw new Error(`register failed: ${reg.status}`)

  const login = await api('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: USER.email, password: USER.password }),
  })
  if (login.status !== 200) throw new Error(`login failed: ${login.status}`)
  const auth = { Authorization: `Bearer ${login.body.token}` }

  await api('/api/admin/scan', { method: 'POST', headers: auth })
  for (let i = 0; i < 60; i++) {
    const { body } = await api('/api/admin/scan/status', { headers: auth })
    if (!body?.running) break
    await new Promise((r) => setTimeout(r, 1000))
  }
  const { body: stats } = await api('/api/admin/stats', { headers: auth })
  console.log(`scan complete: ${stats.total_models} models, ${stats.total_categories} categories`)
  if (!stats.total_models) throw new Error('scan indexed nothing - is sample-models/ mounted?')

  await api('/api/admin/duplicates/detect', { method: 'POST', headers: auth })

  // SwiftShader gives the STL viewer/thumbnails a real WebGL context in headless.
  const browser = await chromium.launch({ args: ['--enable-unsafe-swiftshader'] })
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  page = await ctx.newPage()
  page.on('console', (m) => m.type() === 'error' && problems.push(`console: ${m.text()}`))
  page.on('requestfailed', (r) => problems.push(`request failed: ${r.url()}`))
  page.on('response', (r) => r.status() >= 500 && problems.push(`${r.status()} from ${r.url()}`))

  await page.goto(`${BASE}/register`)
  await shot('01-register', 'form')

  await page.goto(`${BASE}/login`)
  await page.fill('input[type=email]', USER.email)
  await page.fill('input[type=password]', USER.password)
  await shot('02-login', 'button[type=submit]')
  await page.click('button[type=submit]')
  await page.waitForURL(`${BASE}/`)

  // Logged-in session must survive a full page load, not just client-side nav.
  await shot('03-home', `header >> text=${USER.username}`)
  await page.goto(`${BASE}/browse`)
  await shot('04-browse', 'canvas, img')
  await page.goto(`${BASE}/categories`)
  await shot('05-categories', 'text=Calibration')

  const { body: models } = await api('/api/models?limit=50')
  const stl = models.models.find((m) => m.file_type === 'stl')
  await page.goto(`${BASE}/models/${stl.id}`)
  await shot('06-model-detail', 'canvas')

  for (const [name, path] of [['07-duplicates', 'duplicates'], ['08-admin', 'admin']]) {
    await page.goto(`${BASE}/${path}`)
    await shot(name, `header >> text=${USER.username}`)
    if (!page.url().endsWith(`/${path}`)) problems.push(`${name}: redirected to ${page.url()}`)
  }

  await browser.close()
}

try {
  await run()
} catch (e) {
  problems.unshift(e.message)
}

if (problems.length) {
  console.error(`\nFAILED (${problems.length}):`)
  for (const p of new Set(problems)) console.error(`  - ${p}`)
  process.exit(1)
}
console.log('\nOK - all pages rendered, screenshots in docs/screenshots/')
