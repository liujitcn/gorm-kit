package repo

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/liujitcn/gorm-kit/api/gen/go/conf"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

// BuildDao 根据 condition 的 search tag 动态构建查询条件。
// 读取规则：
// 1. tag 格式：`search:"type:eq;column:id;table:user"`。
// 2. table 为空时：默认用 model.TableName()，并且必须能在模型 gorm:"column:xxx" 中匹配到 column。
// 3. table 非空时：直接使用 table+column 构建字段，不强制模型列校验。
func BuildDao(dao gen.Dao, model, condition any) (gen.Dao, error) {
	// 先拿到模型元信息（结构体值 + 默认表名）。
	modelVal, modelTable, err := parseModelMeta(model)
	if err != nil {
		return nil, err
	}

	// condition 允许为空；为空时仅应用默认排序。
	if condition != nil {
		condVal := reflect.Indirect(reflect.ValueOf(condition))
		if condVal.IsValid() && condVal.Kind() == reflect.Struct {
			condType := condVal.Type()
			for i := 0; i < condVal.NumField(); i++ {
				condField := condType.Field(i)
				tag := parseSearchTag(condField.Tag.Get("search"))

				op, ok := parseSearchOperator(tag["type"])
				column := strings.TrimSpace(tag["column"])
				// operator 无效或 column 为空时，跳过该字段。
				if !ok || column == "" {
					continue
				}

				condFieldVal := condVal.Field(i)
				// 条件值为空（或无意义）时，跳过该字段。
				if !shouldApplyFilter(op, condFieldVal) {
					continue
				}

				// 统一构建查询字段表达式（包含 table 回退和列校验逻辑）。
				queryFieldVal, buildErr := buildQueryFieldFromTag(modelVal, modelTable, tag, column, op, condFieldVal)
				if buildErr != nil {
					return nil, buildErr
				}

				// ORDER 单独处理，不进入 Where 条件。
				if op == conf.Operator_ORDER {
					next, orderOK := applyOrder(dao, queryFieldVal, condFieldVal)
					if !orderOK {
						return nil, fmt.Errorf("invalid order value for %s", column)
					}
					dao = next
					continue
				}

				expr, exprOK := buildConditionExpr(queryFieldVal, op, condFieldVal)
				if !exprOK {
					return nil, fmt.Errorf("invalid condition op=%s column=%s", queryOperatorName(op), column)
				}
				dao = applyWhere(dao, expr)
			}
		}
	}

	// 每次都尝试追加默认排序（sort ASC -> updated_at DESC）。
	return applyDefaultOrderIfExists(dao, modelVal, modelTable), nil
}

// parseModelMeta 校验 model 并提取元信息。
// 要求 model 必须是 struct/*struct，且实现 TableName() string。
func parseModelMeta(model any) (reflect.Value, string, error) {
	if model == nil {
		return reflect.Value{}, "", errors.New("model is nil")
	}

	raw := reflect.ValueOf(model)
	typ := raw.Type()

	if typ.Kind() == reflect.Ptr {
		if typ.Elem().Kind() != reflect.Struct {
			return reflect.Value{}, "", errors.New("model must be struct")
		}
		if raw.IsNil() {
			// 允许 nil 指针：按类型构造零值用于反射字段读取。
			raw = reflect.New(typ.Elem())
		}
	} else if typ.Kind() != reflect.Struct {
		return reflect.Value{}, "", errors.New("model must be struct")
	}

	modelVal := reflect.Indirect(raw)
	if !modelVal.IsValid() || modelVal.Kind() != reflect.Struct {
		return reflect.Value{}, "", errors.New("model must be struct")
	}

	tableName, ok := callTableName(raw)
	if !ok {
		// 若方法定义在值接收者上，尝试在解引用值上获取。
		tableName, ok = callTableName(modelVal)
	}
	if !ok || strings.TrimSpace(tableName) == "" {
		return reflect.Value{}, "", errors.New("model must implement TableName() string")
	}

	return modelVal, strings.TrimSpace(tableName), nil
}

// callTableName 反射调用 TableName() string。
func callTableName(v reflect.Value) (string, bool) {
	method := v.MethodByName("TableName")
	if !method.IsValid() || method.Type().NumIn() != 0 || method.Type().NumOut() != 1 {
		return "", false
	}
	if method.Type().Out(0).Kind() != reflect.String {
		return "", false
	}
	out := method.Call(nil)
	if len(out) != 1 {
		return "", false
	}
	return out[0].String(), true
}

