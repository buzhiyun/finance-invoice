package excel

import (
	"fmt"

	"github.com/buzhiyun/finance-invoice/task"
	"github.com/xuri/excelize/v2"
)

var headers = []string{
	"发票类型", "发票号码", "开票日期",
	"购方名称", "购方统一社会信用代码/纳税人识别号",
	"销方名称", "销方统一社会信用代码/纳税人识别号",
	"项目名称", "金额", "税率", "税额",
	"价税合计（大写）", "（小写）",
	"备注", "开票人", "原始文件名", "识别状态",
}

var colWidths = []float64{
	18, // A 发票类型
	28, // B 发票号码
	16, // C 开票日期
	30, // D 购方名称
	26, // E 购方统一社会信用代码
	30, // F 销方名称
	26, // G 销方统一社会信用代码
	20, // H 项目名称
	12, // I 金额
	8,  // J 税率
	12, // K 税额
	30, // L 价税合计（大写）
	14, // M （小写）
	32, // N 备注
	10, // O 开票人
	22, // P 原始文件名
	10, // Q 识别状态
}

type Generator struct{}

func (g *Generator) Generate(t *task.BatchTask, outputPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"
	index, err := f.GetSheetIndex(sheet)
	if err != nil {
		return fmt.Errorf("get sheet index: %w", err)
	}
	f.SetActiveSheet(index)

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
	})
	if err != nil {
		return fmt.Errorf("create header style: %w", err)
	}

	cellStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
	})
	if err != nil {
		return fmt.Errorf("create cell style: %w", err)
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	row := 2
	for _, ft := range t.Files {
		for _, pr := range ft.Pages {
			status := "成功"
			if pr.Status == "failed" {
				status = fmt.Sprintf("失败(%s)", pr.Error)
			}

			values := []string{
				pr.Fields.InvoiceType,
				pr.Fields.InvoiceNumber,
				pr.Fields.InvoiceDate,
				pr.Fields.BuyerName,
				pr.Fields.BuyerTaxID,
				pr.Fields.SellerName,
				pr.Fields.SellerTaxID,
				pr.Fields.ItemName,
				pr.Fields.Amount,
				pr.Fields.TaxRate,
				pr.Fields.TaxAmount,
				pr.Fields.TotalUpper,
				pr.Fields.TotalLower,
				pr.Fields.Remarks,
				pr.Fields.Issuer,
				ft.Filename,
				status,
			}

			for i, v := range values {
				cell, _ := excelize.CoordinatesToCellName(i+1, row)
				f.SetCellValue(sheet, cell, v)
				f.SetCellStyle(sheet, cell, cell, cellStyle)
			}
			row++
		}
	}

	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("save excel: %w", err)
	}

	return nil
}
