package service

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/xuri/excelize/v2"
)

var availabilityCompensationExcelLocation = time.FixedZone("CST", 8*60*60)

type availabilityCompensationExcelStyles struct {
	header   int
	bold     int
	wrap     int
	number   int
	percent2 int
	percent3 int
	red      int
}

// BuildAvailabilityCompensationWorkbook 生成与人工对账导出结构一致的 Excel 报告。
func BuildAvailabilityCompensationWorkbook(result *model.AvailabilityCompensationResult) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	styles, err := newAvailabilityCompensationStyles(f)
	if err != nil {
		return nil, err
	}
	if err := f.SetSheetName(f.GetSheetName(0), "说明"); err != nil {
		return nil, err
	}
	for _, sheet := range []string{"业界对标", "版本1-全部错误归供应商", "版本2-剔除系统自身错误", "供应商汇总", "渠道对照表"} {
		if _, err := f.NewSheet(sheet); err != nil {
			return nil, err
		}
	}

	if err := writeAvailabilityCompensationNotes(f, result, styles); err != nil {
		return nil, err
	}
	if err := writeAvailabilityCompensationBenchmark(f, styles); err != nil {
		return nil, err
	}
	if err := writeAvailabilityCompensationVersionSheet(f, result, "版本1-全部错误归供应商", false, styles); err != nil {
		return nil, err
	}
	if err := writeAvailabilityCompensationVersionSheet(f, result, "版本2-剔除系统自身错误", true, styles); err != nil {
		return nil, err
	}
	if err := writeAvailabilityCompensationSupplierSheet(f, result, styles); err != nil {
		return nil, err
	}
	if err := writeAvailabilityCompensationChannelSheet(f, result, styles); err != nil {
		return nil, err
	}

	active, err := f.GetSheetIndex("说明")
	if err != nil {
		return nil, err
	}
	f.SetActiveSheet(active)
	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func newAvailabilityCompensationStyles(f *excelize.File) (availabilityCompensationExcelStyles, error) {
	header, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1F4E79"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return availabilityCompensationExcelStyles{}, err
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return availabilityCompensationExcelStyles{}, err
	}
	wrap, err := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Vertical: "top", WrapText: true}})
	if err != nil {
		return availabilityCompensationExcelStyles{}, err
	}
	number, err := f.NewStyle(&excelize.Style{CustomNumFmt: availabilityCompensationNumberFormat("#,##0")})
	if err != nil {
		return availabilityCompensationExcelStyles{}, err
	}
	percent2, err := f.NewStyle(&excelize.Style{CustomNumFmt: availabilityCompensationNumberFormat(`0.00"%"`)})
	if err != nil {
		return availabilityCompensationExcelStyles{}, err
	}
	percent3, err := f.NewStyle(&excelize.Style{CustomNumFmt: availabilityCompensationNumberFormat(`0.000"%"`)})
	if err != nil {
		return availabilityCompensationExcelStyles{}, err
	}
	red, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFC7CE"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return availabilityCompensationExcelStyles{}, err
	}
	return availabilityCompensationExcelStyles{
		header:   header,
		bold:     bold,
		wrap:     wrap,
		number:   number,
		percent2: percent2,
		percent3: percent3,
		red:      red,
	}, nil
}

func availabilityCompensationNumberFormat(format string) *string {
	return &format
}

