package importer

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

var cst = time.FixedZone("CST", 8*60*60)

func ParseCSV(sourceType string, reader io.Reader) (*Preview, error) {
	records, err := readCSV(reader)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errors.New("csv is empty")
	}

	header := makeHeaderIndex(records[0])
	rows := records[1:]

	var normalized []PreviewRow
	for i, row := range rows {
		rowNumber := i + 1
		if isEmptyRow(row) {
			continue
		}

		var parsed PreviewRow
		var err error
		switch sourceType {
		case SourceTypeWechat:
			parsed = parseWechatRow(rowNumber, header, row)
		case SourceTypeAlipay:
			parsed = parseAlipayRow(rowNumber, header, row)
		case SourceTypeGeneric:
			parsed = parseGenericRow(rowNumber, header, row)
		default:
			err = fmt.Errorf("unsupported source type: %s", sourceType)
		}
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, parsed)
	}

	markSuspiciousRows(normalized)

	return &Preview{
		SourceType: sourceType,
		Rows:       normalized,
	}, nil
}

func readCSV(reader io.Reader) ([][]string, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true
	return csvReader.ReadAll()
}

func makeHeaderIndex(headers []string) map[string]int {
	index := make(map[string]int, len(headers))
	for i, header := range headers {
		index[normalizeHeader(header)] = i
	}
	return index
}

func normalizeHeader(header string) string {
	header = strings.TrimSpace(header)
	header = strings.ReplaceAll(header, "（", "(")
	header = strings.ReplaceAll(header, "）", ")")
	return strings.ToLower(header)
}

func cell(header map[string]int, row []string, names ...string) string {
	for _, name := range names {
		idx, ok := header[normalizeHeader(name)]
		if !ok || idx >= len(row) {
			continue
		}
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func isEmptyRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func parseWechatRow(rowNumber int, header map[string]int, row []string) PreviewRow {
	txType := cell(header, row, "交易类型")
	direction := normalizeDirection(cell(header, row, "收/支"), txType)
	amount, amountErr := parseYuanToCents(cell(header, row, "金额(元)", "金额（元）"))

	result := baseRow(rowNumber, direction, cell(header, row, "商品"), cell(header, row, "交易对方"), cell(header, row, "备注"))
	result.AmountCents = amount
	result.ExternalOrderID = cell(header, row, "交易单号")
	result.SourceAccount = cell(header, row, "支付方式")

	if occurredAt, err := parseLocalTime(cell(header, row, "交易时间")); err == nil {
		result.OccurredAt = occurredAt
	} else {
		markInvalid(&result, ErrorCodeTimeInvalid, "交易时间无效")
	}
	if amountErr != nil {
		markInvalid(&result, ErrorCodeAmountInvalid, "金额不能为空或格式不正确")
	}

	return result
}

func parseAlipayRow(rowNumber int, header map[string]int, row []string) PreviewRow {
	txType := cell(header, row, "类型")
	direction := normalizeDirection(cell(header, row, "收/支"), txType)
	amount, amountErr := parseYuanToCents(cell(header, row, "金额(元)", "金额（元）"))

	result := baseRow(rowNumber, direction, cell(header, row, "商品名称"), cell(header, row, "交易对方"), cell(header, row, "备注"))
	result.AmountCents = amount
	result.ExternalOrderID = cell(header, row, "交易号")

	occurredRaw := cell(header, row, "付款时间")
	if occurredRaw == "" {
		occurredRaw = cell(header, row, "交易创建时间")
	}
	if occurredAt, err := parseLocalTime(occurredRaw); err == nil {
		result.OccurredAt = occurredAt
	} else {
		markInvalid(&result, ErrorCodeTimeInvalid, "交易时间无效")
	}
	if amountErr != nil {
		markInvalid(&result, ErrorCodeAmountInvalid, "金额不能为空或格式不正确")
	}

	return result
}

func parseGenericRow(rowNumber int, header map[string]int, row []string) PreviewRow {
	direction := normalizeDirection(cell(header, row, "direction"), "")
	amount, amountErr := parseCents(cell(header, row, "amount_cents"))

	result := baseRow(rowNumber, direction, cell(header, row, "title"), cell(header, row, "merchant"), cell(header, row, "description"))
	result.AmountCents = amount
	result.ExternalOrderID = cell(header, row, "external_order_id")
	result.SourceAccount = cell(header, row, "source_account")

	if occurredAt, err := parseRFC3339Time(cell(header, row, "occurred_at")); err == nil {
		result.OccurredAt = occurredAt
	} else {
		markInvalid(&result, ErrorCodeTimeInvalid, "交易时间无效")
	}
	if amountErr != nil {
		markInvalid(&result, ErrorCodeAmountInvalid, "金额不能为空或格式不正确")
	}

	return result
}

func baseRow(rowNumber int, direction, title, merchant, description string) PreviewRow {
	targetType := targetTransactionType(direction)
	rowStatus := RowStatusPending
	if targetType == TargetTransactionSkipped {
		rowStatus = RowStatusSkipped
	}

	return PreviewRow{
		RowNumber:             rowNumber,
		Title:                 strings.TrimSpace(title),
		Merchant:              strings.TrimSpace(merchant),
		Description:           strings.TrimSpace(description),
		Direction:             direction,
		TargetTransactionType: targetType,
		DuplicateStatus:       DuplicateStatusNew,
		RowStatus:             rowStatus,
	}
}

func normalizeDirection(rawDirection, txType string) string {
	rawDirection = strings.TrimSpace(strings.ToLower(rawDirection))
	txType = strings.TrimSpace(strings.ToLower(txType))

	if strings.Contains(txType, "退款") {
		return DirectionRefund
	}
	if strings.Contains(txType, "转账") || rawDirection == "transfer" {
		return DirectionTransfer
	}

	switch rawDirection {
	case "支出", "expense":
		return DirectionExpense
	case "收入", "income":
		return DirectionIncome
	case "退款", "refund":
		return DirectionRefund
	case "转账", "transfer":
		return DirectionTransfer
	default:
		return DirectionUnknown
	}
}

func targetTransactionType(direction string) string {
	switch direction {
	case DirectionExpense:
		return TargetTransactionExpense
	case DirectionIncome, DirectionRefund:
		return TargetTransactionIncome
	default:
		return TargetTransactionSkipped
	}
}

func markInvalid(row *PreviewRow, code string, message string) {
	row.DuplicateStatus = DuplicateStatusInvalid
	row.RowStatus = RowStatusFailed
	row.TargetTransactionType = TargetTransactionSkipped
	row.Error = &RowError{Code: code, Message: message}
}

func parseLocalTime(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("time is empty")
	}

	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, cst)
	if err != nil {
		return "", err
	}
	return parsed.Format(time.RFC3339), nil
}

