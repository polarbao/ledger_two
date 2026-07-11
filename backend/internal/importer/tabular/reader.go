package tabular

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	FormatCSV  = "csv"
	FormatXLSX = "xlsx"

	SourceWechat  = "wechat"
	SourceAlipay  = "alipay"
	SourceGeneric = "generic"

	maxSheets     = 10
	maxHeaderRows = 50
	maxColumns    = 64
	MaxDataRows   = 2000
)

var (
	ErrFormatMismatch    = errors.New("file extension does not match content")
	ErrUnsupportedFormat = errors.New("unsupported import file format")
	ErrHeaderNotFound    = errors.New("supported bill header not found")
	ErrAmbiguousWorkbook = errors.New("multiple worksheets match the selected source")
	ErrUnsupportedLayout = errors.New("unsupported workbook structure")
	ErrTooManyRows       = errors.New("import file contains too many rows")
)

type Metadata struct {
	ParserVersion   string `json:"parser_version"`
	SheetName       string `json:"sheet_name,omitempty"`
	HeaderRowNumber int    `json:"header_row_number"`
	ParsedRows      int    `json:"parsed_rows"`
	ScannedSheets   int    `json:"scanned_sheets,omitempty"`
	MaxColumns      int    `json:"max_columns"`
}

type Row struct {
	Number int
	Values []string
}

type Document struct {
	Format   string
	Header   []string
	Rows     []Row
	Metadata Metadata
}

type sheetCandidate struct {
	name        string
	headerIndex int
	rows        [][]string
}

func Read(filename, sourceType string, content []byte) (*Document, error) {
	format, err := detectFormat(filename, content)
	if err != nil {
		return nil, err
	}
	if format == FormatXLSX && sourceType == SourceGeneric {
		return nil, ErrUnsupportedFormat
	}

	switch format {
	case FormatCSV:
		return readCSV(sourceType, content)
	case FormatXLSX:
		return readXLSX(sourceType, content)
	default:
		return nil, ErrUnsupportedFormat
	}
}

func detectFormat(filename string, content []byte) (string, error) {
	extension := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	isZIP := len(content) >= 4 && bytes.Equal(content[:4], []byte{'P', 'K', 0x03, 0x04})

	switch extension {
	case ".csv":
		if isZIP {
			return "", ErrFormatMismatch
		}
		return FormatCSV, nil
	case ".xlsx":
		if !isZIP {
			return "", ErrFormatMismatch
		}
		if err := validateOOXML(content); err != nil {
			return "", err
		}
		return FormatXLSX, nil
	default:
		return "", ErrUnsupportedFormat
	}
}

func validateOOXML(content []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return fmt.Errorf("%w: invalid zip container", ErrUnsupportedFormat)
	}
	required := map[string]bool{
		"[Content_Types].xml": false,
		"_rels/.rels":         false,
		"xl/workbook.xml":     false,
	}
	for _, file := range reader.File {
		if _, ok := required[file.Name]; ok {
			required[file.Name] = true
		}
		lower := strings.ToLower(file.Name)
		if strings.Contains(lower, "vbaproject") || strings.HasSuffix(lower, ".bin") {
			return fmt.Errorf("%w: macro or binary workbook part", ErrUnsupportedFormat)
		}
	}
	for name, present := range required {
		if !present {
			return fmt.Errorf("%w: missing %s", ErrUnsupportedFormat, name)
		}
	}
	return nil
}

func readCSV(sourceType string, content []byte) (*Document, error) {
	decoded, err := decodeCSV(content)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(strings.NewReader(decoded))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	headerIndex := findHeader(sourceType, records)
	if headerIndex < 0 {
		return nil, ErrHeaderNotFound
	}

	doc := &Document{
		Format: FormatCSV,
		Header: append([]string(nil), records[headerIndex]...),
		Metadata: Metadata{
			ParserVersion:   "tabular-v1",
			HeaderRowNumber: headerIndex + 1,
		},
	}
	for _, values := range records[headerIndex+1:] {
		if isEmpty(values) {
			continue
		}
		if len(doc.Rows) >= MaxDataRows {
			return nil, ErrTooManyRows
		}
		doc.Rows = append(doc.Rows, Row{Number: len(doc.Rows) + 1, Values: append([]string(nil), values...)})
		if len(values) > doc.Metadata.MaxColumns {
			doc.Metadata.MaxColumns = len(values)
		}
	}
	doc.Metadata.ParsedRows = len(doc.Rows)
	return doc, nil
}

func decodeCSV(content []byte) (string, error) {
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(content) {
		return string(content), nil
	}
	decoded, _, err := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), content)
	if err != nil || !utf8.Valid(decoded) {
		return "", fmt.Errorf("decode csv: %w", err)
	}
	return string(decoded), nil
}

