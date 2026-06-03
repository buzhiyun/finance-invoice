package zhipu

import (
	"testing"
)

// Test with the sample from user's original message (full-width colons, no <br>)
func TestExtractInvoiceFields_Sample1(t *testing.T) {
	ocrResp := &OCRResponse{
		MDResults: `![](page=0,bbox=[65, 61, 250, 244])

<div align="center">

电子发票（增值税专用发票）

</div>

全国统一发票监制章

国家税务总局

深圳市税务局

发票号码：26957000000105785659

开票日期：2026年04月15日

<table border="1"><tr><td>购买方信息</td><td colspan="4">名称：安徽七天网络科技有限公司
统一社会信用代码/纳税人识别号：9134010007564947X2</td><td>销售方信息</td><td colspan="4">名称：深圳市腾讯计算机系统有限公司
统一社会信用代码/纳税人识别号：91440300708461136T</td></tr><tr><td colspan="10">项目名称规格型号单位数量单价金额税率/征收率税额
*信息系统增值服务*技术服务
合计¥283.02¥16.98</td></tr><tr><td colspan="2">价税合计（大写）</td><td colspan="8">⊗叁佰元整（小写）¥300.00</td></tr><tr><td>备注</td><td colspan="9">销方开户银行：招商银行深圳汉京中心支行 银行账号：817282299610001；</td></tr></table>

开票人：谢同燕`,
		LayoutDetails: [][]LayoutItem{
			{
				{Content: "<div align=\"center\">\n\n电子发票（增值税专用发票）\n\n</div>", NativeLabel: "figure_title", Label: "text", Index: 1},
			},
		},
	}

	fields, err := extractInvoiceFields(ocrResp)
	if err != nil {
		t.Fatalf("extractInvoiceFields failed: %v", err)
	}

	assertField(t, "InvoiceType", "电子发票（增值税专用发票）", fields.InvoiceType)
	assertField(t, "InvoiceNumber", "26957000000105785659", fields.InvoiceNumber)
	assertField(t, "InvoiceDate", "2026年04月15日", fields.InvoiceDate)
	assertField(t, "BuyerName", "安徽七天网络科技有限公司", fields.BuyerName)
	assertField(t, "BuyerTaxID", "9134010007564947X2", fields.BuyerTaxID)
	assertField(t, "SellerName", "深圳市腾讯计算机系统有限公司", fields.SellerName)
	assertField(t, "SellerTaxID", "91440300708461136T", fields.SellerTaxID)
	assertField(t, "ItemName", "*信息系统增值服务*技术服务", fields.ItemName)
	assertField(t, "Amount", "283.02", fields.Amount)
	assertField(t, "TaxRate", "6%", fields.TaxRate)
	assertField(t, "TaxAmount", "16.98", fields.TaxAmount)
	assertField(t, "TotalUpper", "叁佰元整", fields.TotalUpper)
	assertField(t, "TotalLower", "¥300.00", fields.TotalLower)
	assertField(t, "Remarks", "销方开户银行：招商银行深圳汉京中心支行 银行账号：817282299610001；", fields.Remarks)
	assertField(t, "Issuer", "谢同燕", fields.Issuer)
}

