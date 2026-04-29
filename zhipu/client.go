package zhipu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	uploadURL = "https://bigmodel.cn/api/biz/file/uploadTemporaryImage"
	ocrURL    = "https://open.bigmodel.cn/api/paas/v4/layout_parsing"
	Model     = "glm-ocr"
)

type InvoiceFields struct {
	InvoiceType   string `json:"invoice_type"`
	InvoiceNumber string `json:"invoice_number"`
	InvoiceDate   string `json:"invoice_date"`
	BuyerName     string `json:"buyer_name"`
	BuyerTaxID    string `json:"buyer_tax_id"`
	SellerName    string `json:"seller_name"`
	SellerTaxID   string `json:"seller_tax_id"`
	ItemName      string `json:"item_name"`
	Amount        string `json:"amount"`
	TaxRate       string `json:"tax_rate"`
	TaxAmount     string `json:"tax_amount"`
	TotalUpper    string `json:"total_upper"`
	TotalLower    string `json:"total_lower"`
	Remarks       string `json:"remarks"`
	Issuer        string `json:"issuer"`
}

type OCRResponse struct {
	ID            string         `json:"id"`
	Model         string         `json:"model"`
	LayoutDetails [][]LayoutItem `json:"layout_details"`
	MDResults     string         `json:"md_results"`
	Error         *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

type LayoutItem struct {
	Content     string `json:"content"`
	Index       int    `json:"index"`
	Label       string `json:"label"`
	NativeLabel string `json:"native_label"`
}

type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

func (c *Client) UploadFile(filename string, data []byte) (string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeQuotes(filename)))
	h.Set("Content-Type", "application/pdf")
	part, err := writer.CreatePart(h)
	if err != nil {
		return "", fmt.Errorf("create form part: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("write file data: %w", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", uploadURL, &buf)
	if err != nil {
		return "", fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", c.APIKey)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://bigmodel.cn")
	req.Header.Set("Referer", "https://bigmodel.cn/trialcenter/modeltrial/visual")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read upload response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(body))
	}

	var uploadResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		URL  string `json:"url"`
	}
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return "", fmt.Errorf("parse upload response: %w, body: %s", err, string(body))
	}

	if uploadResp.Code != 200 {
		return "", fmt.Errorf("upload failed (code %d): %s, body: %s", uploadResp.Code, uploadResp.Msg, string(body))
	}

	if uploadResp.URL == "" {
		return "", fmt.Errorf("upload returned empty URL, body: %s", string(body))
	}

	return uploadResp.URL, nil
}

func (c *Client) RecognizeInvoice(fileURL string) (*InvoiceFields, error) {
	reqBody := map[string]any{
		"model": Model,
		"file":  fileURL,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", ocrURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create OCR request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OCR request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read OCR response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OCR failed (status %d): %s", resp.StatusCode, string(body))
	}

	var ocrResp OCRResponse
	if err := json.Unmarshal(body, &ocrResp); err != nil {
		return nil, fmt.Errorf("parse OCR response: %w, body: %s", err, string(body))
	}

	if ocrResp.Error != nil {
		return nil, fmt.Errorf("OCR API error [%s]: %s", ocrResp.Error.Code, ocrResp.Error.Message)
	}

	if ocrResp.MDResults == "" && len(ocrResp.LayoutDetails) == 0 {
		return nil, fmt.Errorf("OCR returned empty results")
	}

	log.Printf("[Zhipu] OCR原始返回: %s", string(body))

	return extractInvoiceFields(&ocrResp)
}

