package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	_ "github.com/liujitcn/kratos-kit/database/gorm/driver/mysql"
)

// main 解析与 options 对齐的命令行参数并执行代码生成。
func main() {
	defaultOpts := defaultOptions()
	driver := flag.String("driver", "", "数据库驱动")
	source := flag.String("source", "", "数据库连接串")
	basePath := flag.String("base_path", "", "统一基础路径，例如 test")
	outPath := flag.String("out_path", "", "query 输出目录")
	modelPkgPath := flag.String("model_pkg_path", "", "model 包路径")
	dataPath := flag.String("data_path", "", "data 输出目录")
	flag.Usage = buildUsage(defaultOpts)
	flag.Parse()

	opts := defaultOpts
	WithDriver(*driver)(&opts)
	WithSource(*source)(&opts)
	WithBasePath(*basePath)(&opts)
	WithOutPath(*outPath)(&opts)
	WithModelPkgPath(*modelPkgPath)(&opts)
	WithDataPath(*dataPath)(&opts)
	// 只要传入 base_path，就统一为最终路径增加前缀，包括显式传入的路径参数。
	opts.ApplyBasePath()
	if err := validateOptions(opts); err != nil {
		flag.Usage()
		log.Fatal(err)
	}

	g := &Gen{opts: opts}
	tables, err := g.Generate()
	if err != nil {
		log.Fatal(err)
	}
	if err = generateModelCommentFile(opts, tables); err != nil {
		log.Fatal(err)
	}
	if err = generateDataFiles(opts, tables); err != nil {
		log.Fatal(err)
	}
}

// buildUsage 构建命令行帮助输出，明确参数含义、必填项与示例。
func buildUsage(defaultOpts options) func() {
	return func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "用法:\n")
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  %s -source=<dsn> [参数]\n\n", os.Args[0])
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "参数:\n")
		flag.PrintDefaults()
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "\n说明:\n")
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  -source 为必填项。\n")
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  -driver 默认为 %q。\n", defaultOpts.driver)
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  -base_path 会统一为 models、query、data 的最终路径增加目录前缀。\n")
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  -out_path 默认为 %q。\n", defaultOpts.outPath)
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  -model_pkg_path 默认为 %q。\n", defaultOpts.modelPkgPath)
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  -data_path 默认为 %q。\n", defaultOpts.dataPath)
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "\n示例:\n")
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  %s -source='root:123456@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms'\n", os.Args[0])
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  %s -source='root:123456@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms' -base_path=test\n", os.Args[0])
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  %s -source='root:123456@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms' -base_path=.server/pkg -out_path=query1/tet -model_pkg_path=models1/tst -data_path=./data1\n", os.Args[0])
	}
}

// validateOptions 校验运行生成器所需的关键参数。
func validateOptions(opts options) error {
	if strings.TrimSpace(opts.source) == "" {
		return fmt.Errorf("source 参数不能为空")
	}
	return nil
}
