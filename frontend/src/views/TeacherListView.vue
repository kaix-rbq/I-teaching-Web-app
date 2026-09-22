<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { fetchTeacherScoresApi } from '@/api/teacher'
import type { TeacherScoreItem } from '@/types/teacher'

const router = useRouter(); const loading = ref(true); const rows = ref<TeacherScoreItem[]>([])
async function load(){ loading.value=true; try { rows.value=(await fetchTeacherScoresApi()).list } finally { loading.value=false } }
function score(v:number|null){ return v == null ? '—' : v.toFixed(2) }
onMounted(load)
</script>
<template>
  <PageHeader><template #title>教师评价</template><template #subtitle>综合分、五维评分与样本覆盖</template></PageHeader>
  <el-skeleton v-if="loading" :rows="8" animated />
  <EmptyState v-else-if="!rows.length" description="暂无教师评价数据" />
  <el-table v-else :data="rows" stripe row-key="teacherId">
    <el-table-column prop="teacherName" label="教师" min-width="150" />
    <el-table-column prop="jobNo" label="工号" width="130" />
    <el-table-column label="综合分" width="120"><template #default="{row}">{{ score(row.compositeScore) }}</template></el-table-column>
    <el-table-column label="评价次数" width="110"><template #default="{row}">{{ row.sample.evaluatedCount }}</template></el-table-column>
    <el-table-column label="状态" width="140"><template #default="{row}"><el-tag v-if="!row.sample.sampleSufficient" type="warning">样本不足</el-tag><el-tag v-else type="success">样本充足</el-tag></template></el-table-column>
    <el-table-column label="操作" width="120"><template #default="{row}"><el-button link type="primary" @click="router.push({name:'teacher-detail',params:{id:row.teacherId}})">查看</el-button></template></el-table-column>
  </el-table>
</template>
