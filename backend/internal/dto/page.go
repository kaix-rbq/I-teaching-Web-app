package dto

// PageResult 是分页响应信封（与前端 PageResult<T> 对齐）。
type PageResult[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

// NewPageResult 组装分页响应；list 为 nil 时归一化为空数组，避免前端拿到 null。
func NewPageResult[T any](list []T, total int64, page, pageSize int) *PageResult[T] {
	if list == nil {
		list = []T{}
	}
	return &PageResult[T]{List: list, Total: total, Page: page, PageSize: pageSize}
}
