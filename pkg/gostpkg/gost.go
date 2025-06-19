package gostpkg

import (
	"github.com/go-gost/gost/cmd/gost"
)

// SetupWithConfig 加载 YAML 配置并启动服务
func SetupWithConfig(cfgFile string) error {
	// 复用 gost 的 CLI：等同于 go-gost/gost 执行 `gost -C cfgFile`
	gost.Cmd.SetArgs([]string{"-C", cfgFile})
	return gost.Cmd.Execute()
}
