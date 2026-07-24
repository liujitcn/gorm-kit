package repository

import (
	"reflect"
	"strings"
)

const (
	// 常见数据库单条 SQL 参数上限估算值（用于批量写入分批）。
	defaultMaxSQLVars = 65535
	// 自适应失败时的兜底批次。
	defaultFallbackBatchSize = 100
	// 防止单批过大导致 SQL 包体过大或事务过重。
	maxAutoBatchSize = 1000
)

// calcAutoBatchSize 根据模型字段数和 SQL 参数上限估算批量写入大小。
// 估算步骤：
// 1. 先估算单行插入字段数。
// 2. 用 maxVars/fieldCount 计算建议批次。
// 3. 最终批次受 [1, maxAutoBatchSize] 和 len(list) 约束。
func calcAutoBatchSize[T any](list []*T) int {
	if len(list) == 0 {
		// 空列表场景不会真正入库，返回最小合法批次即可。
		return 1
	}

	columnCount := estimateInsertColumnCount[T]()
	if columnCount <= 0 {
		return minInt(len(list), defaultFallbackBatchSize)
	}

	size := defaultMaxSQLVars / columnCount
	if size <= 0 {
		// 极端场景兜底，确保至少单条写入。
		size = 1
	}
	if size > maxAutoBatchSize {
		// 限制单批上限，避免单条 SQL 过大。
		size = maxAutoBatchSize
	}
	if size > len(list) {
		// 批次不能大于待写入数据量。
		size = len(list)
	}
	return size
}

// estimateInsertColumnCount 估算单行插入字段数。
// 优先使用 gorm tag 字段数量；若拿不到则退回导出字段数量估算。
func estimateInsertColumnCount[T any]() int {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return 0
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		// 非结构体类型无法按字段估算列数。
		return 0
	}

	gormTagged := countInsertFieldsByGormTag(t)
	if gormTagged > 0 {
		return gormTagged
	}
	return countExportedInsertFields(t)
}

// countInsertFieldsByGormTag 统计可写入的 gorm 标记字段数（递归展开匿名结构体）。
func countInsertFieldsByGormTag(t reflect.Type) int {
	count := 0
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Anonymous {
			et := sf.Type
			if et.Kind() == reflect.Ptr {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				count += countInsertFieldsByGormTag(et)
			}
			continue
		}
		if !sf.IsExported() {
			continue
		}
		gormTag := strings.TrimSpace(sf.Tag.Get("gorm"))
		if gormTag == "" {
			continue
		}
		if strings.Contains(gormTag, "-") {
			// gorm:"-" 表示忽略字段，不计入插入列数。
			continue
		}
		count++
	}
	return count
}

// countExportedInsertFields 在缺少 gorm tag 时，按导出字段数量做保守估算。
func countExportedInsertFields(t reflect.Type) int {
	count := 0
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Anonymous {
			et := sf.Type
			if et.Kind() == reflect.Ptr {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				count += countExportedInsertFields(et)
			}
			continue
		}
		if sf.IsExported() {
			count++
		}
	}
	return count
}

// minInt 返回两个整数中的较小值。
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
