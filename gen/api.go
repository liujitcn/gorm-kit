package gen

import "github.com/liujitcn/gorm-kit/gen/internal/config"

// ConfigOptions 描述从服务 data.yaml 批量生成数据库代码的参数。
type ConfigOptions = config.ConfigOptions

// GenerateConfig 读取服务配置并生成一个或多个数据源的代码。
func GenerateConfig(opts ConfigOptions) error {
	return config.GenerateConfig(opts)
}
