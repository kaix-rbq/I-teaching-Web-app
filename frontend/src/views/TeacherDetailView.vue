<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { fetchTeacherEvaluationsApi, fetchTeacherSummaryApi } from '@/api/teacher'
import type { TeacherSummary, TeacherTimelineItem } from '@/types/teacher'
const route=useRoute(); const summary=ref<TeacherSummary|null>(null); const timeline=ref<TeacherTimelineItem[]>([]); const loading=ref(true)
async function load(){loading.value=true;try{const id=String(route.params.id);[summary.value,timeline.value]=await Promise.all([fetchTeacherSummaryApi(id),fetchTeacherEvaluationsApi(id).then(v=>v.list)])}finally{loading.value=false}}
function score(v:number|null){return v==null?'—':v.toFixed(2)}
onMounted(load)
</script>
<template>
 <PageHeader><template #title>{{ summary?.teacherName || '教师详情' }}</template><template #subtitle>综合评分与历次评价</template></PageHeader>
 <el-skeleton v-if="loading" :rows="10" animated />
 <EmptyState v-else-if="!summary" description="教师评价不存在" />
 <template v-else>
  <div class="teacher-detail__score"><strong>{{ score(summary.compositeScore) }}</strong><span>综合分</span><el-tag v-if="!summary.sample.sampleSufficient" type="warning">样本不足（n={{summary.sample.evaluatedCount}}）</el-tag></div>
  <el-card><template #header>五维评分</template><el-table :data="summary.dimensions"><el-table-column prop="name" label="维度"/><el-table-column label="评分"><template #default="{row}">{{score(row.score)}}</template></el-table-column><el-table-column label="权重"><template #default="{row}">{{row.isObservation?'观察项':`${Math.round(row.weight*100)}%`}}</template></el-table-column></el-table></el-card>
  <el-card class="teacher-detail__timeline"><template #header>历次评价</template><el-table :data="timeline"><el-table-column prop="sessionDate" label="日期" width="130"/><el-table-column prop="courseName" label="课程"/><el-table-column prop="topic" label="主题"/><el-table-column label="综合分" width="110"><template #default="{row}">{{score(row.compositeScore)}}</template></el-table-column></el-table></el-card>
 </template>
</template>
<style scoped lang="scss">.teacher-detail__score{display:flex;align-items:center;gap:var(--spacing-4);padding:var(--spacing-6);margin-bottom:var(--spacing-4);background:var(--color-bg-card);border:1px solid var(--color-divider);border-radius:var(--radius-lg)}.teacher-detail__score strong{font-size:var(--font-size-stat);color:var(--color-primary)}.teacher-detail__score span{color:var(--color-text-tertiary)}.teacher-detail__timeline{margin-top:var(--spacing-4)}</style>
