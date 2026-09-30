package generator

import (
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
		normalizeFieldGormTag(tag, "time.Time")
		if _, exists := tag["type"]; exists {
			t.Fatal("方言时间类型应移除 type 标签")
		}
		if tag["comment"][0] != "创建时间" {
			t.Fatal("其他标签不应被修改")
		}
	})
	t.Run("中立类型改写", func(t *testing.T) {
		tag := field.GormTag{"type": {"character varying(100)"}}
		normalizeFieldGormTag(tag, "string")
		if tag["type"][0] != "varchar(100)" {
			t.Fatalf("期望 varchar(100) 实际 %s", tag["type"][0])
		}
	})
	t.Run("布尔tinyint去标签", func(t *testing.T) {
		tag := field.GormTag{"type": {"tinyint(1)"}}
		normalizeFieldGormTag(tag, "bool")
		if _, exists := tag["type"]; exists {
			t.Fatal("布尔列应移除 type 标签")
		}
	})
	t.Run("无标签透传", func(t *testing.T) {
		tag := field.GormTag{"column": {"name"}}
		normalizeFieldGormTag(tag, "string")
		if _, exists := tag["type"]; exists {
			t.Fatal("不应凭空生成 type 标签")
		}
	})
}