// Test with the real OCR output (half-width colons, <br>, <th>, space in 合 计)
func TestExtractInvoiceFields_Sample2(t *testing.T) {
	ocrResp := &OCRResponse{
		MDResults: `![](page=0,bbox=[58, 52, 251, 246])

<div align="center">

电子发票（普通发票）

</div>

全国统一发票监制章

国家税务总局

河南省税务局

发票号码：26412000000911845396

开票日期：2026年03月17日

<table class="table table-bordered"><thead><tr><th>购买方信息</th><td colspan="3">名称:安徽七天网络科技有限公司<br>统一社会信用代码/纳税人识别号:9134010007564947X2</td><th>销售方信息</th><td colspan="3">名称:河南业朋网络科技有限公司<br>统一社会信用代码/纳税人识别号:91410103MACRCE8H7P</td></tr></thead><tbody><tr><td colspan="8">项目名称<br>*现代服务*阅卷系统服务费</td></tr><tr><td colspan="8">合 计<br>¥1253.24<br>¥12.53</td></tr><tr><td colspan="2">价税合计（大写）</td><td colspan="6">⊗壹仟贰佰陆拾伍圆柒角柒分<br>(小写)¥1265.77</td></tr><tr><td>备注</td><td colspan="7">销方开户银行:中国农业银行股份有限公司郑州大学南路支行; 银行账号:16055201040012831</td></tr></tbody></table>

开票人：惠军平`,
		LayoutDetails: [][]LayoutItem{
			{
				{Content: "<div align=\"center\">\n\n电子发票（普通发票）\n\n</div>", NativeLabel: "figure_title", Label: "text", Index: 1},
			},
		},
	}

	fields, err := extractInvoiceFields(ocrResp)
	if err != nil {
		t.Fatalf("extractInvoiceFields failed: %v", err)
	}

	assertField(t, "InvoiceType", "电子发票（普通发票）", fields.InvoiceType)
	assertField(t, "InvoiceNumber", "26412000000911845396", fields.InvoiceNumber)
	assertField(t, "InvoiceDate", "2026年03月17日", fields.InvoiceDate)
	assertField(t, "BuyerName", "安徽七天网络科技有限公司", fields.BuyerName)
	assertField(t, "BuyerTaxID", "9134010007564947X2", fields.BuyerTaxID)
	assertField(t, "SellerName", "河南业朋网络科技有限公司", fields.SellerName)
	assertField(t, "SellerTaxID", "91410103MACRCE8H7P", fields.SellerTaxID)
	assertField(t, "ItemName", "*现代服务*阅卷系统服务费", fields.ItemName)
	assertField(t, "Amount", "1253.24", fields.Amount)
	assertField(t, "TaxRate", "1%", fields.TaxRate)
	assertField(t, "TaxAmount", "12.53", fields.TaxAmount)
	assertField(t, "TotalUpper", "壹仟贰佰陆拾伍圆柒角柒分", fields.TotalUpper)
	assertField(t, "TotalLower", "¥1265.77", fields.TotalLower)
	assertField(t, "Remarks", "销方开户银行:中国农业银行股份有限公司郑州大学南路支行; 银行账号:16055201040012831", fields.Remarks)
	assertField(t, "Issuer", "惠军平", fields.Issuer)
}

