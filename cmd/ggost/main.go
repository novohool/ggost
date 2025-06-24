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
	"time"

	"github.com/go-gost/core/chain"
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

// buildChain 构建代理链
func buildChain(chainCfg ChainConfig) (*chain.Chain, error) {
	// 转换为 gostpkg 中的类型
	gostChainCfg := gostpkg.ChainConfig{
		Name: chainCfg.Name,
		Hops: make([]gostpkg.HopConfig, len(chainCfg.Hops)),
	}
	
	for i, hop := range chainCfg.Hops {
		gostChainCfg.Hops[i] = gostpkg.HopConfig{
			Nodes: make([]gostpkg.NodeConfig, len(hop.Nodes)),
		}
		
		for j, node := range hop.Nodes {
			gostChainCfg.Hops[i].Nodes[j] = gostpkg.NodeConfig{
				Addr: node.Addr,
				Connector: struct {
					Type string `yaml:"type"`
					Auth struct {
						Username string `yaml:"username"`
						Password string `yaml:"password"`
					} `yaml:"auth"`
				}{
					Type: node.Connector.Type,
					Auth: struct {
						Username string `yaml:"username"`
						Password string `yaml:"password"`
					}{
						Username: node.Connector.Auth.Username,
						Password: node.Connector.Auth.Password,
					},
				},
				Dialer: struct {
					Type     string            `yaml:"type"`
					Metadata map[string]string `yaml:"metadata"`
				}{
					Type:     node.Dialer.Type,
					Metadata: node.Dialer.Metadata,
				},
			}
		}
	}
	
	return gostpkg.BuildChain(gostChainCfg)
}

// startService 启动服务
func startService(svcCfg ServiceConfig, chains map[string]*chain.Chain) error {
	// 转换为 gostpkg 中的类型
	gostSvcCfg := gostpkg.ServiceConfig{
		Name: svcCfg.Name,
		Addr: svcCfg.Addr,
		Handler: gostpkg.HandlerConfig{
			Type:  svcCfg.Handler.Type,
			Chain: svcCfg.Handler.Chain,
		},
		Listener: gostpkg.ListenerConfig{
			Type: svcCfg.Listener.Type,
		},
	}
	
	return gostpkg.StartService(gostSvcCfg, chains)
}

func StartGostWithConfig(cfgFile string) error {
	// 读取并解析配置文件
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 创建链映射
	chains := make(map[string]*chain.Chain)
	for _, chainCfg := range cfg.Chains {
		c, err := buildChain(chainCfg)
		if err != nil {
			return fmt.Errorf("构建链 %s 失败: %v", chainCfg.Name, err)
		}
		chains[chainCfg.Name] = c
	}

	// 启动所有服务
	for _, svcCfg := range cfg.Services {
		if err := startService(svcCfg, chains); err != nil {
			return fmt.Errorf("启动服务 %s 失败: %v", svcCfg.Name, err)
		}
		log.Printf("服务 %s 已启动，监听地址: %s", svcCfg.Name, svcCfg.Addr)
	}

	return nil
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
		if err := StartGostWithConfig(*cfg); err != nil {
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
