# gen

当前目录提供基于 `gorm/gen` 的代码生成入口，支持：

- 通过 `config.yaml` 加载生成配置
- 通过命令行参数覆盖配置文件中的单项字段
- 生成 `models`、`query` 与 `data`
- 每次生成 `data` 前自动清空目标目录，避免旧文件残留

## 配置示例

配置字段与 `options` 保持一致：

```yaml
driver: mysql
source: root:112233@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms
out_path: query
model_pkg_path: models
data_path: data
acronyms:
  api: API
  sku: SKU
```

## 使用方式

默认读取当前目录下的 `config.yaml`：

```bash
go run .
```

指定配置文件路径：

```bash
go run . -config ./config.yaml
```

使用命令行覆盖配置文件中的单项字段：

```bash
go run . -config ./config.yaml -set model_pkg_path=models1/tst -set out_path=query1/tet -set data_path=./data1
go run . -set source='root:123456@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms'
go run . -set acronyms.api=API -set acronyms.sku=SKU
```

## 覆盖项

当前支持以下 `-set key=value` 覆盖项：

- `driver`
- `source`
- `out_path`
- `model_pkg_path`
- `data_path`
- `acronyms.xxx`

## 生成规则

- `model_pkg_path`、`out_path`、`data_path` 会同时影响对应目录的生成结果
- `data` 中引用的 `models`、`query` 会跟随实际导入路径与目标包名变化
- `data` 包名取 `data_path` 最后一层目录名
- `data` 中每个 Repo 默认生成导出结构体，并内嵌通用 `BaseRepo` 与 `*Data`
- `data` 中 Repo 主键优先取模型声明顺序上的第一个 `int64` 主键字段；联合主键表同样按该规则生成，若不存在 `int64` 主键则回退到第一个主键字段
- 生成 `data` 前会先删除整个 `data_path`

## 默认值

- `driver` 默认 `mysql`
- `source` 默认使用内置 DSN
- `out_path` 默认 `query`
- `model_pkg_path` 默认 `models`
- `data_path` 默认 `data`
