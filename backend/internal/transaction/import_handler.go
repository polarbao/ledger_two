package transaction

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

// CSVParseResponse 返回给前端的 CSV 解析原始数据结构
type CSVParseResponse struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// HandleParseCSV 处理 CSV 文件解析请求
// @brief 接收上传的 CSV 文件，自动检测并转换 GBK/UTF-8 字符集，解析出行列数据返回，不写入 DB
// @param w http.ResponseWriter 响应写入句柄
// @param r *http.Request 携带 multipart 文件的请求句柄
func (h *Handler) HandleParseCSV(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
		return
	}

	// 限制文件大小为 2MB (2 * 1024 * 1024 字节)
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		response.WriteError(w, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeImportFileInvalid, "上传的文件大小超过 2MB 限制"))
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		response.WriteError(w, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeImportFileInvalid, "获取上传文件失败，请提供 file 字段"))
		return
	}
	defer file.Close()

	// 校验后缀名是否为 .csv
	fileName := strings.ToLower(fileHeader.Filename)
	if !strings.HasSuffix(fileName, ".csv") {
		response.WriteError(w, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeImportFileInvalid, "只支持导入 CSV 格式的文件"))
		return
	}

	// 读取整个文件内容用于编码判定
	bytesData, err := io.ReadAll(file)
	if err != nil {
		response.WriteError(w, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeImportFileInvalid, "读取文件数据失败"))
		return
	}

	// 自动字符集解码适配：判断是否为 UTF-8，如果不是，尝试以 GBK 进行流转换
	var reader io.Reader = bytes.NewReader(bytesData)
	if !utf8.Valid(bytesData) {
		reader = transform.NewReader(bytes.NewReader(bytesData), simplifiedchinese.GBK.NewDecoder())
	}

	csvReader := csv.NewReader(reader)
	// 允许每行的列数是不一致的，防止解析微信/支付宝账单的描述或总结行报错
	csvReader.FieldsPerRecord = -1
	// 允许 CSV 中存在不规则的引号格式以提供最宽松的兼容
	csvReader.LazyQuotes = true

	allRecords, err := csvReader.ReadAll()
	if err != nil {
		response.WriteError(w, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeImportFileInvalid, fmt.Sprintf("解析 CSV 结构失败: %v", err)))
		return
	}

	if len(allRecords) == 0 {
		response.WriteError(w, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeImportFileInvalid, "CSV 文件内容为空"))
		return
	}

	// 定位真实的表头位置。微信/支付宝等账单开头经常有说明文字，我们需要定位包含核心列的行作为表头。
	headerIdx := -1
	for i, row := range allRecords {
		hasTime := false
		hasAmount := false
		for _, col := range row {
			colClean := strings.TrimSpace(col)
			// 匹配包含“时间”/“日期”和“金额”/“收/支”/“收/付款”等组合的行作为表头
			if strings.Contains(colClean, "时间") || strings.Contains(colClean, "日期") {
				hasTime = true
			}
			if strings.Contains(colClean, "金额") || strings.Contains(colClean, "收/支") || strings.Contains(colClean, "收/付款人") {
				hasAmount = true
			}
		}
		if hasTime && hasAmount {
			headerIdx = i
			break
		}
	}

	var headers []string
	var dataRows [][]string

	if headerIdx != -1 {
		rawHeaders := allRecords[headerIdx]
		for _, h := range rawHeaders {
			headers = append(headers, strings.TrimSpace(h))
		}

		// 从表头下一行开始做为数据行
		for i := headerIdx + 1; i < len(allRecords); i++ {
			row := allRecords[i]
			if len(row) == 0 {
				continue
			}

			// 清理并检测是否为空行或无效干扰行
			isEmptyOrSummary := true
			var cleanRow []string
			for _, val := range row {
				cleanVal := strings.TrimSpace(val)
				cleanRow = append(cleanRow, cleanVal)
				if cleanVal != "" {
					isEmptyOrSummary = false
				}
			}

			if isEmptyOrSummary {
				continue
			}

			// 排除微信/支付宝结尾说明或汇总统计干扰行
			if len(cleanRow) < len(headers)/2 {
				continue
			}
			if strings.HasPrefix(cleanRow[0], "导出时间") || strings.HasPrefix(cleanRow[0], "生成时间") || strings.HasPrefix(cleanRow[0], "数据条数") || strings.HasPrefix(cleanRow[0], "-----------") {
				continue
			}

			dataRows = append(dataRows, cleanRow)
		}
	} else {
		// 若找不到明显表头，默认以非空的第一行为表头
		rawHeaders := allRecords[0]
		for _, h := range rawHeaders {
			headers = append(headers, strings.TrimSpace(h))
		}
		for i := 1; i < len(allRecords); i++ {
			row := allRecords[i]
			if len(row) == 0 {
				continue
			}
			var cleanRow []string
			for _, val := range row {
				cleanRow = append(cleanRow, strings.TrimSpace(val))
			}
			dataRows = append(dataRows, cleanRow)
		}
	}

	response.JSON(w, http.StatusOK, CSVParseResponse{
		Headers: headers,
		Rows:    dataRows,
	})
}

// HandleAnalyzeImport 处理待导入 CSV 数据去重分析
func (h *Handler) HandleAnalyzeImport(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
		return
	}

	var req AnalyzeImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "请求体格式无效")
		return
	}

	res, err := h.service.AnalyzeImport(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleCommitImport 处理 CSV 账单批量事务确认写入
func (h *Handler) HandleCommitImport(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
		return
	}

	var req CommitImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "请求体格式无效")
		return
	}

	if req.Filename == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "导入文件名 filename 不能为空")
		return
	}

	if len(req.Items) == 0 {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "导入账单列表不能为空")
		return
	}

	err := h.service.CommitImport(r.Context(), currentUserID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