func writeAvailabilityCompensationNotes(f *excelize.File, result *model.AvailabilityCompensationResult, styles availabilityCompensationExcelStyles) error {
	const sheet = "说明"
	startDate := availabilityCompensationDate(result.StartTimestamp)
	endExclusive := availabilityCompensationDate(result.EndTimestamp)
	endDate := availabilityCompensationDate(result.EndTimestamp - 1)
	periodState := "本周期已结束，为定格数"
	if result.Snapshot {
		periodState = "本周期尚未结束，当前为生成时点快照，最终以周期结束后的数据为准"
	}
	coverage := "所选周期内没有到达网关且可归属渠道的请求。"
	if result.DataStartTimestamp > 0 {
		coverage = fmt.Sprintf("网关日志库覆盖 %s ~ %s（北京时间）；仅含到达网关且可归属渠道的请求；报告生成于 %s。",
			availabilityCompensationDateTime(result.DataStartTimestamp),
			availabilityCompensationDateTime(result.DataEndTimestamp),
			availabilityCompensationDateTime(time.Now().Unix()))
	}
	notes := [][2]string{
		{"报告", fmt.Sprintf("供应商可用性赔偿测算（%s ~ %s）", startDate, endDate)},
		{"统计周期", fmt.Sprintf("%s 00:00 至 %s 00:00（北京时间），共 %d 分钟；%s", startDate, endExclusive, result.TotalMinutes, periodState)},
		{"数据覆盖", coverage},
		{"公式", "服务可用性 = (服务周期总分钟数 − 总错误分钟数) / 服务周期总分钟数 × 100%；不同模型独立统计"},
		{"版本1（全部错误归供应商）", "所有错误日志均计入失败，含 400/401/403 等用户侧错误——对索赔最有利口径"},
		{"版本2（剔除系统自身错误）", "仅计入服务端错误（5xx、404、408、429、529 及无状态码连接级），剔除 400/401/403/413 等用户侧错误——更可辩护口径"},
		{"错误分钟定义", "分钟错误率 = 该分钟失败请求数 ÷ 该分钟总请求数；≥50% 记为错误分钟（主口径）；同时给出“任一失败”参考列"},
		{"版本2口径说明", "剔除用户侧错误仅计服务端——与业界通行做法一致（百度千帆仅计服务端错误码，鉴权失败/欠费等不算有效请求）；版本2的分钟分母为成功请求 + 服务端错误"},
		{"粒度说明", "合同按模型独立统计；本报告按“模型×渠道”承载行统计并汇总到供应商——避免跨渠道重试把供应商失败稀释掉"},
		{"赔偿档位", "可用性>99.9% 无赔偿；99%≤可用性≤99.9% 赔月度服务费 5%；95%≤可用性<99% 赔 8%；<95% 赔 10%（档位按“错误率≥50%”口径的可用性判定，基数月度服务费由商务提供）"},
		{"注意", "无流量分钟按可用计；供应商级错误分钟按模型×渠道分钟行汇总，单模型 SLA 以版本表为准；渠道开通晚于周期起点时，认定时可与对方约定按实际服务期起算"},
	}
	for index, note := range notes {
		row := index + 1
		if err := f.SetCellValue(sheet, "A"+strconv.Itoa(row), note[0]); err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, "B"+strconv.Itoa(row), note[1]); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), styles.bold); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, "B"+strconv.Itoa(row), "B"+strconv.Itoa(row), styles.wrap); err != nil {
			return err
		}
		if err := f.SetRowHeight(sheet, row, 30); err != nil {
			return err
		}
	}
	if err := f.SetColWidth(sheet, "A", "A", 26); err != nil {
		return err
	}
	return f.SetColWidth(sheet, "B", "B", 110)
}

