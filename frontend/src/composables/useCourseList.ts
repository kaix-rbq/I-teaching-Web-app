import { onMounted, reactive, ref } from 'vue'
import { fetchCoursesApi } from '@/api/course'
import type { Course, CourseQuery, CourseStatus } from '@/types/course'
import { usePagination } from './usePagination'

export interface CourseListQuery {
  semester: string
  department: string
  teacherId: string
  status: CourseStatus | ''
  keyword: string
}

export function useCourseList() {
  const { page, pageSize, total, setTotal, setPage, setPageSize } = usePagination()
  const query = reactive<CourseListQuery>({
    semester: '',
    department: '',
    teacherId: '',
    status: '',
    keyword: ''
  })
  const rows = ref<Course[]>([])
  const loading = ref(false)
  const error = ref(false)

  async function fetchList(): Promise<void> {
    loading.value = true
    error.value = false
    try {
      const params: CourseQuery = {
        ...query,
        status: query.status || undefined,
        page: page.value,
        pageSize: pageSize.value
      }
      const result = await fetchCoursesApi(params)
      rows.value = result.list
      setTotal(result.total)
    } catch {
      error.value = true
      rows.value = []
      setTotal(0)
    } finally {
      loading.value = false
    }
  }

  function search(): void {
    setPage(1)
    void fetchList()
  }

  function reset(): void {
    query.semester = ''
    query.department = ''
    query.teacherId = ''
    query.status = ''
    query.keyword = ''
    setPage(1)
    void fetchList()
  }

  function handlePageChange(value: number): void {
    setPage(value)
    void fetchList()
  }

  function handlePageSizeChange(value: number): void {
    setPageSize(value)
    void fetchList()
  }

  onMounted(() => {
    void fetchList()
  })

  return {
    query,
    rows,
    loading,
    error,
    page,
    pageSize,
    total,
    fetchList,
    search,
    reset,
    handlePageChange,
    handlePageSizeChange
  }
}
