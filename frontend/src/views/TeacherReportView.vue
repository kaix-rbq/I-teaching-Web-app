<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchTeacherReportApi } from '@/api/teacher'
import type { TeacherReport } from '@/types/teacher'
import { useSemester } from '@/composables/useSemester'
import PageHeader from '@/components/common/PageHeader.vue'
import QualityBadge from '@/components/common/QualityBadge.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const route = useRoute()
const router = useRouter()
const { semester } = useSemester()

const loading = ref(false)
const failed = ref(false)
const report = ref<TeacherReport | null>(null)

const teacherId = computed(() => String(route.params.id))

async function load(): Promise<void> {
  loading.value = true
  failed.value = false
  try {
    report.value = await fetchTeacherReportApi(teacherId.value, { semester: semester.value })
  } catch {
    failed.value = true
    report.value = null
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(semester, load)

const summary = computed(() => report.value?.summary ?? null)

function print(): void {
  window.print()
}
</script>

<template>
  <div class="report">
    <PageHeader :title="'教学质量报告'" subtitle="一页纸 · 仅用于教学支持，不作为考核依据">
      <template #actions>
        <el-button @click="router.back()">返回</el-button>
        <el-button type="primary" :disabled="!report" @click="print">打印 / 导出 PDF</el-button>
      </template>
    </PageHeader>

    <el-alert
      class="report__ethics"
      title="仅用于教学支持，不作为考核依据"
      type="warning"
      :closable="false"
      show-icon
    />

    <el-skeleton v-if="loading" :rows="8" animated />
    <EmptyState v-else-if="failed" description="报告加载失败">
      <template #action>
        <el-button @click="load">重新加载</el-button>
      </template>
    </EmptyState>

    <template v-else-if="report && summary">
      <section class="report__card report__head">
        <div class="report__headline">
          <h2>{{ report.teacherName }}</h2>
          <span class="report__meta">{{ report.department }} · 工号 {{ report.jobNo }} · {{ report.semester }}</span>
        </div>
        <div class="report__composite">
          <span class="report__composite-value">
            {{ summary.compositeScore === null ? '—' : summary.compositeScore.toFixed(2) }}
          </span>
          <QualityBadge :score="summary.compositeScore" show-score />
          <span class="report__sample">
            已评价 {{ summary.sample.evaluatedCount }} 次课
            <template v-if="!summary.sample.sampleSufficient">（样本不足）</template>
          </span>
        </div>
      </section>

      <section class="report__card">
        <h3 class="report__title">五维评分（AI 计分）</h3>
        <ul class="report__dims">
          <li v-for="d in summary.dimensions" :key="d.key" class="report__dim">
            <span class="report__dim-name">{{ d.name }}</span>
            <span class="report__dim-bar">
              <i :style="{ width: `${d.score ?? 0}%` }" />
            </span>
            <span class="report__dim-score">{{ d.score === null ? '—' : d.score.toFixed(1) }}</span>
            <span v-if="d.isObservation" class="report__dim-tag">观测项</span>
          </li>
        </ul>
      </section>

      <section class="report__card">
        <h3 class="report__title">最近督导评语</h3>
        <EmptyState v-if="report.recentComments.length === 0" description="暂无督导评语" />
        <ul v-else class="report__comments">
          <li v-for="(c, i) in report.recentComments" :key="i" class="report__comment">
            <p class="report__comment-meta">{{ c.evaluatorName || '督导' }} · {{ c.createdAt.slice(0, 10) }}</p>
            <p v-if="c.comment" class="report__comment-text">{{ c.comment }}</p>
            <p v-if="c.highlights" class="report__comment-line"><b>亮点：</b>{{ c.highlights }}</p>
            <p v-if="c.improvements" class="report__comment-line"><b>主要问题：</b>{{ c.improvements }}</p>
            <p v-if="c.suggestions" class="report__comment-line"><b>改进建议：</b>{{ c.suggestions }}</p>
          </li>
        </ul>
      </section>

      <section class="report__card">
        <h3 class="report__title">待改进清单</h3>
        <EmptyState v-if="report.improvementItems.length === 0" description="暂无待改进项" />
        <ul v-else class="report__items">
          <li v-for="it in report.improvementItems" :key="it.id" class="report__item">
            <span class="report__item-status" :data-status="it.status">
              {{ it.status === 'closed' ? '已关闭' : it.status === 'done_self' ? '待督导确认' : '进行中' }}
            </span>
            <span class="report__item-title">{{ it.title }}</span>
            <span v-if="it.doneNote" class="report__item-note">{{ it.doneNote }}</span>
          </li>
        </ul>
      </section>

      <p class="report__footer">生成时间：{{ report.generatedAt.slice(0, 19).replace('T', ' ') }}</p>
    </template>
  </div>
</template>

<style scoped lang="scss">
.report { display: flex; flex-direction: column; gap: var(--spacing-4); }
.report__ethics { margin-bottom: var(--spacing-2); }
.report__card { background: var(--color-bg-card); border: 1px solid var(--color-border); border-radius: var(--radius-lg); padding: var(--spacing-6); }
.report__head { display: flex; align-items: center; justify-content: space-between; }
.report__headline h2 { margin: 0; font-size: var(--font-size-2xl); color: var(--color-text-primary); }
.report__meta { font-size: var(--font-size-sm); color: var(--color-text-tertiary); }
.report__composite { display: flex; align-items: baseline; gap: var(--spacing-3); }
.report__composite-value { font-family: var(--font-score); font-size: var(--font-size-score-xl); color: var(--color-text-primary); }
.report__sample { font-size: var(--font-size-sm); color: var(--color-text-tertiary); }
.report__title { margin: 0 0 var(--spacing-4); font-size: var(--font-size-lg); color: var(--color-text-primary); }
.report__dims, .report__comments, .report__items { list-style: none; margin: 0; padding: 0; }
.report__dim { display: grid; grid-template-columns: 180px 1fr 56px auto; align-items: center; gap: var(--spacing-3); padding: var(--spacing-2) 0; }
.report__dim-name { font-size: var(--font-size-base); color: var(--color-text-secondary); }
.report__dim-bar { display: block; height: 8px; background: var(--color-divider); border-radius: var(--radius-sm); overflow: hidden; }
.report__dim-bar i { display: block; height: 100%; background: var(--gradient-score); }
.report__dim-score { font-family: var(--font-score); text-align: right; }
.report__dim-tag { font-size: var(--font-size-xs); color: var(--color-text-tertiary); }
.report__comment { padding: var(--spacing-3) 0; border-bottom: 1px solid var(--color-divider); }
.report__comment:last-child { border-bottom: none; }
.report__comment-meta { margin: 0 0 var(--spacing-2); font-size: var(--font-size-sm); color: var(--color-text-tertiary); }
.report__comment-text { margin: 0 0 var(--spacing-1); color: var(--color-text-primary); }
.report__comment-line { margin: 0 0 var(--spacing-1); font-size: var(--font-size-sm); color: var(--color-text-secondary); }
.report__item { display: flex; align-items: center; gap: var(--spacing-3); padding: var(--spacing-2) 0; }
.report__item-status { font-size: var(--font-size-xs); padding: 2px 8px; border-radius: var(--radius-sm); background: var(--color-primary-bg); color: var(--color-primary); }
.report__item-status[data-status='closed'] { background: var(--color-success); color: #fff; }
.report__item-title { color: var(--color-text-primary); }
.report__item-note { font-size: var(--font-size-sm); color: var(--color-text-tertiary); }
.report__footer { text-align: right; font-size: var(--font-size-xs); color: var(--color-text-tertiary); }

@media print {
  .report__ethics { display: none; }
  .report__card { border: none; padding: 0 0 var(--spacing-4); break-inside: avoid; }
}
</style>
