package generator

import (
	"strconv"
	"testing"

	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	"gorm.io/gen/field"
)

// TestDriverFieldKind 验证 gorm-gen Go 类型到共用映射语义类别的归纳。
func TestDriverFieldKind(t *testing.T) {
	cases := []struct {
		goType string
		kind   driver.FieldKind
	}{
		{goType: "string", kind: driver.FieldKindString},
		{goType: "time.Time", kind: driver.FieldKindTime},
		{goType: "[]byte", kind: driver.FieldKindBytes},
		{goType: "int64", kind: driver.FieldKindInt},
		{goType: "bool", kind: driver.FieldKindBool},
		{goType: "uuid.UUID", kind: driver.FieldKindOther},
	}
	for _, testCase := range cases {
		if kind := driverFieldKind(testCase.goType); kind != testCase.kind {
			t.Fatalf("Go 类型 %s 期望 %v 实际 %v", testCase.goType, testCase.kind, kind)
		}
	}
}

// TestNormalizeFieldGormTag 验证字段 type 标签的改写与移除，且不影响其他标签。
func TestNormalizeFieldGormTag(t *testing.T) {
	t.Run("方言时间类型去标签", func(t *testing.T) {
		tag := field.GormTag{"type": {"datetime(3)"}, "comment": {"创建时间"}}
		normalizeFieldGormTag(tag, "time.Time", "base_demo", "created_at", nil)
		if _, exists := tag["type"]; exists {
			t.Fatal("方言时间类型应移除 type 标签")
		}
		if tag["comment"][0] != "创建时间" {
			t.Fatal("其他标签不应被修改")
		}
	})
	t.Run("中立类型改写", func(t *testing.T) {
		tag := field.GormTag{"type": {"character varying(100)"}}
		normalizeFieldGormTag(tag, "string", "base_demo", "name", nil)
		if tag["type"][0] != "varchar(100)" {
			t.Fatalf("期望 varchar(100) 实际 %s", tag["type"][0])
		}
	})
	t.Run("布尔tinyint去标签", func(t *testing.T) {
		tag := field.GormTag{"type": {"tinyint(1)"}}
		normalizeFieldGormTag(tag, "bool", "base_demo", "enabled", nil)
		if _, exists := tag["type"]; exists {
			t.Fatal("布尔列应移除 type 标签")
		}
	})
	t.Run("无标签透传", func(t *testing.T) {
		tag := field.GormTag{"column": {"name"}}
		normalizeFieldGormTag(tag, "string", "base_demo", "name", nil)
		if _, exists := tag["type"]; exists {
			t.Fatal("不应凭空生成 type 标签")
		}
	})
	t.Run("PG检查约束找回二进制长度", func(t *testing.T) {
		tag := field.GormTag{"type": {"bytea"}, "column": {"digest"}}
		sizes := map[string]map[string]int{"base_redact_storage_value": {"digest": 32}}
		normalizeFieldGormTag(tag, "[]byte", "base_redact_storage_value", "digest", sizes)
		if values := tag["size"]; len(values) != 1 || values[0] != "32" {
			t.Fatalf("size 标签期望 32 实际 %v", values)
		}
		if _, exists := tag["type"]; exists {
			t.Fatal("type 标签应被移除")
		}
	})
	t.Run("无约束时二进制列不带长度", func(t *testing.T) {
		tag := field.GormTag{"type": {"bytea"}}
		normalizeFieldGormTag(tag, "[]byte", "base_demo", "content", nil)
		if _, exists := tag["size"]; exists {
			t.Fatal("无约束记录时不应生成 size 标签")
		}
	})
}

// TestBinarySizeDefPattern 验证长度检查约束定义的解析覆盖不同拼写。
func TestBinarySizeDefPattern(t *testing.T) {
	cases := []struct {
		def    string
		column string
		size   int
	}{
		{def: `CHECK (octet_length("digest") <= 32)`, column: "digest", size: 32},
		{def: `CHECK (octet_length(digest) <= 1023)`, column: "digest", size: 1023},
		{def: `CHECK (((octet_length("credential_id")) <= 1023))`, column: "credential_id", size: 1023},
		{def: `CHECK ((digest IS NOT NULL))`, column: "", size: 0},
	}
	for _, testCase := range cases {
		matches := binarySizeDefPattern.FindStringSubmatch(testCase.def)
		if testCase.size == 0 {
			if matches != nil {
				t.Fatalf("%s 不应匹配", testCase.def)
			}
			continue
		}
		if matches == nil || matches[1] != testCase.column {
			t.Fatalf("%s 列名解析失败", testCase.def)
		}
		if value, _ := strconv.Atoi(matches[2]); value != testCase.size {
			t.Fatalf("%s 长度期望 %d 实际 %s", testCase.def, testCase.size, matches[2])
		}
	}
}