func readXLSX(sourceType string, content []byte) (*Document, error) {
	file, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{
		UnzipSizeLimit:    32 << 20,
		UnzipXMLSizeLimit: 8 << 20,
	})
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer file.Close()

	sheetNames := file.GetSheetList()
	if len(sheetNames) > maxSheets {
		return nil, ErrUnsupportedLayout
	}
	candidates := make([]sheetCandidate, 0, 1)
	scanned := 0
	for _, sheetName := range sheetNames {
		visible, err := file.GetSheetVisible(sheetName)
		if err != nil {
			return nil, err
		}
		if !visible {
			continue
		}
		scanned++
		rows, err := file.GetRows(sheetName)
		if err != nil {
			return nil, err
		}
		headerIndex := findHeader(sourceType, rows)
		if headerIndex >= 0 {
			candidates = append(candidates, sheetCandidate{name: sheetName, headerIndex: headerIndex, rows: rows})
		}
	}
	if len(candidates) == 0 {
		return nil, ErrHeaderNotFound
	}
	if len(candidates) > 1 {
		return nil, ErrAmbiguousWorkbook
	}

	candidate := candidates[0]
	if err := validateDataRegion(file, candidate); err != nil {
		return nil, err
	}
	doc := &Document{
		Format: FormatXLSX,
		Header: append([]string(nil), candidate.rows[candidate.headerIndex]...),
		Metadata: Metadata{
			ParserVersion:   "xlsx-v1",
			SheetName:       candidate.name,
			HeaderRowNumber: candidate.headerIndex + 1,
			ScannedSheets:   scanned,
		},
	}
	for index, values := range candidate.rows[candidate.headerIndex+1:] {
		if isEmpty(values) {
			continue
		}
		if len(doc.Rows) >= MaxDataRows {
			return nil, ErrTooManyRows
		}
		if len(values) > maxColumns {
			return nil, ErrUnsupportedLayout
		}
		doc.Rows = append(doc.Rows, Row{
			Number: candidate.headerIndex + 2 + index,
			Values: append([]string(nil), values...),
		})
		if len(values) > doc.Metadata.MaxColumns {
			doc.Metadata.MaxColumns = len(values)
		}
	}
	doc.Metadata.ParsedRows = len(doc.Rows)
	return doc, nil
}

func validateDataRegion(file *excelize.File, candidate sheetCandidate) error {
	dataStartRow := candidate.headerIndex + 2
	mergeCells, err := file.GetMergeCells(candidate.name, true)
	if err != nil {
		return err
	}
	for _, merged := range mergeCells {
		_, startRow, err := excelize.CellNameToCoordinates(merged.GetStartAxis())
		if err != nil {
			return err
		}
		_, endRow, err := excelize.CellNameToCoordinates(merged.GetEndAxis())
		if err != nil {
			return err
		}
		if endRow >= dataStartRow && startRow <= len(candidate.rows) {
			return fmt.Errorf("%w: merged cells in data region", ErrUnsupportedLayout)
		}
	}

	for rowIndex, values := range candidate.rows[candidate.headerIndex+1:] {
		rowNumber := candidate.headerIndex + 2 + rowIndex
		if isEmpty(values) {
			continue
		}
		visible, err := file.GetRowVisible(candidate.name, rowNumber)
		if err != nil {
			return err
		}
		if !visible {
			return fmt.Errorf("%w: hidden row %d", ErrUnsupportedLayout, rowNumber)
		}
		for column := 1; column <= len(values); column++ {
			columnName, err := excelize.ColumnNumberToName(column)
			if err != nil {
				return err
			}
			columnVisible, err := file.GetColVisible(candidate.name, columnName)
			if err != nil {
				return err
			}
			if !columnVisible {
				return fmt.Errorf("%w: hidden column %s", ErrUnsupportedLayout, columnName)
			}
			axis, err := excelize.CoordinatesToCellName(column, rowNumber)
			if err != nil {
				return err
			}
			formula, err := file.GetCellFormula(candidate.name, axis)
			if err != nil {
				return err
			}
			if formula != "" {
				return fmt.Errorf("%w: formula in %s", ErrUnsupportedLayout, axis)
			}
		}
	}
	return nil
}

func findHeader(sourceType string, rows [][]string) int {
	groups := requiredHeaderGroupsForSource(sourceType)
	if len(groups) == 0 {
		return -1
	}
	limit := len(rows)
	if limit > maxHeaderRows {
		limit = maxHeaderRows
	}
	for index := 0; index < limit; index++ {
		headers := make(map[string]bool, len(rows[index]))
		for _, value := range rows[index] {
			headers[normalizeHeader(value)] = true
		}
		matched := true
		for _, aliases := range groups {
			groupMatched := false
			for _, alias := range aliases {
				if headers[normalizeHeader(alias)] {
					groupMatched = true
					break
				}
			}
			if !groupMatched {
				matched = false
				break
			}
		}
		if matched {
			return index
		}
	}
	return -1
}

func requiredHeaderGroupsForSource(sourceType string) [][]string {
	switch sourceType {
	case SourceWechat:
		return [][]string{{"交易时间"}, {"交易类型"}, {"交易对方"}, {"商品"}, {"收/支"}, {"金额(元)", "金额"}, {"交易单号"}}
	case SourceAlipay:
		return [][]string{{"交易时间", "付款时间", "交易创建时间"}, {"交易分类", "类型"}, {"交易对方"}, {"商品说明", "商品名称"}, {"收/支"}, {"金额", "金额(元)"}, {"交易订单号", "交易号"}}
	case SourceGeneric:
		return [][]string{{"occurred_at"}, {"title"}, {"amount_cents"}, {"direction"}}
	default:
		return nil
	}
}

func normalizeHeader(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "（", "(")
	value = strings.ReplaceAll(value, "）", ")")
	return strings.ToLower(value)
}

func isEmpty(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}