var (
	reInvoiceNumber = regexp.MustCompile(`发票号码[：:]\s*(\d+)`)
	reInvoiceDate   = regexp.MustCompile(`开票日期[：:]\s*(\d{4}\s*年\s*\d{1,2}\s*月\s*\d{1,2}\s*日)`)
	reTaxID         = regexp.MustCompile(`(?:统一社会信用代码[/／]纳税人识别号|纳税人识别号|统一社会信用代码)[：:]\s*([A-Za-z0-9]{15,20})`)
	reName          = regexp.MustCompile(`名称[：:][ \t]*(.+?)[ \t]*(?:统一社会|纳税人|项目名称|\n)`)
	reItemName      = regexp.MustCompile(`(\*[^*]+\*[^*¥\n]+)`)
	reAmountLine    = regexp.MustCompile(`合\s*计\s*[¥￥]?\s*([\d,.]+)(?:\s*[¥￥]?\s*([\d,.]+))?`)
	reTaxRate       = regexp.MustCompile(`(?:税率|征收率)[：:/／]*\s*(\d{1,2}%)`)
	reTotalUpper    = regexp.MustCompile(`[⊗⊙○]?\s*([零壹贰叁肆伍陆柒捌玖拾佰仟万亿圆元整角分]+)`)
	reTotalLower    = regexp.MustCompile(`小写[）)）]*[：:]*\s*[¥￥]?\s*([\d,.]+)`)
	reRemarks       = regexp.MustCompile(`备注[：:]*[ \t]*(.+)`)
	reIssuer        = regexp.MustCompile(`开票人[：:]\s*(.+)`)
	reHTMLTag       = regexp.MustCompile(`<[^>]+>`)
	reImgRef        = regexp.MustCompile(`!\[.*?\]\(.*?\)`)
	reMultiSpace    = regexp.MustCompile(`[ \t]+`)
	reMultiNewline  = regexp.MustCompile(`\n{3,}`)
	reBR            = regexp.MustCompile(`<br\s*/?>`)
	reTRClose       = regexp.MustCompile(`</tr>`)
	reTHClose       = regexp.MustCompile(`</th>`)
	reTDClose       = regexp.MustCompile(`</td>`)
	reHeadClose     = regexp.MustCompile(`</thead>`)
	reBodyClose     = regexp.MustCompile(`</tbody>`)
)

func extractInvoiceFields(ocrResp *OCRResponse) (*InvoiceFields, error) {
	text := stripHTMLTags(ocrResp.MDResults)
	fields := &InvoiceFields{}

	for _, page := range ocrResp.LayoutDetails {
		for _, item := range page {
			if item.NativeLabel == "figure_title" {
				fields.InvoiceType = strings.TrimSpace(stripHTMLTags(item.Content))
			}
		}
	}

	if m := reInvoiceNumber.FindStringSubmatch(text); len(m) > 1 {
		fields.InvoiceNumber = m[1]
	}

	if m := reInvoiceDate.FindStringSubmatch(text); len(m) > 1 {
		fields.InvoiceDate = strings.ReplaceAll(m[1], " ", "")
	}

	extractBuyerSeller(text, fields)

	itemMatches := reItemName.FindAllStringSubmatch(text, -1)
	if len(itemMatches) > 0 {
		var items []string
		for _, m := range itemMatches {
			items = append(items, strings.TrimSpace(m[1]))
		}
		fields.ItemName = strings.Join(items, "、")
	}

	if m := reAmountLine.FindStringSubmatch(text); len(m) > 1 {
		fields.Amount = m[1]
		if len(m) > 2 && m[2] != "" {
			fields.TaxAmount = m[2]
		}
	}

	if m := reTaxRate.FindStringSubmatch(text); len(m) > 1 {
		fields.TaxRate = m[1]
	} else {
		fields.TaxRate = computeTaxRate(fields.Amount, fields.TaxAmount)
	}

	if m := reTotalUpper.FindStringSubmatch(text); len(m) > 1 {
		fields.TotalUpper = m[1]
	}

	if m := reTotalLower.FindStringSubmatch(text); len(m) > 1 {
		fields.TotalLower = "¥" + m[1]
	}

	if m := reRemarks.FindStringSubmatch(text); len(m) > 1 {
		fields.Remarks = strings.TrimSpace(m[1])
	}

	if m := reIssuer.FindStringSubmatch(text); len(m) > 1 {
		fields.Issuer = strings.TrimSpace(m[1])
	}

	log.Printf("[Zhipu] 提取结果: 类型=%s, 号码=%s, 日期=%s, 购方=%s, 购方税号=%s, 销方=%s, 销方税号=%s, 项目=%s, 金额=%s, 税率=%s, 税额=%s, 大写=%s, 小写=%s, 备注=%s, 开票人=%s",
		fields.InvoiceType, fields.InvoiceNumber, fields.InvoiceDate,
		fields.BuyerName, fields.BuyerTaxID, fields.SellerName, fields.SellerTaxID,
		fields.ItemName, fields.Amount, fields.TaxRate, fields.TaxAmount,
		fields.TotalUpper, fields.TotalLower, fields.Remarks, fields.Issuer)

	return fields, nil
}

