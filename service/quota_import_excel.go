package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/xuri/excelize/v2"
)

const (
	quotaImportMaxRows  = 5000
	quotaImportUnitYuan = 500000 // 基准 CNY：1 元 = 500000 quota
)

// BuildQuotaImportTemplate 运行时生成标准额度申请模版（与解析列定位保持一致）。
// 表头：年月/部门/姓名/账号/配额/备注，第二行为填写示例；程序化生成以避免
// 在仓库中维护会被办公软件写入文档追踪元数据的二进制模版文件。
func BuildQuotaImportTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	headers := []string{"年月", "部门", "姓名", "账号", "配额", "备注"}
	for i, h := range headers {
		if err := f.SetCellValue("Sheet1", fmt.Sprintf("%c1", 'A'+i), h); err != nil {
			return nil, err
		}
	}
	example := []string{"xxxx-xx", "xxxx", "用户1", "test1", "xxxx", "xxxx"}
	for i, v := range example {
		if err := f.SetCellValue("Sheet1", fmt.Sprintf("%c2", 'A'+i), v); err != nil {
			return nil, err
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("生成模版失败: %w", err)
	}
	return buf.Bytes(), nil
}

// 订阅座席文案中的实际额度，如“高级座席-542元/月（实际1000元额度）”
var quotaAmountPattern = regexp.MustCompile(`实际\s*([0-9]+(?:\.[0-9]+)?)\s*元额度`)

// ParseQuotaImportExcel 解析上传的额度 Excel，返回按“账号最终生效”去重后的行。
// 列定位基于表头：账号(必需)、额度(元)(可选，优先)、订阅座席(额度文案兜底来源)。
// 本流程只服务月度清零用户：只做“按名单设置额度”，不承载任何清零与策略设置（清零由月度/周期清零任务独立处理）。
func ParseQuotaImportExcel(r io.Reader) ([]model.QuotaImportPreviewRow, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("无法解析 Excel 文件: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel 文件中没有工作表")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("无法读取工作表: %w", err)
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("Excel 中没有数据行")
	}

	// 表头定位：兼容文件在表头前带标题行/空行，扫描前 5 行
	accountCol := -1
	amountCol := -1
	seatCol := -1
	headerRowIndex := -1
	scanLimit := len(rows)
	if scanLimit > 5 {
		scanLimit = 5
	}
	for r := 0; r < scanLimit; r++ {
		for i, cell := range rows[r] {
			header := strings.TrimSpace(cell)
			switch header {
			case "账号", "用户名", "username":
				accountCol = i
			case "配额", "额度(元)", "额度（元）", "额度":
				amountCol = i
			case "订阅座席", "订阅坐席":
				seatCol = i
			}
		}
		if accountCol >= 0 {
			headerRowIndex = r
			break
		}
	}
	if headerRowIndex < 0 {
		return nil, fmt.Errorf("未找到「账号」列，请使用标准模版")
	}

	type parsedRow struct {
		rowNumber int
		username  string
		amount    float64
	}
	parsed := make([]parsedRow, 0, len(rows))
	parseIssue := map[int]string{}

	for i := headerRowIndex + 1; i < len(rows) && len(parsed) < quotaImportMaxRows; i++ {
		cells := rows[i]
		username := ""
		if accountCol < len(cells) {
			username = strings.TrimSpace(cells[accountCol])
		}
		if username == "" {
			continue
		}

		amount, issue := extractAmount(cells, amountCol, seatCol)
		if issue != "" {
			parseIssue[i+1] = issue
		}
		parsed = append(parsed, parsedRow{rowNumber: i + 1, username: username, amount: amount})
	}

	// 同账号多次出现：取最后一次出现的额度；额度不一致则标记
	final := map[string]parsedRow{}
	dupAmountDiff := map[string]bool{}
	occurrence := map[string]int{}
	for _, row := range parsed {
		key := strings.ToLower(row.username)
		if prev, ok := final[key]; ok && prev.amount != row.amount {
			dupAmountDiff[key] = true
		}
		occurrence[key]++
		final[key] = row
	}

	result := make([]model.QuotaImportPreviewRow, 0, len(final))
	for _, row := range parsed {
		key := strings.ToLower(row.username)
		last, ok := final[key]
		if !ok || last.rowNumber != row.rowNumber {
			continue // 非最后一次出现，跳过（最终生效行才进入结果）
		}
		preview := model.QuotaImportPreviewRow{
			RowNumber:  row.rowNumber,
			Username:   row.username,
			AmountYuan: row.amount,
			Duplicate:  occurrence[key] > 1,
			Issue:      parseIssue[row.rowNumber],
		}
		if dupAmountDiff[key] {
			preview.Issue = joinIssue(preview.Issue, "同账号存在多个不同额度，以最后一次出现为准")
		}
		if preview.AmountYuan > 0 {
			preview.Quota = int(math.Round(row.amount * float64(quotaImportUnitYuan)))
		}
		result = append(result, preview)
	}
	return result, nil
}

// FillQuotaPreviewUserState 匹配系统用户，回填 user_id、当前额度与是否非月度策略；
// 只对"账号不存在/非月度清零"标记跳过（Excel 只服务月度清零用户，与角色无关）
func FillQuotaPreviewUserState(rows []model.QuotaImportPreviewRow) error {
	userMap, err := model.UserMapByUsername()
	if err != nil {
		return err
	}
	for i := range rows {
		if user, ok := userMap[strings.ToLower(rows[i].Username)]; ok && user.Id > 0 {
			rows[i].UserId = user.Id
			rows[i].Matched = true
			rows[i].CurrentQuota = user.Quota
			if user.EffectiveQuotaResetMode() != operation_setting.QuotaResetModeMonthly {
				rows[i].NonMonthly = true
				rows[i].Issue = joinIssue(rows[i].Issue, "非月度清零策略，Excel 将跳过")
			}
		} else {
			rows[i].Issue = joinIssue(rows[i].Issue, "账号不存在")
		}
	}
	return nil
}

// ComputeQuotaImportHash 计算解析结果的内容指纹，执行时校验与预览一致
func ComputeQuotaImportHash(rows []model.QuotaImportPreviewRow) string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%s=%.2f", strings.ToLower(row.Username), row.AmountYuan))
	}
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

func extractAmount(cells []string, amountCol, seatCol int) (float64, string) {
	if amountCol >= 0 && amountCol < len(cells) {
		v := strings.TrimSpace(cells[amountCol])
		if v != "" {
			if amount, err := strconv.ParseFloat(strings.TrimSuffix(v, "元"), 64); err == nil {
				return amount, ""
			}
			return 0, "额度列无法解析为数字"
		}
	}
	if seatCol >= 0 && seatCol < len(cells) {
		if m := quotaAmountPattern.FindStringSubmatch(cells[seatCol]); m != nil {
			amount, err := strconv.ParseFloat(m[1], 64)
			if err == nil {
				return amount, ""
			}
		}
		return 0, "未能从「订阅座席」解析额度（需含“实际X元额度”）"
	}
	return 0, "缺少额度来源（无「额度(元)」列或「订阅座席」列）"
}

func joinIssue(existing, add string) string {
	if existing == "" {
		return add
	}
	return existing + "；" + add
}