// parseSearchTag 解析 search tag，返回小写 key 的 map。
func parseSearchTag(raw string) map[string]string {
	result := make(map[string]string)
	if raw == "" {
		return result
	}

	for _, part := range strings.Split(raw, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(kv[0]))
		v := strings.TrimSpace(kv[1])
		if k != "" && v != "" {
			result[k] = v
		}
	}
	return result
}

// parseSearchOperator 仅按 proto 枚举名解析操作符（不兼容数字和别名）。
func parseSearchOperator(raw string) (conf.Operator, bool) {
	name := strings.ToUpper(strings.TrimSpace(raw))
	if name == "" {
		return 0, false
	}
	num, ok := conf.Operator_value[name]
	if !ok {
		return 0, false
	}
	return conf.Operator(num), true
}

// shouldApplyFilter 判断当前字段是否应该参与过滤。
func shouldApplyFilter(op conf.Operator, v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}

	if op == conf.Operator_ORDER {
		// ORDER 支持 string/int/uint；不识别时跳过。
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return false
			}
			v = v.Elem()
		}
		if v.Kind() == reflect.String {
			return strings.TrimSpace(v.String()) != ""
		}
		if isIntKind(v.Kind()) || isUintKind(v.Kind()) {
			// 允许 0（ASC）参与排序构建。
			return true
		}
		return false
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return !v.IsNil()
	case reflect.Slice, reflect.Array:
		return v.Len() > 0
	default:
		return !v.IsZero()
	}
}

// buildQueryFieldFromTag 按 tag 与模型信息构建查询字段表达式。
func buildQueryFieldFromTag(modelVal reflect.Value, modelTable string, tag map[string]string, column string, op conf.Operator, condFieldVal reflect.Value) (reflect.Value, error) {
	table := strings.TrimSpace(tag["table"])
	if table == "" {
		table = modelTable
		// table 为空时必须按 gorm column 严格校验。
		modelFieldVal := resolveModelFieldByGormColumn(modelVal, column)
		if !modelFieldVal.IsValid() {
			return reflect.Value{}, fmt.Errorf("column %s not found on model gorm tag", column)
		}
		return buildQueryFieldValue(table, column, modelFieldVal.Type()), nil
	}

	// table 非空时不做强校验；如果模型可找到字段则用字段类型优化表达式。
	modelFieldVal := resolveModelField(modelVal, column)
	if modelFieldVal.IsValid() {
		return buildQueryFieldValue(table, column, modelFieldVal.Type()), nil
	}
	// table 非空且模型不存在该字段时，按条件值推断字段类型，尽量兼容全部 operator。
	if inferredType, ok := inferFieldTypeFromCondition(op, condFieldVal); ok {
		return buildQueryFieldValue(table, column, inferredType), nil
	}
	return reflect.ValueOf(field.NewField(table, column)), nil
}

// inferFieldTypeFromCondition 基于条件值推断查询字段类型。
// 用于 table 非空且模型列无法匹配时的类型回退。
func inferFieldTypeFromCondition(op conf.Operator, v reflect.Value) (reflect.Type, bool) {
	if !v.IsValid() {
		return nil, false
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}

	// IN/NIN/BETWEEN 使用元素类型进行字段构建。
	if op == conf.Operator_IN || op == conf.Operator_NIN || op == conf.Operator_BETWEEN {
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			return v.Type().Elem(), true
		}
	}

	// IS_NULL/IS_NOT_NULL/ORDER 不依赖值类型，交给通用字段即可。
	if op == conf.Operator_IS_NULL || op == conf.Operator_IS_NOT_NULL || op == conf.Operator_ORDER {
		return nil, false
	}

	return v.Type(), true
}

// applyDefaultOrderIfExists 追加默认排序：
// 1. 有 sort 字段时按升序。
// 2. 否则有 updated_at 字段时按降序。
func applyDefaultOrderIfExists(dao gen.Dao, modelVal reflect.Value, table string) gen.Dao {
	sortField := resolveModelField(modelVal, "sort")
	if sortField.IsValid() {
		expr, ok := callNoArgMethod(buildQueryFieldValue(table, "sort", sortField.Type()), "Asc")
		if ok {
			return applyOrderExpr(dao, expr)
		}
	}

	updatedAtField := resolveModelField(modelVal, "updated_at")
	if updatedAtField.IsValid() {
		expr, ok := callNoArgMethod(buildQueryFieldValue(table, "updated_at", updatedAtField.Type()), "Desc")
		if ok {
			return applyOrderExpr(dao, expr)
		}
	}
	return dao
}