// Test with missing "销售方信息" cell - OCR drops the cell, seller info
// appears as the second "名称" in the same table row without a "销售方" keyword.
func TestExtractInvoiceFields_MissingSellerCell(t *testing.T) {
	ocrResp := &OCRResponse{
		MDResults: `![](page=0,bbox=[58, 52, 251, 245])

<div align="center">

电子发票（普通发票）

</div>

全国统一发票监制章

国家税务总局

甘肃省税务局

发票号码：26622000000179523856

开票日期：2026年03月17日

<table border="1"><tr><td>购买方信息</td><td colspan="4">名称：安徽七天网络科技有限公司
统一社会信用代码/纳税人识别号：9134010007564947X2</td><td colspan="3">名称：甘肃中启网络科技有限公司
统一社会信用代码/纳税人识别号：91620104MAELGLFD88</td></tr><tr><td colspan="9">项目名称
*现代服务*服务费
合计
¥7916.73
¥79.17</td></tr><tr><td colspan="2">价税合计（大写）</td><td colspan="6">⊗柒仟玖佰玖拾伍圆玖角整
(小写)¥7995.90</td></tr><tr><td>备注</td><td colspan="7">销方开户银行：中国建设银行股份有限公司兰州银滩支行；银行账号：62050110228500000932</td></tr></table>

开票人：刘见华`,
		LayoutDetails: [][]LayoutItem{
			{
				{Content: "<div align=\"center\">\n\n电子发票（普通发票）\n\n</div>", NativeLabel: "figure_title", Label: "text", Index: 1},
			},
		},
	}

	fields, err := extractInvoiceFields(ocrResp)
	if err != nil {
		t.Fatalf("extractInvoiceFields failed: %v", err)
	}

	assertField(t, "InvoiceType", "电子发票（普通发票）", fields.InvoiceType)
	assertField(t, "InvoiceNumber", "26622000000179523856", fields.InvoiceNumber)
	assertField(t, "InvoiceDate", "2026年03月17日", fields.InvoiceDate)
	assertField(t, "BuyerName", "安徽七天网络科技有限公司", fields.BuyerName)
	assertField(t, "BuyerTaxID", "9134010007564947X2", fields.BuyerTaxID)
	assertField(t, "SellerName", "甘肃中启网络科技有限公司", fields.SellerName)
	assertField(t, "SellerTaxID", "91620104MAELGLFD88", fields.SellerTaxID)
	assertField(t, "ItemName", "*现代服务*服务费", fields.ItemName)
	assertField(t, "Amount", "7916.73", fields.Amount)
	assertField(t, "TaxRate", "1%", fields.TaxRate)
	assertField(t, "TaxAmount", "79.17", fields.TaxAmount)
	assertField(t, "TotalUpper", "柒仟玖佰玖拾伍圆玖角整", fields.TotalUpper)
	assertField(t, "TotalLower", "¥7995.90", fields.TotalLower)
	assertField(t, "Remarks", "销方开户银行：中国建设银行股份有限公司兰州银滩支行；银行账号：62050110228500000932", fields.Remarks)
	assertField(t, "Issuer", "刘见华", fields.Issuer)
}

// Test with missing "销售方信息" cell AND "销方名称" in remarks — the "销方"
// keyword in remarks must NOT be used as the seller section boundary.
// This is the bug case where seller tax ID 91131003MA7HFPJJ1Y was lost.
func TestExtractInvoiceFields_SellerNameInRemarks(t *testing.T) {
	ocrResp := &OCRResponse{
		MDResults: `![](page=0,bbox=[58, 51, 251, 245])

<div align="center">

电子发票（普通发票）

</div>

全国统一发票监制章

国家税务总局

河北省税务局

发票号码：2613200001552803991

开票日期：2026年05月22日

<table border="1"><tr><td>购买方信息</td><td colspan="4">名称：安徽七天网络科技有限公司
统一社会信用代码/纳税人识别号：9134010007564947X2</td><td colspan="3">名称：河北拓永网络科技有限公司
统一社会信用代码/纳税人识别号：91131003MA7HFPJJ1Y</td></tr><tr><td colspan="9">项目名称规格型号单位数量单价金额税率/征收率税额
*其他教育辅助服务*网络阅卷服务费
合计¥1908.55¥19.09</td></tr><tr><td colspan="2">价税合计（大写）</td><td colspan="6">⊗壹仟玖佰贰拾柒圆陆角肆分（小写）¥1927.64</td></tr><tr><td>备注</td><td colspan="7">销方名称：河北拓永网络科技有限公司 销方地址及电话：廊坊市广阳区锦绣花苑17-3-501 19933670678 销方开户行及账号：中国银行廊坊市广阳道支行101926233449</td></tr></table>

开票人：郝浴满`,
		LayoutDetails: [][]LayoutItem{
			{
				{Content: "<div align=\"center\">\n\n电子发票（普通发票）\n\n</div>", NativeLabel: "figure_title", Label: "text", Index: 1},
			},
		},
	}

	fields, err := extractInvoiceFields(ocrResp)
	if err != nil {
		t.Fatalf("extractInvoiceFields failed: %v", err)
	}

	assertField(t, "InvoiceType", "电子发票（普通发票）", fields.InvoiceType)
	assertField(t, "InvoiceNumber", "2613200001552803991", fields.InvoiceNumber)
	assertField(t, "InvoiceDate", "2026年05月22日", fields.InvoiceDate)
	assertField(t, "BuyerName", "安徽七天网络科技有限公司", fields.BuyerName)
	assertField(t, "BuyerTaxID", "9134010007564947X2", fields.BuyerTaxID)
	assertField(t, "SellerName", "河北拓永网络科技有限公司", fields.SellerName)
	assertField(t, "SellerTaxID", "91131003MA7HFPJJ1Y", fields.SellerTaxID)
	assertField(t, "ItemName", "*其他教育辅助服务*网络阅卷服务费", fields.ItemName)
	assertField(t, "Amount", "1908.55", fields.Amount)
	assertField(t, "TaxRate", "1%", fields.TaxRate)
	assertField(t, "TaxAmount", "19.09", fields.TaxAmount)
	assertField(t, "TotalUpper", "壹仟玖佰贰拾柒圆陆角肆分", fields.TotalUpper)
	assertField(t, "TotalLower", "¥1927.64", fields.TotalLower)
	assertField(t, "Issuer", "郝浴满", fields.Issuer)
}

