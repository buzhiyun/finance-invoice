package task

import (
	"io"
	"log"
	"strings"

	"github.com/ledongthuc/pdf"
)

func extractPDFText(pageFile string) []string {
	f, r, err := pdf.Open(pageFile)
	if err != nil {
		log.Printf("[PDF] 无法打开 %s: %v", pageFile, err)
		return nil
	}
	defer f.Close()

	reader, err := r.GetPlainText()
	if err != nil {
		log.Printf("[PDF] 无法提取文本 %s: %v", pageFile, err)
		return nil
	}

	var buf strings.Builder
	if _, err := io.Copy(&buf, reader); err != nil {
		log.Printf("[PDF] 读取文本失败 %s: %v", pageFile, err)
		return nil
	}

	text := buf.String()
	if text == "" {
		return nil
	}

	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return nil
	}

	log.Printf("[PDF] 提取到 %d 行文本: %s", len(lines), strings.Join(lines, " | "))
	return lines
}

// FieldMatchesPDFText checks if an extracted field value can be found
// in the PDF's raw text lines. Returns true if the value is confirmed
// or if validation is not possible (empty value or no PDF text).
func FieldMatchesPDFText(value string, pdfText []string) bool {
	if value == "" || len(pdfText) == 0 {
		return true
	}

	v := strings.TrimPrefix(value, "¥")
	v = strings.TrimPrefix(v, "￥")
	v = strings.ReplaceAll(v, ",", "")
	v = strings.TrimSpace(v)
	if v == "" {
		return true
	}

	for _, line := range pdfText {
		l := strings.ReplaceAll(line, " ", "")
		l = strings.ReplaceAll(l, ",", "")
		if strings.Contains(l, v) {
			return true
		}
		vNoSpace := strings.ReplaceAll(v, " ", "")
		if vNoSpace != v && strings.Contains(l, vNoSpace) {
			return true
		}
	}
	return false
}