// resolveModelField 按 gorm column 或字段名匹配模型字段（仅匹配有 gorm tag 的字段）。
func resolveModelField(modelVal reflect.Value, column string) reflect.Value {
	if !modelVal.IsValid() || column == "" {
		return reflect.Value{}
	}

	fieldName := toModelFieldName(column)
	for i := 0; i < modelVal.NumField(); i++ {
		sf := modelVal.Type().Field(i)
		gormTag := strings.TrimSpace(sf.Tag.Get("gorm"))
		if gormTag == "" {
			continue
		}
		if gormColumn, ok := parseGormColumn(gormTag); ok && strings.EqualFold(gormColumn, column) {
			return modelVal.Field(i)
		}
		if strings.EqualFold(sf.Name, fieldName) || strings.EqualFold(sf.Name, column) {
			return modelVal.Field(i)
		}
	}
	return reflect.Value{}
}

// resolveModelFieldByGormColumn 仅按 gorm:"column:xxx" 严格匹配模型字段。
func resolveModelFieldByGormColumn(modelVal reflect.Value, column string) reflect.Value {
	if !modelVal.IsValid() || column == "" {
		return reflect.Value{}
	}
	for i := 0; i < modelVal.NumField(); i++ {
		sf := modelVal.Type().Field(i)
		gormTag := strings.TrimSpace(sf.Tag.Get("gorm"))
		if gormTag == "" {
			continue
		}
		gormColumn, ok := parseGormColumn(gormTag)
		if ok && strings.EqualFold(gormColumn, column) {
			return modelVal.Field(i)
		}
	}
	return reflect.Value{}
}

// parseGormColumn 从 gorm tag 中提取 column 值。
func parseGormColumn(gormTag string) (string, bool) {
	for _, part := range strings.Split(gormTag, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(kv[0]), "column") {
			column := strings.TrimSpace(kv[1])
			if column != "" {
				return column, true
			}
		}
	}
	return "", false
}