func writeAvailabilityCompensationBenchmark(f *excelize.File, styles availabilityCompensationExcelStyles) error {
	const sheet = "业界对标"
	headers := []string{"指标", "本报告口径", "业界标准（公开 SLA 条款）", "对本报告的含义"}
	rows := [][]interface{}{
		{"可用性公式", "(服务周期总分钟数 − 总错误分钟数) / 总分钟数 × 100%，不同模型独立统计", "百度千帆大模型服务 SLA 同一公式；阿里云系普遍采用“每5分钟错误率的月度平均”变体（每5分钟错误率=该5分钟失败请求数/有效总请求数）", "与合同按模型独立统计的约定完全一致；分钟/5分钟粒度差异不改变结论量级"},
		{"错误分钟判定", "分钟错误率 ≥ 50% 记为错误分钟（主口径）；另给“任一失败”参考列", "百度千帆：某分钟内有效请求超过10次且全部返回服务端错误码 InternalError，才记为错误分钟（100%全失败＋最低10次请求门槛）", "业界口径严于本报告（须全失败且设样本量门槛）；50% 阈值是待定标参数，认定前须与对方书面确认"},
		{"错误范围", "版本1=全部错误归供应商；版本2=仅服务端错误（5xx、404、408、429、529 及连接级）", "千帆仅将服务端错误码计入失败请求；身份验证/鉴权失败、欠费、黑客攻击等用户侧请求不算有效请求，不参与错误分钟判定", "业界默认接近版本2；建议以版本2为主张口径、版本1为谈判上限"},
		{"服务周期", "自然月；本报告总分钟数 = 周期天数 × 24 × 60", "千帆：一个服务周期为一个自然月，总分钟数=天数×24×60", "与业界一致；渠道开通晚于周期起点时，建议约定按实际服务期起算"},
		{"统计边界", "仅覆盖“到达我方网关”的请求", "千帆将“因平台服务故障导致未能到达服务器的请求”也计为失败请求（以平台侧记录认定）", "我方日志无法统计未到达请求；认定时建议约定以我方日志为基准、对方日志交叉复核"},
		{"赔偿档位", "合同：>99.9% 无赔偿；[99%,99.9%] 赔月度服务费 5%；[95%,99%) 赔 8%；<95% 赔 10%，上限当月服务费", "千帆：≥99.9% 承诺；[95%,99.9%) 赔 10%；<95% 赔 25%，上限25%，以代金券支付", "本合同档位更细、比例更低；档位判定取“≥50%”口径的可用性"},
		{"赔付申请时限", "以我方与供应商合同约定为准", "千帆：次月第5个工作日后可申请，月份结束后2个月内有效", "业界惯例可供流程参考"},
	}
	source := []interface{}{"来源", "百度千帆《服务可用性协议》：cloud.baidu.com/doc/qianfan/s/xmh4su51x", "阿里云《大模型服务平台百炼模型推理服务等级协议》2026-08-26 变更公告（2026-09-28 生效）：www.aliyun.com/notice/118594", "阿里云系“每5分钟错误率”条款示例：视觉智能开放平台服务等级协议（terms.aliyun.com）"}
	return writeAvailabilityCompensationTable(f, sheet, headers, append(rows, source), []float64{14, 46, 56, 56}, styles, nil, nil, nil, false, true)
}