func extractBuyerSeller(text string, fields *InvoiceFields) {
	buyerSection, sellerSection := splitBuyerSeller(text)

	if m := reName.FindStringSubmatch(buyerSection); len(m) > 1 {
		fields.BuyerName = strings.TrimSpace(m[1])
	}
	if m := reTaxID.FindStringSubmatch(buyerSection); len(m) > 1 {
		fields.BuyerTaxID = m[1]
	}

	if m := reName.FindStringSubmatch(sellerSection); len(m) > 1 {
		fields.SellerName = strings.TrimSpace(m[1])
	}
	if m := reTaxID.FindStringSubmatch(sellerSection); len(m) > 1 {
		fields.SellerTaxID = m[1]
	}

	// Fallback: OCR sometimes drops the "销售方信息" cell, so "销售方" keyword
	// is missing and sellerSection is empty. In that case, find all "名称" and
	// tax ID occurrences in the full text and use the 2nd match for seller.
	if fields.SellerName == "" || fields.BuyerName == "" {
		nameMatches := reName.FindAllStringSubmatch(text, -1)
		if len(nameMatches) >= 1 && fields.BuyerName == "" {
			fields.BuyerName = strings.TrimSpace(nameMatches[0][1])
		}
		if len(nameMatches) >= 2 && fields.SellerName == "" {
			fields.SellerName = strings.TrimSpace(nameMatches[1][1])
		}

		taxIDMatches := reTaxID.FindAllStringSubmatch(text, -1)
		if len(taxIDMatches) >= 1 && fields.BuyerTaxID == "" {
			fields.BuyerTaxID = taxIDMatches[0][1]
		}
		if len(taxIDMatches) >= 2 && fields.SellerTaxID == "" {
			fields.SellerTaxID = taxIDMatches[1][1]
		}
	}
}

func splitBuyerSeller(text string) (buyer, seller string) {
	buyerIdx := firstIndex(text, "购买方", "购方")
	sellerIdx := firstIndex(text, "销售方", "销方")

	if buyerIdx < 0 && sellerIdx < 0 {
		return "", ""
	}

	if buyerIdx >= 0 && sellerIdx >= 0 && sellerIdx > buyerIdx {
		buyer = text[buyerIdx:sellerIdx]
		seller = text[sellerIdx:]
	} else if buyerIdx >= 0 {
		buyer = text[buyerIdx:]
	} else {
		seller = text[sellerIdx:]
	}

	return
}

func firstIndex(s string, subs ...string) int {
	minIdx := -1
	for _, sub := range subs {
		if i := strings.Index(s, sub); i >= 0 {
			if minIdx < 0 || i < minIdx {
				minIdx = i
			}
		}
	}
	return minIdx
}

func computeTaxRate(amountStr, taxAmountStr string) string {
	amount, err1 := strconv.ParseFloat(strings.ReplaceAll(amountStr, ",", ""), 64)
	taxAmount, err2 := strconv.ParseFloat(strings.ReplaceAll(taxAmountStr, ",", ""), 64)
	if err1 != nil || err2 != nil || amount <= 0 {
		return ""
	}
	rate := taxAmount / amount * 100
	for _, r := range []float64{1, 3, 5, 6, 9, 10, 13} {
		if math.Abs(rate-r) < 0.5 {
			return fmt.Sprintf("%.0f%%", r)
		}
	}
	return ""
}

func stripHTMLTags(s string) string {
	// Replace <br> with space before removing tags
	s = reBR.ReplaceAllString(s, " ")
	// Preserve table structure boundaries
	s = reTRClose.ReplaceAllString(s, "\n")
	s = reTHClose.ReplaceAllString(s, " ")
	s = reTDClose.ReplaceAllString(s, " ")
	s = reHeadClose.ReplaceAllString(s, "\n")
	s = reBodyClose.ReplaceAllString(s, "\n")
	// Remove all remaining HTML tags
	s = reHTMLTag.ReplaceAllString(s, "")
	// Remove image references
	s = reImgRef.ReplaceAllString(s, "")
	// Decode HTML entities
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	// Collapse whitespace
	s = reMultiSpace.ReplaceAllString(s, " ")
	s = reMultiNewline.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}