// toModelFieldName 将 snake_case 列名转为 CamelCase 字段名用于回退匹配。
func toModelFieldName(column string) string {
	column = strings.TrimSpace(column)
	if column == "" {
		return ""
	}
	parts := strings.FieldsFunc(column, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		runes := []rune(strings.ToLower(parts[i]))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}

// buildQueryFieldValue 按字段类型构建 gorm/gen 字段表达式。
func buildQueryFieldValue(table, column string, modelFieldType reflect.Type) reflect.Value {
	if modelFieldType.Kind() == reflect.Ptr {
		modelFieldType = modelFieldType.Elem()
	}
	if modelFieldType == reflect.TypeOf(time.Time{}) {
		return reflect.ValueOf(field.NewTime(table, column))
	}

	switch modelFieldType.Kind() {
	case reflect.String:
		return reflect.ValueOf(field.NewString(table, column))
	case reflect.Int:
		return reflect.ValueOf(field.NewInt(table, column))
	case reflect.Int8:
		return reflect.ValueOf(field.NewInt8(table, column))
	case reflect.Int16:
		return reflect.ValueOf(field.NewInt16(table, column))
	case reflect.Int32:
		return reflect.ValueOf(field.NewInt32(table, column))
	case reflect.Int64:
		return reflect.ValueOf(field.NewInt64(table, column))
	case reflect.Uint:
		return reflect.ValueOf(field.NewUint(table, column))
	case reflect.Uint8:
		return reflect.ValueOf(field.NewUint8(table, column))
	case reflect.Uint16:
		return reflect.ValueOf(field.NewUint16(table, column))
	case reflect.Uint32:
		return reflect.ValueOf(field.NewUint32(table, column))
	case reflect.Uint64:
		return reflect.ValueOf(field.NewUint64(table, column))
	case reflect.Float32:
		return reflect.ValueOf(field.NewFloat32(table, column))
	case reflect.Float64:
		return reflect.ValueOf(field.NewFloat64(table, column))
	case reflect.Bool:
		return reflect.ValueOf(field.NewBool(table, column))
	case reflect.Slice:
		if modelFieldType.Elem().Kind() == reflect.Uint8 {
			return reflect.ValueOf(field.NewBytes(table, column))
		}
	}
	// 未覆盖类型回退为通用字段表达式，保证构建不中断。
	return reflect.ValueOf(field.NewField(table, column))
}

// buildConditionExpr 根据 Operator 构建 where 表达式。
func buildConditionExpr(modelField reflect.Value, op conf.Operator, condField reflect.Value) (reflect.Value, bool) {
	if condField.Kind() == reflect.Ptr {
		condField = condField.Elem()
	}

	switch op {
	case conf.Operator_EQ:
		return callMethod(modelField, "Eq", condField)
	case conf.Operator_NEQ:
		return callMethod(modelField, "Neq", condField)
	case conf.Operator_GT:
		return callMethod(modelField, "Gt", condField)
	case conf.Operator_GTE:
		return callMethod(modelField, "Gte", condField)
	case conf.Operator_LT:
		return callMethod(modelField, "Lt", condField)
	case conf.Operator_LTE:
		return callMethod(modelField, "Lte", condField)
	case conf.Operator_LIKE:
		return callMethod(modelField, "Like", condField)
	case conf.Operator_NOT_LIKE:
		return callMethod(modelField, "NotLike", condField)
	case conf.Operator_IN:
		if condField.Kind() != reflect.Slice && condField.Kind() != reflect.Array {
			return reflect.Value{}, false
		}
		return callVariadicMethod(modelField, "In", condField)
	case conf.Operator_NIN:
		if condField.Kind() != reflect.Slice && condField.Kind() != reflect.Array {
			return reflect.Value{}, false
		}
		return callVariadicMethod(modelField, "NotIn", condField)
	case conf.Operator_IS_NULL:
		return callNoArgMethod(modelField, "IsNull")
	case conf.Operator_IS_NOT_NULL:
		return callNoArgMethod(modelField, "IsNotNull")
	case conf.Operator_BETWEEN:
		return buildBetweenExpr(modelField, condField)
	case conf.Operator_REGEXP:
		return callMethod(modelField, "Regexp", condField)
	case conf.Operator_CONTAINS:
		s, ok := toString(condField)
		if !ok {
			return reflect.Value{}, false
		}
		return callMethod(modelField, "Like", reflect.ValueOf(buildContainsPattern(s)))
	case conf.Operator_STARTS_WITH:
		s, ok := toString(condField)
		if !ok {
			return reflect.Value{}, false
		}
		return callMethod(modelField, "Like", reflect.ValueOf(buildStartsWithPattern(s)))
	case conf.Operator_ENDS_WITH:
		s, ok := toString(condField)
		if !ok {
			return reflect.Value{}, false
		}
		return callMethod(modelField, "Like", reflect.ValueOf(buildEndsWithPattern(s)))
	default:
		return reflect.Value{}, false
	}
}

// queryOperatorName 返回操作符名（用于错误信息）。
func queryOperatorName(op conf.Operator) string {
	return strings.ToLower(op.String())
}

// applyOrder 应用排序表达式。
// 排序值会优先按 conf.Sort 枚举解析，解析失败默认 ASC。
func applyOrder(dao gen.Dao, modelField reflect.Value, condField reflect.Value) (gen.Dao, bool) {
	if condField.Kind() == reflect.Ptr {
		condField = condField.Elem()
	}

	direction := parseSortDirection(condField)
	var expr reflect.Value
	var ok bool

	if direction == "desc" {
		expr, ok = callNoArgMethod(modelField, "Desc")
	} else {
		expr, ok = callNoArgMethod(modelField, "Asc")
	}
	if !ok {
		return dao, false
	}
	return applyOrderExpr(dao, expr), true
}

// parseSortDirection 解析排序方向，无法识别时默认 asc。
func parseSortDirection(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		raw := strings.TrimSpace(v.String())
		if raw == "" {
			return "asc"
		}
		// 仅按 Sort 枚举名解析（ASC/DESC）。
		if num, ok := conf.Sort_value[strings.ToUpper(raw)]; ok {
			return sortDirectionFromEnum(conf.Sort(num))
		}
		return "asc"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 支持直接传 Sort 枚举值（0/1）。
		return sortDirectionFromEnum(conf.Sort(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		// 支持直接传 Sort 枚举值（0/1）。
		return sortDirectionFromEnum(conf.Sort(v.Uint()))
	default:
		return "asc"
	}
}

// sortDirectionFromEnum 将 Sort 枚举转为 asc/desc。
func sortDirectionFromEnum(sort conf.Sort) string {
	if sort == conf.Sort_DESC {
		return "desc"
	}
	return "asc"
}

// buildBetweenExpr 构建 BETWEEN 表达式，要求 condField 为长度 2 的切片/数组。
func buildBetweenExpr(modelField reflect.Value, condField reflect.Value) (reflect.Value, bool) {
	if condField.Kind() != reflect.Slice && condField.Kind() != reflect.Array {
		return reflect.Value{}, false
	}
	if condField.Len() != 2 {
		return reflect.Value{}, false
	}
	left := condField.Index(0)
	right := condField.Index(1)
	return callTwoArgMethod(modelField, "Between", left, right)
}

// toString 将 reflect 值安全转为 string。
func toString(v reflect.Value) (string, bool) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.String {
		return "", false
	}
	return v.String(), true
}

// buildContainsPattern 构造包含匹配模式。
func buildContainsPattern(key string) string {
	return fmt.Sprintf("%%%s%%", key)
}

// buildStartsWithPattern 构造前缀匹配模式。
func buildStartsWithPattern(key string) string {
	return fmt.Sprintf("%s%%", key)
}

// buildEndsWithPattern 构造后缀匹配模式。
func buildEndsWithPattern(key string) string {
	return fmt.Sprintf("%%%s", key)
}

// callMethod 反射调用单参数方法，必要时做可转换类型适配。
func callMethod(target reflect.Value, name string, arg reflect.Value) (reflect.Value, bool) {
	method := target.MethodByName(name)
	if !method.IsValid() || method.Type().NumIn() != 1 {
		return reflect.Value{}, false
	}

	inType := method.Type().In(0)
	if !arg.Type().AssignableTo(inType) {
		if arg.Type().ConvertibleTo(inType) {
			arg = arg.Convert(inType)
		} else {
			return reflect.Value{}, false
		}
	}

	out := method.Call([]reflect.Value{arg})
	if len(out) == 0 {
		return reflect.Value{}, false
	}
	return out[0], true
}

// callTwoArgMethod 反射调用双参数方法。
func callTwoArgMethod(target reflect.Value, name string, arg1 reflect.Value, arg2 reflect.Value) (reflect.Value, bool) {
	method := target.MethodByName(name)
	if !method.IsValid() || method.Type().NumIn() != 2 {
		return reflect.Value{}, false
	}

	inType1 := method.Type().In(0)
	if !arg1.Type().AssignableTo(inType1) {
		if arg1.Type().ConvertibleTo(inType1) {
			arg1 = arg1.Convert(inType1)
		} else {
			return reflect.Value{}, false
		}
	}

	inType2 := method.Type().In(1)
	if !arg2.Type().AssignableTo(inType2) {
		if arg2.Type().ConvertibleTo(inType2) {
			arg2 = arg2.Convert(inType2)
		} else {
			return reflect.Value{}, false
		}
	}

	out := method.Call([]reflect.Value{arg1, arg2})
	if len(out) == 0 {
		return reflect.Value{}, false
	}
	return out[0], true
}

// callNoArgMethod 反射调用无参方法。
func callNoArgMethod(target reflect.Value, name string) (reflect.Value, bool) {
	method := target.MethodByName(name)
	if !method.IsValid() || method.Type().NumIn() != 0 {
		return reflect.Value{}, false
	}
	out := method.Call(nil)
	if len(out) == 0 {
		return reflect.Value{}, false
	}
	return out[0], true
}

// callVariadicMethod 反射调用 variadic 方法，并逐项做类型适配。
func callVariadicMethod(target reflect.Value, name string, args reflect.Value) (reflect.Value, bool) {
	method := target.MethodByName(name)
	if !method.IsValid() || !method.Type().IsVariadic() || method.Type().NumIn() != 1 {
		return reflect.Value{}, false
	}

	inElemType := method.Type().In(0).Elem()
	callArgs := make([]reflect.Value, 0, args.Len())
	for i := 0; i < args.Len(); i++ {
		arg := args.Index(i)
		if !arg.Type().AssignableTo(inElemType) {
			if arg.Type().ConvertibleTo(inElemType) {
				arg = arg.Convert(inElemType)
			} else {
				return reflect.Value{}, false
			}
		}
		callArgs = append(callArgs, arg)
	}

	out := method.Call(callArgs)
	if len(out) == 0 {
		return reflect.Value{}, false
	}
	return out[0], true
}

// applyWhere 将 reflect 表达式安全转换为 gen.Condition 并写入 dao。
func applyWhere(dao gen.Dao, expr reflect.Value) gen.Dao {
	if !expr.IsValid() {
		return dao
	}
	cond, ok := expr.Interface().(gen.Condition)
	if !ok {
		return dao
	}
	return dao.Where(cond)
}

// applyOrderExpr 将 reflect 表达式安全转换为 field.Expr 并写入 dao。
func applyOrderExpr(dao gen.Dao, expr reflect.Value) gen.Dao {
	if !expr.IsValid() {
		return dao
	}
	orderExpr, ok := expr.Interface().(field.Expr)
	if !ok {
		return dao
	}
	return dao.Order(orderExpr)
}

// isIntKind 判断是否为有符号整数类型。
func isIntKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

// isUintKind 判断是否为无符号整数类型。
func isUintKind(k reflect.Kind) bool {
	switch k {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}
