package kb

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFileMarkdownAndTXT(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "a.md")
	require.NoError(t, os.WriteFile(md, []byte("# 标题\r\n\r\n正文"), 0o600))
	text, format, err := ExtractFile(md)
	require.NoError(t, err)
	assert.Equal(t, "md", format)
	assert.Contains(t, text, "标题")
	assert.NotContains(t, text, "\r")

	txt := filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(txt, []byte("纯文本"), 0o600))
	text, format, err = ExtractFile(txt)
	require.NoError(t, err)
	assert.Equal(t, "txt", format)
	assert.Equal(t, "纯文本", text)
}

func TestExtractDOCX(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.docx")
	f, err := os.Create(path)
	require.NoError(t, err)
	zw := zip.NewWriter(f)
	w, err := zw.Create("word/document.xml")
	require.NoError(t, err)
	_, err = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>第一段</w:t></w:r></w:p>
    <w:p><w:r><w:t>第二段</w:t></w:r><w:r><w:t>续</w:t></w:r></w:p>
  </w:body>
</w:document>`))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())

	text, format, err := ExtractFile(path)
	require.NoError(t, err)
	assert.Equal(t, "docx", format)
	assert.Contains(t, text, "第一段")
	assert.Contains(t, text, "第二段续")
}

func TestExtractPDFRejected(t *testing.T) {
	_, _, err := ExtractFile(filepath.Join(t.TempDir(), "a.pdf"))
	require.ErrorIs(t, err, ErrPDFUnsupported)
}
