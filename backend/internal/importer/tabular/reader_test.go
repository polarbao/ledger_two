package tabular

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func TestReadAlipayCSVFindsPreambleHeaderAndDecodesGB18030(t *testing.T) {
	var lines []string
	for i := 1; i < 24; i++ {
		lines = append(lines, fmt.Sprintf("说明%d", i))
	}
	lines = append(lines,
		"交易时间,交易分类,交易对方,对方账号,商品说明,收/支,金额,收/付款方式,交易状态,交易订单号,商家订单号,备注,",
		"2026-07-01 12:30:00,餐饮美食,示例商户,/,午餐,支出,35.80,余额,交易成功,000123,merchant-1,,",
	)

	encoded, _, err := transform.Bytes(simplifiedchinese.GB18030.NewEncoder(), []byte(strings.Join(lines, "\r\n")))
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}

	doc, err := Read("alipay.csv", SourceAlipay, encoded)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if doc.Format != FormatCSV || doc.Metadata.HeaderRowNumber != 24 {
		t.Fatalf("unexpected document metadata: %+v", doc)
	}
	if len(doc.Rows) != 1 || doc.Rows[0].Number != 1 {
		t.Fatalf("unexpected rows: %+v", doc.Rows)
	}
	if got := doc.Rows[0].Values[2]; got != "示例商户" {
		t.Fatalf("merchant = %q, want 示例商户", got)
	}
}

func TestReadWechatXLSXFindsHeaderOnRow18(t *testing.T) {
	content := buildWorkbook(t, map[string][][]any{
		"Sheet1": {
			{18, "交易时间", "交易类型", "交易对方", "商品", "收/支", "金额(元)", "支付方式", "当前状态", "交易单号", "商户单号", "备注"},
			{19, "2026-07-01 12:30:00", "商户消费", "示例商户", "午餐", "支出", "35.80", "零钱", "支付成功", "000123", "merchant-1", ""},
		},
	})

	doc, err := Read("wechat.xlsx", SourceWechat, content)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if doc.Format != FormatXLSX || doc.Metadata.SheetName != "Sheet1" || doc.Metadata.HeaderRowNumber != 18 {
		t.Fatalf("unexpected document metadata: %+v", doc.Metadata)
	}
	if len(doc.Rows) != 1 || doc.Rows[0].Number != 19 {
		t.Fatalf("unexpected rows: %+v", doc.Rows)
	}
	if got := doc.Rows[0].Values[8]; got != "000123" {
		t.Fatalf("order id = %q, want 000123", got)
	}
}

func TestReadRejectsFormatMismatchAndNonWechatXLSX(t *testing.T) {
	xlsx := buildWorkbook(t, map[string][][]any{"Sheet1": {{1, "title"}}})

	if _, err := Read("renamed.csv", SourceWechat, xlsx); !errors.Is(err, ErrFormatMismatch) {
		t.Fatalf("expected ErrFormatMismatch, got %v", err)
	}
	if _, err := Read("generic.xlsx", SourceGeneric, xlsx); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
	if _, err := Read("alipay.xlsx", SourceAlipay, xlsx); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected Alipay XLSX to return ErrUnsupportedFormat, got %v", err)
	}
}

func TestReadRejectsAmbiguousWorkbook(t *testing.T) {
	header := []any{1, "交易时间", "交易类型", "交易对方", "商品", "收/支", "金额(元)", "支付方式", "当前状态", "交易单号", "商户单号", "备注"}
	xlsx := buildWorkbook(t, map[string][][]any{
		"账单一": {header},
		"账单二": {header},
	})

	if _, err := Read("wechat.xlsx", SourceWechat, xlsx); !errors.Is(err, ErrAmbiguousWorkbook) {
		t.Fatalf("expected ErrAmbiguousWorkbook, got %v", err)
	}
}

func buildWorkbook(t *testing.T, sheets map[string][][]any) []byte {
	t.Helper()

	file := excelize.NewFile()
	t.Cleanup(func() { _ = file.Close() })
	first := true
	for sheetName, rows := range sheets {
		if first {
			if err := file.SetSheetName("Sheet1", sheetName); err != nil {
				t.Fatalf("rename first sheet: %v", err)
			}
			first = false
		} else if _, err := file.NewSheet(sheetName); err != nil {
			t.Fatalf("create sheet %s: %v", sheetName, err)
		}
		for _, row := range rows {
			rowNumber := row[0].(int)
			values := append([]any(nil), row[1:]...)
			axis, err := excelize.CoordinatesToCellName(1, rowNumber)
			if err != nil {
				t.Fatalf("make row axis: %v", err)
			}
			if err := file.SetSheetRow(sheetName, axis, &values); err != nil {
				t.Fatalf("set sheet row %s!%s: %v", sheetName, axis, err)
			}
		}
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	return bytes.Clone(buffer.Bytes())
}
