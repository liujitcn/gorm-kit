package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/liujitcn/gorm-kit/gen"
)

// main 解析命令行参数并按服务配置生成数据库代码。
func main() {
	configPath := flag.String("config", "./configs/data.yaml", "服务数据配置文件")
	databaseName := flag.String("database", "", "配置文件中的数据源名称")
	table := flag.String("table", "", "指定表名，支持逗号分隔多个表")
	basePath := flag.String("base_path", "", "生成根目录，默认 gen")
	flag.Usage = buildUsage
	flag.Parse()

	err := gen.GenerateConfig(gen.ConfigOptions{
		ConfigPath:   *configPath,
		DatabaseName: *databaseName,
		Table:        *table,
		BasePath:     *basePath,
	})
	if err != nil {
		exitWithError(err)
	}
}

// buildUsage 构建命令行帮助输出。
func buildUsage() {
	_, _ = fmt.Fprintf(flag.CommandLine.Output(), "用法:\n")
	_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  %s [-config=./configs/data.yaml] [参数]\n\n", os.Args[0])
	flag.PrintDefaults()
	_, _ = fmt.Fprintf(flag.CommandLine.Output(), "\n说明:\n")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  未传 database 时使用默认数据源；传入 database 时使用对应命名数据源。")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  table 支持 user,user2；未传 database 时同样使用默认数据源。")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  所有数据源都输出到 base_path，可用 base_path 覆盖默认 gen。")
}

// exitWithError 输出错误并以失败状态退出。
func exitWithError(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
