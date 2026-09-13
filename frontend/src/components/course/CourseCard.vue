<script setup lang="ts">
import { getCourseStatusMeta } from '@/constants'
import type { Course } from '@/types/course'

defineProps<{ course: Course }>()

const emit = defineEmits<{ (e: 'click', course: Course): void }>()
</script>

<template>
  <article class="course-card" @click="emit('click', course)">
    <header class="course-card__header">
      <div class="course-card__title">
        <h3 class="course-card__name">{{ course.name }}</h3>
        <span class="course-card__code">{{ course.code }}</span>
      </div>
      <el-tag :type="getCourseStatusMeta(course.status).tag" effect="light" round>
        {{ getCourseStatusMeta(course.status).label }}
      </el-tag>
    </header>

    <p class="course-card__desc">{{ course.description }}</p>

    <div class="course-card__meta">
      <span>{{ course.semester }}</span>
      <span>{{ course.department }}</span>
    </div>

    <footer class="course-card__footer">
      <div class="course-card__metric">
        <span class="course-card__metric-value tabular-nums">{{ course.classes.length }}</span>
        <span class="course-card__metric-label">班级</span>
      </div>
      <div class="course-card__metric">
        <span class="course-card__metric-value tabular-nums">{{ course.studentCount }}</span>
        <span class="course-card__metric-label">学生人次</span>
      </div>
      <div class="course-card__metric">
        <span class="course-card__metric-value tabular-nums">{{ course.resourceCount }}</span>
        <span class="course-card__metric-label">资源</span>
      </div>
    </footer>
  </article>
</template>

<style scoped lang="scss">
.course-card {
  display: flex;
  flex-direction: column;
  padding: var(--spacing-6);
  cursor: pointer;
  background-color: var(--color-bg-card);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  transition: box-shadow 0.2s ease, transform 0.2s ease;

  &:hover {
    box-shadow: var(--shadow-card-hover);
    transform: translateY(-2px);
  }

  &__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--spacing-3);
  }

  &__title {
    min-width: 0;
  }

  &__name {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__code {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__desc {
    display: -webkit-box;
    margin-top: var(--spacing-3);
    overflow: hidden;
    font-size: var(--font-size-sm);
    color: var(--color-text-tertiary);
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  &__meta {
    display: flex;
    gap: var(--spacing-3);
    margin-top: var(--spacing-3);
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }

  &__footer {
    display: flex;
    gap: var(--spacing-6);
    padding-top: var(--spacing-4);
    margin-top: var(--spacing-4);
    border-top: 1px solid var(--color-divider);
  }

  &__metric {
    display: flex;
    align-items: baseline;
    gap: var(--spacing-1);
  }

  &__metric-value {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__metric-label {
    font-size: var(--font-size-xs);
    color: var(--color-text-tertiary);
  }
}
</style>
