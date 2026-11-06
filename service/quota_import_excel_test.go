package service

import (
	"bytes"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/xuri/excelize/v2"
)

func buildTestExcel(t *testing.T, headers []string, rows [][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	if err := f.SetSheetRow(sheet, "A1", &headers); err != nil {
		t.Fatalf("set header: %v", err)
	}
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			t.Fatalf("set row: %v", err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	return buf.Bytes()
}

func TestParseQuotaImportExcelFromSeatText(t *testing.T) {
	data := buildTestExcel(t,
		[]string{"部门对接人", "部门", "姓名", "账号", "订阅座席", "备注"},
		[][]string{
			{"刘椰平", "数字供应链部", "陈建威", "e-Jianwei.Chen2", "高级座席-542元/月（实际1000元额度）", "外包"},
			{"刘椰平", "数字供应链部", "孙洁飞", "e-Jiefei.Sun", "高级座席-542元/月（实际1000元额度）", "外包"},
			// 同账号重复出现，额度一致 → 去重
			{"刘椰平", "数字供应链部", "陈建威", "e-Jianwei.Chen2", "高级座席-542元/月（实际1000元额度）", "外包"},
			// 同账号不同额度 → 取最后一次出现
			{"刘椰平", "智能制造部", "王泽鹏", "e-Zepeng.Wang", "高级座席-542元/月（实际500元额度）", "外包"},
			{"刘椰平", "智能制造部", "王泽鹏", "e-Zepeng.Wang", "高级座席-542元/月（实际2000元额度）", "外包"},
			// 无 e- 前缀账号
			{"刘椰平", "数字供应链部", "周淼钟", "Miaozhong.Zhou", "高级座席-542元/月（实际1000元额度）", "外包"},
			// 账号为空的行应被跳过
			{"", "", "", "", "", ""},
		},
	)

	rows, err := ParseQuotaImportExcel(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("expected 4 unique rows, got %d: %+v", len(rows), rows)
	}
	byName := map[string]struct {
		quota    int
		amount   float64
		dup      bool
		hasIssue bool
	}{}
	for _, row := range rows {
		byName[row.Username] = struct {
			quota    int
			amount   float64
			dup      bool
			hasIssue bool
		}{row.Quota, row.AmountYuan, row.Duplicate, row.Issue != ""}
	}

	if got := byName["e-Jianwei.Chen2"]; got.quota != 500000000 || got.amount != 1000 || !got.dup {
		t.Errorf("duplicate same-amount row wrong: %+v", got)
	}
	if got := byName["e-Zepeng.Wang"]; got.quota != 1000000000 || got.amount != 2000 {
		t.Errorf("last-occurrence-wins wrong: %+v", got)
	}
	if got := byName["Miaozhong.Zhou"]; got.quota != 500000000 {
		t.Errorf("no-prefix account wrong: %+v", got)
	}
}

func TestParseQuotaImportExcelAmountColumnPriority(t *testing.T) {
	data := buildTestExcel(t,
		[]string{"账号", "订阅座席", "额度(元)"},
		[][]string{
			{"user-a", "高级座席-542元/月（实际1000元额度）", "88"},
			{"user-b", "", "12.5"},
			{"user-c", "高级座席-542元/月（实际1000元额度）", ""},
		},
	)

	rows, err := ParseQuotaImportExcel(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].AmountYuan != 88 || rows[0].Quota != 44000000 {
		t.Errorf("amount column should take priority: %+v", rows[0])
	}
	if rows[1].AmountYuan != 12.5 || rows[1].Quota != 6250000 {
		t.Errorf("decimal amount wrong: %+v", rows[1])
	}
	if rows[2].AmountYuan != 1000 {
		t.Errorf("empty amount cell should fall back to seat text: %+v", rows[2])
	}
}

func TestParseQuotaImportExcelIssues(t *testing.T) {
	data := buildTestExcel(t,
		[]string{"账号", "订阅座席"},
		[][]string{
			{"user-a", "座席文案里没有额度"},
		},
	)

	rows, err := ParseQuotaImportExcel(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 1 || rows[0].Issue == "" {
		t.Fatalf("expected issue row, got %+v", rows)
	}
	if rows[0].Quota != 0 {
		t.Errorf("unparseable amount should yield zero quota: %+v", rows[0])
	}
}

func TestParseQuotaImportExcelQuotaColumn(t *testing.T) {
	// 新模版（2026-10）：额度列名为「配额」，直接填具体额度数值
	data := buildTestExcel(t,
		[]string{"年月", "部门", "姓名", "账号", "配额", "备注"},
		[][]string{
			{"2026-11", "数字供应链部", "张三", "e-A.Example", "1000", "正常"},
			{"2026-11", "数字供应链部", "李四", "e-B.Example", "66.5", "小数"},
			{"2026-11", "数字供应链部", "王五", "e-C.Example", "", "空配额"},
		},
	)

	rows, err := ParseQuotaImportExcel(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].AmountYuan != 1000 || rows[0].Quota != 500000000 {
		t.Errorf("quota column parse wrong: %+v", rows[0])
	}
	if rows[1].AmountYuan != 66.5 || rows[1].Quota != 33250000 {
		t.Errorf("decimal quota wrong: %+v", rows[1])
	}
	if rows[2].AmountYuan != 0 || rows[2].Issue == "" {
		t.Errorf("empty quota should mark issue: %+v", rows[2])
	}
}

func TestParseQuotaImportExcelMissingAccountColumn(t *testing.T) {
	data := buildTestExcel(t,
		[]string{"姓名", "订阅座席"},
		[][]string{{"某人", "高级座席-542元/月（实际1000元额度）"}},
	)

	if _, err := ParseQuotaImportExcel(bytes.NewReader(data)); err == nil {
		t.Fatal("expected error for missing account column")
	}
}

func TestParseQuotaImportExcelHeaderAfterTitleRows(t *testing.T) {
	data := buildTestExcel(t,
		[]string{"XX部门2026年11月额度申请表"},
		[][]string{
			{"", "", ""},
			{"部门对接人", "部门", "姓名", "账号", "订阅座席", "备注"},
			{"刘椰平", "数字供应链部", "陈建威", "e-Jianwei.Chen2", "高级座席-542元/月（实际1000元额度）", "外包"},
		},
	)

	rows, err := ParseQuotaImportExcel(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 1 || rows[0].Username != "e-Jianwei.Chen2" || rows[0].Quota != 500000000 {
		t.Fatalf("header-after-title-rows parse wrong: %+v", rows)
	}
}

func TestParseQuotaImportExcelIgnoresStrategyColumn(t *testing.T) {
	// Excel 只服务月度清零用户：即使文件里带「清零策略」列，也不参与解析与生效
	data := buildTestExcel(t,
		[]string{"账号", "配额", "清零策略"},
		[][]string{
			{"user-a", "100", "每月清零"},
			{"user-b", "200", "周期清零"},
			{"user-c", "300", ""},
			{"user-d", "400", "随便写不该报错"},
		},
	)

	rows, err := ParseQuotaImportExcel(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	for _, row := range rows {
		if row.Issue != "" {
			t.Errorf("strategy column must not produce issues: %+v", row)
		}
	}
	if rows[0].Quota != 50000000 || rows[1].Quota != 100000000 || rows[3].Quota != 200000000 {
		t.Errorf("amounts must parse regardless of strategy column: %+v", rows)
	}
}

func TestComputeQuotaImportHashIgnoresStrategyColumn(t *testing.T) {
	rows := []model.QuotaImportPreviewRow{
		{Username: "user-a", AmountYuan: 100},
		{Username: "user-b", AmountYuan: 200},
	}

	if ComputeQuotaImportHash(rows) != ComputeQuotaImportHash(rows) {
		t.Fatal("hash must be stable for identical rows")
	}
	if ComputeQuotaImportHash(rows) != ComputeQuotaImportHash(append([]model.QuotaImportPreviewRow(nil), rows...)) {
		t.Fatal("hash must not depend on anything but account and amount")
	}
}

func TestBuildQuotaImportTemplateParsesBack(t *testing.T) {
	data, err := BuildQuotaImportTemplate()
	if err != nil {
		t.Fatalf("build template: %v", err)
	}
	rows, err := ParseQuotaImportExcel(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse generated template: %v", err)
	}
	// 示例行账号 test1 不存在，预览行保留但带 Issue 标记；关键是表头可定位、不报「未找到账号列」
	if len(rows) != 1 {
		t.Fatalf("expected 1 example row, got %d", len(rows))
	}
	if rows[0].Username != "test1" {
		t.Errorf("example account = %q, want test1", rows[0].Username)
	}
}
