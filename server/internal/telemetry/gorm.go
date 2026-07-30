package telemetry

import (
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// GormPlugins 返回每个数据库实例独立使用的 OpenTelemetry 插件集合。
func GormPlugins() map[string]gorm.Plugin {
	plugin := tracing.NewPlugin(
		tracing.WithoutQueryVariables(),
		tracing.WithoutMetrics(),
	)
	return map[string]gorm.Plugin{
		plugin.Name(): plugin,
	}
}