// Test with empty remarks - should NOT capture "开票人" into remarks
func TestExtractInvoiceFields_EmptyRemarks(t *testing.T) {
	ocrResp := &OCRResponse{
		MDResults: `<table><tr><td>备注</td><td colspan="9"></td></tr></table>

开票人：陈玉龙`,
	}

	fields, err := extractInvoiceFields(ocrResp)
	if err != nil {
		t.Fatalf("extractInvoiceFields failed: %v", err)
	}

	assertField(t, "Remarks", "", fields.Remarks)
	assertField(t, "Issuer", "陈玉龙", fields.Issuer)
}

func TestExtractTotalUpper_With捌(t *testing.T) {
	text := stripHTMLTags("⊗壹万零捌拾捌圆壹角贰分 (小写)¥10088.12")
	m := reTotalUpper.FindStringSubmatch(text)
	if len(m) < 2 {
		t.Fatalf("reTotalUpper did not match, text: %s", text)
	}
	if m[1] != "壹万零捌拾捌圆壹角贰分" {
		t.Errorf("TotalUpper = %q, want %q", m[1], "壹万零捌拾捌圆壹角贰分")
	}
}

func TestComputeTaxRate(t *testing.T) {
	tests := []struct {
		amount    string
		taxAmount string
		want      string
	}{
		{"283.02", "16.98", "6%"},
		{"100.00", "6.00", "6%"},
		{"100.00", "13.00", "13%"},
		{"100.00", "9.00", "9%"},
		{"100.00", "3.00", "3%"},
		{"100.00", "1.00", "1%"},
		{"1253.24", "12.53", "1%"},
		{"0", "6.00", ""},
		{"100.00", "0", ""},
		{"abc", "6.00", ""},
	}

	for _, tt := range tests {
		got := computeTaxRate(tt.amount, tt.taxAmount)
		if got != tt.want {
			t.Errorf("computeTaxRate(%q, %q) = %q, want %q", tt.amount, tt.taxAmount, got, tt.want)
		}
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			"<div align=\"center\">\n\n电子发票\n\n</div>",
			"电子发票",
		},
		{
			"<table><tr><td>名称：公司A</td><td>金额：100</td></tr></table>",
			"名称：公司A 金额：100",
		},
		{
			"<th>购买方信息</th><td>名称:公司A<br>税号:123</td>",
			"购买方信息 名称:公司A 税号:123",
		},
		{
			"![](page=0,bbox=[65, 61, 250, 244])\n\n文本",
			"文本",
		},
	}

	for _, tt := range tests {
		got := stripHTMLTags(tt.input)
		if got != tt.want {
			t.Errorf("stripHTMLTags(%q)\n= %q\nwant %q", tt.input, got, tt.want)
		}
	}
}

func assertField(t *testing.T, name, want, got string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}
