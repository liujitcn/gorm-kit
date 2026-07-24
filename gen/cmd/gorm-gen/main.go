package main

import (
	"flag"
	"fmt"
	"os"

	generator "github.com/liujitcn/gorm-kit/gen"
)

// main 解析命令行参数并选择单库兼容模式或配置文件批量模式。
func main() {
	configPath := flag.String("config", "./configs/data.yaml", "服务数据配置文件")
	driverName := flag.String("driver", "", "数据库驱动，单库兼容模式使用")
	source := flag.String("source", "", "数据库连接串，传入后使用单库兼容模式")
	databaseName := flag.String("database", "", "配置文件中的数据源名称")
	table := flag.String("table", "", "指定表名，支持逗号分隔多个表")
	basePath := flag.String("base_path", "", "生成根目录，默认 gen")
	outPath := flag.String("out_path", "", "query 输出目录，仅单库兼容模式使用")
	modelPkgPath := flag.String("model_pkg_path", "", "model 包路径，仅单库兼容模式使用")
	dataPath := flag.String("data_path", "", "data 输出目录，仅单库兼容模式使用")
	flag.Usage = buildUsage
	flag.Parse()

	var err error
	if *source != "" {
		err = generateSingle(*driverName, *source, *table, *basePath, *outPath, *modelPkgPath, *dataPath, flagWasSet("config"), *databaseName)
	} else {
		err = generateFromConfig(*configPath, *databaseName, *table, *basePath, *driverName, *outPath, *modelPkgPath, *dataPath)
	}
	if err != nil {
		exitWithError(err)
	}
}

// generateSingle 使用旧参数执行单数据库生成。
func generateSingle(driverName string, source string, table string, basePath string, outPath string, modelPkgPath string, dataPath string, configSet bool, databaseName string) error {
	if configSet {
		return fmt.Errorf("config 与 source 不能同时使用")
	}
	if databaseName != "" {
		return fmt.Errorf("database 参数只能用于 config 模式")
	}
	options := make([]generator.Option, 0, 7)
	if driverName != "" {
		options = append(options, generator.WithDriver(driverName))
	}
	if source != "" {
		options = append(options, generator.WithSource(source))
	}
	if table != "" {
		options = append(options, generator.WithTable(table))
	}
	if basePath != "" {
		options = append(options, generator.WithBasePath(basePath))
	}
	if outPath != "" {
		options = append(options, generator.WithOutPath(outPath))
	}
	if modelPkgPath != "" {
		options = append(options, generator.WithModelPkgPath(modelPkgPath))
	}
	if dataPath != "" {
		options = append(options, generator.WithDataPath(dataPath))
	}
	_, err := generator.NewGen(options...).Generate()
	return err
}

// generateFromConfig 使用统一 data.yaml 生成一个或多个数据源。
func generateFromConfig(configPath string, databaseName string, table string, basePath string, driverName string, outPath string, modelPkgPath string, dataPath string) error {
	if driverName != "" {
		return fmt.Errorf("driver 参数只能用于 source 单库模式")
	}
	if outPath != "" || modelPkgPath != "" || dataPath != "" {
		return fmt.Errorf("config 多数据源模式只支持 base_path 输出配置")
	}
	return generator.GenerateConfig(generator.ConfigOptions{
		ConfigPath:   configPath,
		DatabaseName: databaseName,
		Table:        table,
		BasePath:     basePath,
	})
}

// buildUsage 构建命令行帮助输出。
func buildUsage() {
	_, _ = fmt.Fprintf(flag.CommandLine.Output(), "用法:\n")
	_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  %s -config=./configs/data.yaml [参数]\n", os.Args[0])
	_, _ = fmt.Fprintf(flag.CommandLine.Output(), "  %s -source=<dsn> [单库兼容参数]\n\n", os.Args[0])
	flag.PrintDefaults()
	_, _ = fmt.Fprintf(flag.CommandLine.Output(), "\n说明:\n")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  未传 source 时默认读取 ./configs/data.yaml。")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  未传 database 时合并生成 data.database(default) 与 data.databases。")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  未传 database 且只有 data.databases 时生成全部命名数据源。")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  table 支持 user,user2；多数据源模式必须同时指定 database。")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  config 多数据源模式输出到 gen/<数据源>/，可用 base_path 覆盖 gen。")
}

// flagWasSet 判断命令行是否显式传入指定参数。
func flagWasSet(name string) bool {
	found := false
	flag.Visit(func(item *flag.Flag) {
		if item.Name == name {
			found = true
		}
	})
	return found
}

// exitWithError 输出错误并以失败状态退出。
func exitWithError(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
