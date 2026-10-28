package dto

// AgentChatHistoryItem 是 S3 对话历史项；role 仅允许 user / agent。
type AgentChatHistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AgentChatReq 是 POST /agent/chat 的请求体。
type AgentChatReq struct {
	CourseID uint64                 `json:"courseId"`
	Question string                 `json:"question"`
	History  []AgentChatHistoryItem `json:"history"`
}

// AgentChatCitation 是回答引用的知识库片段。
type AgentChatCitation struct {
	ChunkID string `json:"chunkId"`
	Title   string `json:"title"`
	Quote   string `json:"quote"`
}
