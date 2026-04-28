package zhipu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

const (
	uploadURL = "https://bigmodel.cn/api/biz/file/uploadTemporaryImage"
	chatURL   = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	Model     = "glm-5v-turbo"
)

var invoicePrompt = `你是一个专业的发票识别助手。请仔细识别这张发票图片，提取以下信息并以JSON格式返回。只返回JSON对象，不要返回任何其他内容。

重要提示：
- 发票号码通常在发票右上角区域，标注为"发票号码"，是一串20位数字（全电发票）或8位数字（传统发票），请务必仔细查找并完整提取
- 全电发票（电子发票）的发票号码为20位纯数字
- 传统增值税发票的发票号码为8位纯数字
- 注意区分"发票号码"和"发票代码"，发票代码通常更长且在前方，发票号码是我们需要的字段

字段说明：
- invoice_type: 发票类型（如：电子发票（普通发票）、增值税电子普通发票、增值税专用发票等，按票面实际印制名称填写）
- invoice_number: 发票号码（全电发票为20位数字，传统发票为8位数字，必须完整提取）
- invoice_date: 开票日期（格式：YYYY年MM月DD日）
- buyer_name: 购买方名称
- buyer_tax_id: 购买方统一社会信用代码/纳税人识别号
- seller_name: 销售方名称
- seller_tax_id: 销售方统一社会信用代码/纳税人识别号
- item_name: 项目名称（如有多个项目，用顿号分隔）
- amount: 金额（不含税）
- tax_rate: 税率（如：6%、9%、13%、1%）
- tax_amount: 税额
- total_upper: 价税合计（大写）
- total_lower: 价税合计（小写/阿拉伯数字）
- remarks: 备注
- issuer: 开票人

如果某个字段确实无法识别，请返回空字符串。`

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

type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
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
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "file_url",
						"file_url": map[string]string{
							"url": fileURL,
						},
					},
					{
						"type": "text",
						"text": invoicePrompt,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", chatURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chat request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read chat response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chat failed (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parse chat response: %w, body: %s", err, string(body))
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error [%s]: %s", chatResp.Error.Code, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from model")
	}

	content := chatResp.Choices[0].Message.Content
	log.Printf("[Zhipu] 模型原始输出: %s", content)
	return parseInvoiceFields(content)
}

func parseInvoiceFields(content string) (*InvoiceFields, error) {
	jsonStr := content
	if idx := strings.Index(content, "{"); idx >= 0 {
		endIdx := strings.LastIndex(content, "}")
		if endIdx > idx {
			jsonStr = content[idx : endIdx+1]
		}
	}

	var fields InvoiceFields
	if err := json.Unmarshal([]byte(jsonStr), &fields); err != nil {
		return nil, fmt.Errorf("parse invoice fields: %w, content: %s", err, content)
	}

	return &fields, nil
}

func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}
