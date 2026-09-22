<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { fetchCourseDetailApi, fetchCourseEvaluationSummaryApi } from '@/api/course'
import type { Course, CourseEvaluationSummary } from '@/types/course'
const route=useRoute(); const course=ref<Course|null>(null); const summary=ref<CourseEvaluationSummary|null>(null); const loading=ref(true)
async function load(){loading.value=true;try{const id=String(route.params.id);[course.value,summary.value]=await Promise.all([fetchCourseDetailApi(id),fetchCourseEvaluationSummaryApi(id)])}finally{loading.value=false}}
function score(v:number|null){return v==null?'—':v.toFixed(2)}
onMounted(load)
</script>
<template><PageHeader><template #title>教学提优</template><template #subtitle>{{course?.name||'课程评价与改进'}}</template></PageHeader><el-skeleton v-if="loading" :rows="8" animated/><EmptyState v-else-if="!course||!summary" description="课程评价暂无数据"/><template v-else><el-card><template #header>课程信息</template><el-descriptions :column="3" border><el-descriptions-item label="课程">{{course.name}}</el-descriptions-item><el-descriptions-item label="课程代码">{{course.code}}</el-descriptions-item><el-descriptions-item label="授课教师">{{course.teacherName}}</el-descriptions-item><el-descriptions-item label="学期">{{course.semester}}</el-descriptions-item><el-descriptions-item label="学分">{{course.credit}}</el-descriptions-item><el-descriptions-item label="学时">{{course.hours}}</el-descriptions-item></el-descriptions></el-card><el-card class="course-improve__card"><template #header>评价摘要</template><div class="course-improve__score">{{score(summary.compositeScore)}}<small>综合分</small></div><el-table :data="summary.dimensions"><el-table-column prop="name" label="维度"/><el-table-column label="评分"><template #default="{row}">{{score(row.score)}}</template></el-table-column><el-table-column label="备注"><template #default="{row}">{{row.isObservation?'观察项，不计入综合分':'督导与智能体融合评分'}}</template></el-table-column></el-table></el-card></template></template>
<style scoped lang="scss">.course-improve__card{margin-top:var(--spacing-4)}.course-improve__score{font-size:var(--font-size-stat);font-weight:600;color:var(--color-primary);margin-bottom:var(--spacing-4)}.course-improve__score small{font-size:var(--font-size-sm);font-weight:400;color:var(--color-text-tertiary);margin-left:var(--spacing-2)}</style>