func writeAvailabilityCompensationVersionSheet(f *excelize.File, result *model.AvailabilityCompensationResult, sheet string, serviceVersion bool, styles availabilityCompensationExcelStyles) error {
	failHeader := "错误请求(全部)"
	if serviceVersion {
		failHeader = "服务端错误"
	}
	rows := append([]model.AvailabilityCompensationRow(nil), result.Rows...)
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i].FailAllRequests, rows[j].FailAllRequests
		if serviceVersion {
			left, right = rows[i].FailServiceRequests, rows[j].FailServiceRequests
		}
		if left != right {
			return left > right
		}
		if rows[i].ChannelId != rows[j].ChannelId {
			return rows[i].ChannelId < rows[j].ChannelId
		}
		return rows[i].ModelName < rows[j].ModelName
	})
	headers := []string{"模型", "渠道ID", "渠道名称", "供应商", "成功请求", failHeader, "错误率",
		"错误分钟(≥50%)", "错误分钟(任一)", "可用性(≥50%)", "可用性(任一)", "适用档位", "赔偿比例"}
	tableRows := make([][]interface{}, 0, len(rows))
	for _, row := range rows {
		fail := row.FailAllRequests
		rate := row.ErrorRateAll
		errGe50 := row.ErrorMinutesGe50All
		errAny := row.ErrorMinutesAnyAll
		avGe50 := row.AvailabilityGe50All
		avAny := row.AvailabilityAnyAll
		tier := row.TierAll
		pct := row.CompensationPctAll
		if serviceVersion {
			fail = row.FailServiceRequests
			rate = row.ErrorRateService
			errGe50 = row.ErrorMinutesGe50Service
			errAny = row.ErrorMinutesAnyService
			avGe50 = row.AvailabilityGe50Service
			avAny = row.AvailabilityAnyService
			tier = row.TierService
			pct = row.CompensationPctService
		}
		tableRows = append(tableRows, []interface{}{
			row.ModelName, row.ChannelId, row.ChannelName, row.Supplier, row.OkRequests, fail,
			rate, errGe50, errAny, avGe50, avAny, availabilityCompensationTierLabel(tier), availabilityCompensationPctLabel(pct),
		})
	}
	numFmts := map[int]int{5: styles.number, 6: styles.number, 7: styles.percent2, 8: styles.number, 9: styles.number, 10: styles.percent3, 11: styles.percent3}
	redCols := map[int]bool{12: true, 13: true}
	redRow := func(row []interface{}) bool {
		label, _ := row[12].(string)
		return label != availabilityCompensationTierLabel(model.AvailabilityCompensationTierNone)
	}
	widths := []float64{24, 9, 30, 26, 14, 16, 10, 15, 15, 15, 15, 22, 10}
	return writeAvailabilityCompensationTable(f, sheet, headers, tableRows, widths, styles, numFmts, redCols, redRow, true, false)
}

func writeAvailabilityCompensationSupplierSheet(f *excelize.File, result *model.AvailabilityCompensationResult, styles availabilityCompensationExcelStyles) error {
	const sheet = "供应商汇总"
	headers := []string{"供应商", "成功请求", "错误请求(全部)", "错误率(全部)", "可用性v1(≥50%)", "档位v1", "赔偿比例v1",
		"服务端错误", "错误率(服务端)", "可用性v2(≥50%)", "档位v2", "赔偿比例v2"}
	rows := make([][]interface{}, 0, len(result.Suppliers))
	for _, row := range result.Suppliers {
		rows = append(rows, []interface{}{
			row.Supplier, row.OkRequests, row.FailAllRequests, row.ErrorRateAll, row.AvailabilityGe50All,
			availabilityCompensationTierLabel(row.TierAll), availabilityCompensationPctLabel(row.CompensationPctAll),
			row.FailServiceRequests, row.ErrorRateService, row.AvailabilityGe50Service,
			availabilityCompensationTierLabel(row.TierService), availabilityCompensationPctLabel(row.CompensationPctService),
		})
	}
	numFmts := map[int]int{2: styles.number, 3: styles.number, 4: styles.percent2, 5: styles.percent3, 8: styles.number, 9: styles.percent2, 10: styles.percent3}
	redCols := map[int]bool{6: true, 7: true, 11: true, 12: true}
	redRow := func(row []interface{}) bool {
		label, _ := row[5].(string)
		serviceLabel, _ := row[10].(string)
		none := availabilityCompensationTierLabel(model.AvailabilityCompensationTierNone)
		return label != none || serviceLabel != none
	}
	widths := []float64{28, 14, 16, 14, 18, 22, 12, 14, 14, 18, 22, 12}
	return writeAvailabilityCompensationTable(f, sheet, headers, rows, widths, styles, numFmts, redCols, redRow, true, false)
}

func writeAvailabilityCompensationChannelSheet(f *excelize.File, result *model.AvailabilityCompensationResult, styles availabilityCompensationExcelStyles) error {
	const sheet = "渠道对照表"
	headers := []string{"渠道ID", "渠道名称", "供应商", "状态"}
	rows := make([][]interface{}, 0, len(result.Channels))
	for _, channel := range result.Channels {
		rows = append(rows, []interface{}{channel.Id, channel.Name, channel.Supplier, availabilityCompensationChannelStatus(channel.Status)})
	}
	return writeAvailabilityCompensationTable(f, sheet, headers, rows, []float64{9, 42, 30, 12}, styles, nil, nil, nil, true, false)
}

