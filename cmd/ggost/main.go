package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-gost/core/chain"
	"github.com/go-gost/core/hop"
	"github.com/go-gost/core/listener"
	socks5h "github.com/go-gost/x/handler/socks/v5"
	"github.com/go-gost/x/listener/tcp"
	"github.com/novohool/ggost/pkg/gostpkg"
)

type NodeConfig struct {
	Addr      string `yaml:"addr"`
	Connector struct {
		Type string `yaml:"type"`
		Auth struct {
			Username string `yaml:"username"`
			Password string `yaml:"password"`
		} `yaml:"auth"`
	} `yaml:"connector"`
	Dialer struct {
		Type     string            `yaml:"type"`
		Metadata map[string]string `yaml:"metadata"`
	} `yaml:"dialer"`
}

type HopConfig struct {
	Nodes []NodeConfig `yaml:"nodes"`
}

type ChainConfig struct {
	Name string      `yaml:"name"`
	Hops []HopConfig `yaml:"hops"`
}

type HandlerConfig struct {
	Type  string `yaml:"type"`
	Chain string `yaml:"chain"`
}

type ListenerConfig struct {
	Type string `yaml:"type"`
}

type ServiceConfig struct {
	Name     string         `yaml:"name"`
	Addr     string         `yaml:"addr"`
	Handler  HandlerConfig  `yaml:"handler"`
	Listener ListenerConfig `yaml:"listener"`
}

type Config struct {
	Services []ServiceConfig `yaml:"services"`
	Chains   []ChainConfig   `yaml:"chains"`
}

func parseConfig(cfgFile string) (*Config, error) {
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func waitForPort(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", address)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for gost to listen on %s", address)
}

func main() {
	cfg := flag.String("C", "", "gost config file (yaml)")
	waitAddr := flag.String("wait", "127.0.0.1:1080", "wait for gost to listen on this address")
	flag.Parse()

	if *cfg == "" {
		log.Fatal("请使用 -C 参数指定 gost.yaml 配置文件")
	}

	go func() {
		log.Printf("正在启动 gost，配置文件: %s\n", *cfg)
		if err := gostpkg.StartGostWithConfig(*cfg); err != nil {
			log.Fatalf("gost 启动失败: %v", err)
		}
	}()

	log.Printf("等待 gost 监听端口 %s ...\n", *waitAddr)
	if err := waitForPort(*waitAddr, 5*time.Second); err != nil {
		log.Fatalf("gost 未能在指定时间内监听端口: %v", err)
	}
	log.Println("gost 启动成功！")

	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("未指定要执行的命令")
	}

	cmdName := args[0]
	cmdArgs := args[1:]
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "ALL_PROXY=socks5h://127.0.0.1:1080")

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "命令执行失败: %v\n", err)
		os.Exit(1)
	}
}
