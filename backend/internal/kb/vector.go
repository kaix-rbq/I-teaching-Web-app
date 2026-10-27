package kb

import (
	"encoding/binary"
	"fmt"
	"math"
)

// EncodeVector 把 []float32 按小端紧密写入 LONGBLOB（RAG §4.2）。
func EncodeVector(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, x := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(x))
	}
	return buf
}

// DecodeVector 按维度还原向量；长度不符时报错，避免把不同模型的向量混用。
func DecodeVector(b []byte, dim int) ([]float32, error) {
	if dim <= 0 {
		return nil, fmt.Errorf("kb: 非法向量维度 %d", dim)
	}
	if len(b) != dim*4 {
		return nil, fmt.Errorf("kb: 向量长度 %d 字节，期望 %d 维（%d 字节）", len(b), dim, dim*4)
	}
	out := make([]float32, dim)
	for i := 0; i < dim; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out, nil
}