func writeAvailabilityCompensationTable(
	f *excelize.File,
	sheet string,
	headers []string,
	rows [][]interface{},
	widths []float64,
	styles availabilityCompensationExcelStyles,
	numFmts map[int]int,
	redCols map[int]bool,
	redRow func([]interface{}) bool,
	filter bool,
	boldFirstColumn bool,
) error {
	headerRow := make([]interface{}, len(headers))
	for i, header := range headers {
		headerRow[i] = header
	}
	if err := f.SetSheetRow(sheet, "A1", &headerRow); err != nil {
		return err
	}
	lastColumn, err := excelize.ColumnNumberToName(len(headers))
	if err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A1", lastColumn+"1", styles.header); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheet, 1, 24); err != nil {
		return err
	}
	for rowIndex, row := range rows {
		excelRow := rowIndex + 2
		rowCopy := row
		if err := f.SetSheetRow(sheet, "A"+strconv.Itoa(excelRow), &rowCopy); err != nil {
			return err
		}
		highlight := redRow != nil && redRow(row)
		for columnIndex := range row {
			if numFmtStyle, ok := numFmts[columnIndex+1]; ok {
				cell, err := excelize.CoordinatesToCellName(columnIndex+1, excelRow)
				if err != nil {
					return err
				}
				if err := f.SetCellStyle(sheet, cell, cell, numFmtStyle); err != nil {
					return err
				}
			}
			if highlight && redCols[columnIndex+1] {
				cell, err := excelize.CoordinatesToCellName(columnIndex+1, excelRow)
				if err != nil {
					return err
				}
				if err := f.SetCellStyle(sheet, cell, cell, styles.red); err != nil {
					return err
				}
			}
		}
		if boldFirstColumn {
			cell, err := excelize.CoordinatesToCellName(1, excelRow)
			if err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, cell, cell, styles.bold); err != nil {
				return err
			}
		}
	}
	for columnIndex, width := range widths {
		column, err := excelize.ColumnNumberToName(columnIndex + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, column, column, width); err != nil {
			return err
		}
	}
	if err := f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	if filter && len(rows) > 0 {
		if err := f.AutoFilter(sheet, "A1:"+lastColumn+strconv.Itoa(len(rows)+1), nil); err != nil {
			return err
		}
	}
	return nil
}

func availabilityCompensationTierLabel(tier string) string {
	switch tier {
	case model.AvailabilityCompensationTierNone:
		return "无赔偿"
	case model.AvailabilityCompensationTier99:
		return "99%≤可用性≤99.9%"
	case model.AvailabilityCompensationTier95:
		return "95%≤可用性<99%"
	case model.AvailabilityCompensationTierBelow95:
		return "可用性<95%"
	default:
		return tier
	}
}

func availabilityCompensationPctLabel(pct float64) string {
	if pct <= 0 {
		return ""
	}
	return fmt.Sprintf("%.0f%%", pct)
}

func availabilityCompensationChannelStatus(status int) string {
	switch status {
	case common.ChannelStatusEnabled:
		return "启用"
	case common.ChannelStatusManuallyDisabled:
		return "手动停用"
	case common.ChannelStatusAutoDisabled:
		return "自动封禁"
	default:
		return strconv.Itoa(status)
	}
}

func availabilityCompensationDate(timestamp int64) string {
	return time.Unix(timestamp, 0).In(availabilityCompensationExcelLocation).Format("2006-01-02")
}

func availabilityCompensationDateTime(timestamp int64) string {
	return time.Unix(timestamp, 0).In(availabilityCompensationExcelLocation).Format("2006-01-02 15:04")
}
