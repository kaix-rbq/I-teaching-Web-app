import { ref } from 'vue'

interface PaginationOptions {
  page?: number
  pageSize?: number
}

export function usePagination(options: PaginationOptions = {}) {
  const page = ref(options.page ?? 1)
  const pageSize = ref(options.pageSize ?? 10)
  const total = ref(0)

  function setTotal(value: number): void {
    total.value = value
  }

  function setPage(value: number): void {
    page.value = value
  }

  function setPageSize(value: number): void {
    pageSize.value = value
    page.value = 1
  }

  function reset(): void {
    page.value = 1
  }

  return { page, pageSize, total, setTotal, setPage, setPageSize, reset }
}
