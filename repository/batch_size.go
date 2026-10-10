package repository

import (
	"reflect"
	"strings"
)

const (
	// 跨数据库兼容的单条 SQL 参数上限估算值，兼容 SQLite 常见默认限制。
	defaultMaxSQLVars = 999
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
		return min(len(list), defaultFallbackBatchSize)
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
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Pointer {
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
	for sf := range t.Fields() {
		if sf.Anonymous {
			et := sf.Type
			if et.Kind() == reflect.Pointer {
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
		ignored := false
		for _, tagOption := range strings.Split(gormTag, ";") {
			if tagOption == "-" || tagOption == "-:all" {
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}
		count++
	}
	return count
}

// countExportedInsertFields 在缺少 gorm tag 时，按导出字段数量做保守估算。
func countExportedInsertFields(t reflect.Type) int {
	count := 0
	for sf := range t.Fields() {
		if sf.Anonymous {
			et := sf.Type
			if et.Kind() == reflect.Pointer {
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
