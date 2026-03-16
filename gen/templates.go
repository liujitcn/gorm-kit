package main

const dataFileTemplate = `package {{ .PackageName }}

import (
	"context"

	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"{{ .ModelImportPath }}"
	"{{ .QueryImportPath }}"
)

func init() {
	databaseGorm.RegisterMigrateModels(
{{- range .Tables }}
		new({{ $.ModelPackage }}.{{ .ModelName }}),
{{- end }}
	)
}

type contextTxKey struct{}

var txQueryKey = contextTxKey{}

type Data struct {
	query *{{ .QueryPackage }}.Query
}

// NewData 初始化数据访问对象，并构建默认查询入口。
func NewData(c *databaseGorm.Client) *Data {
	d := &Data{
		query: {{ .QueryPackage }}.Use(c.DB),
	}
	return d
}

// Transaction 定义事务执行能力，便于业务层按接口依赖。
type Transaction interface {
	Transaction(context.Context, func(ctx context.Context) error) error
}

// NewTransaction 创建事务执行器。
func NewTransaction(d *Data) Transaction {
	return d
}

// Transaction 在事务中执行传入函数，并将事务查询对象写入上下文。
func (d *Data) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.query.Transaction(func(tx *{{ .QueryPackage }}.Query) error {
		// 将事务态查询对象注入上下文，仓储层可透明复用当前事务。
		ctx = context.WithValue(ctx, txQueryKey, tx)
		return fn(ctx)
	})
}

// Query 返回当前上下文对应的查询入口；若存在事务则优先返回事务查询对象。
func (d *Data) Query(ctx context.Context) *{{ .QueryPackage }}.Query {
	if ctx == nil {
		return d.query
	}
	tx, ok := ctx.Value(txQueryKey).(*{{ .QueryPackage }}.Query)
	if ok {
		return tx
	}
	return d.query
}
`

const initFileTemplate = `package {{ .PackageName }}

import "github.com/google/wire"

// ProviderSet 定义 data 包依赖注入提供者集合。
var ProviderSet = wire.NewSet(
	NewData,
	NewTransaction,
{{- range .Tables }}
	New{{ .RepoName }}Repo,
{{- end }}
)
`

const repoFileTemplate = `package {{ .PackageName }}

import (
	"context"

	baseRepo "github.com/liujitcn/gorm-kit/repo"
	"{{ .ModelImportPath }}"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

// {{ .Table.RepoName }}Repo 定义 {{ .Table.ModelName }} 的基础仓储能力。
type {{ .Table.RepoName }}Repo struct {
	baseRepo.BaseRepo[{{ .ModelPackage }}.{{ .Table.ModelName }}]
	*Data
}

// New{{ .Table.RepoName }}Repo 创建 {{ .Table.ModelName }} 基础仓储实例。
func New{{ .Table.RepoName }}Repo(data *Data) *{{ .Table.RepoName }}Repo {
	base := baseRepo.NewBaseRepo[{{ .ModelPackage }}.{{ .Table.ModelName }}](
		func(ctx context.Context) gen.Dao {
			return new(data.Query(ctx).{{ .Table.ModelName }}.WithContext(ctx).DO)
		},
		func(ctx context.Context) field.Int64 {
{{- if .Table.HasCompositePrimaryKey }}
			// 联合主键场景默认使用第一个 int64 类型的主键字段。
{{- end }}
			return data.Query(ctx).{{ .Table.ModelName }}.{{ .Table.PrimaryKeyField }}
		},
		func(entity *{{ .ModelPackage }}.{{ .Table.ModelName }}) int64 {
{{- if .Table.HasCompositePrimaryKey }}
			// 联合主键场景默认使用实体上的第一个 int64 类型主键字段值。
{{- end }}
			return entity.{{ .Table.PrimaryKeyField }}
		},
	)
	return &{{ .Table.RepoName }}Repo{
		BaseRepo: base,
		Data:     data,
	}
}
`