func parseRFC3339Time(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("time is empty")
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", err
	}
	return parsed.Format(time.RFC3339), nil
}

func parseYuanToCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", "")
	value = strings.TrimPrefix(value, "¥")
	value = strings.TrimSuffix(value, "元")
	if value == "" {
		return 0, errors.New("amount is empty")
	}

	sign := int64(1)
	if strings.HasPrefix(value, "-") {
		sign = -1
		value = strings.TrimPrefix(value, "-")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid amount: %s", value)
	}

	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}

	var cents int64
	if len(parts) == 2 {
		centText := parts[1]
		if len(centText) > 2 {
			return 0, fmt.Errorf("invalid cent precision: %s", value)
		}
		for len(centText) < 2 {
			centText += "0"
		}
		cents, err = strconv.ParseInt(centText, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	return sign * (yuan*100 + cents), nil
}

func parseCents(value string) (int64, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" {
		return 0, errors.New("amount is empty")
	}
	return strconv.ParseInt(value, 10, 64)
}

func markSuspiciousRows(rows []PreviewRow) {
	for i := range rows {
		if rows[i].DuplicateStatus != DuplicateStatusNew || rows[i].RowStatus != RowStatusPending {
			continue
		}
		currentTime, err := time.Parse(time.RFC3339, rows[i].OccurredAt)
		if err != nil {
			continue
		}

		for j := 0; j < i; j++ {
			if rows[j].RowStatus != RowStatusPending {
				continue
			}
			previousTime, err := time.Parse(time.RFC3339, rows[j].OccurredAt)
			if err != nil {
				continue
			}
			if rows[i].Merchant == rows[j].Merchant &&
				rows[i].AmountCents == rows[j].AmountCents &&
				rows[i].Direction == rows[j].Direction &&
				math.Abs(currentTime.Sub(previousTime).Minutes()) <= 5 {
				rows[i].DuplicateStatus = DuplicateStatusSuspicious
				rows[i].SuspiciousReason = "同商户、同金额、时间相近"
				break
			}
		}
	}
}
