package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/novohool/ggost/pkg/gostpkg"
)

func main() {
	// 1. 接收 -C gost.yaml 参数
	cfg := flag.String("C", "", "gost config file (yaml)")
	flag.Parse()

	// 2. 如果提供了 -C，启动 gost
	if *cfg != "" {
		go func() {
			if err := gostpkg.SetupWithConfig(*cfg); err != nil {
				log.Fatalf("gost start failed: %v", err)
			}
		}()
		// 等一秒确保 gost 已启动（可改为更智能方式）
		// time.Sleep(1 * time.Second)
	}

	// 3. 取接续的命令并执行
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("no command to run")
	}

	cmdName := args[0]
	cmdArgs := args[1:]
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "command failed: %v\n", err)
		os.Exit(1)
	}
}
