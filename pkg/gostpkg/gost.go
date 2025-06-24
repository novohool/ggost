package gostpkg

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"

	"github.com/go-gost/core/auth"
	"github.com/go-gost/core/chain"
	"github.com/go-gost/core/connector"
	"github.com/go-gost/core/dialer"
	"github.com/go-gost/core/handler"
	"github.com/go-gost/core/hop"
	"github.com/go-gost/core/listener"
	"github.com/go-gost/core/service"
	socks5c "github.com/go-gost/x/connector/socks/v5"
	httpc "github.com/go-gost/x/connector/http"
	"github.com/go-gost/x/dialer/tcp"
	socks5h "github.com/go-gost/x/handler/socks/v5"
	tcpln "github.com/go-gost/x/listener/tcp"
)

// NodeConfig 节点配置
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

// HopConfig 跳跃配置
type HopConfig struct {
	Nodes []NodeConfig `yaml:"nodes"`
}

// ChainConfig 链配置
type ChainConfig struct {
	Name string      `yaml:"name"`
	Hops []HopConfig `yaml:"hops"`
}

// HandlerConfig 处理器配置
type HandlerConfig struct {
	Type  string `yaml:"type"`
	Chain string `yaml:"chain"`
}

// ListenerConfig 监听器配置
type ListenerConfig struct {
	Type string `yaml:"type"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name     string         `yaml:"name"`
	Addr     string         `yaml:"addr"`
	Handler  HandlerConfig  `yaml:"handler"`
	Listener ListenerConfig `yaml:"listener"`
}

// Config 总配置
type Config struct {
	Services []ServiceConfig `yaml:"services"`
	Chains   []ChainConfig   `yaml:"chains"`
}

// BuildChain 构建代理链
func BuildChain(chainCfg ChainConfig) (*chain.Chain, error) {
	var hops []*hop.Hop
	
	for _, hopCfg := range chainCfg.Hops {
		var nodes []*chain.Node
		
		for _, nodeCfg := range hopCfg.Nodes {
			// 创建节点
			node := chain.NewNode(nodeCfg.Addr, nodeCfg.Addr)
			
			// 设置连接器
			var conn connector.Connector
			switch nodeCfg.Connector.Type {
			case "socks5":
				conn = socks5c.NewConnector()
			case "http":
				conn = httpc.NewConnector()
			default:
				conn = socks5c.NewConnector() // 默认使用 socks5
			}
			
			// 设置拨号器
			var d dialer.Dialer
			switch nodeCfg.Dialer.Type {
			case "tcp":
				d = tcp.NewDialer()
			default:
				d = tcp.NewDialer() // 默认使用 tcp
			}
			
			// 设置认证
			if nodeCfg.Connector.Auth.Username != "" {
				node = node.WithAuth(auth.NewAuth(
					auth.UserAuthOption(
						url.UserPassword(
							nodeCfg.Connector.Auth.Username,
							nodeCfg.Connector.Auth.Password,
						),
					),
				))
			}
			
			node = node.WithConnector(conn).WithDialer(d)
			nodes = append(nodes, node)
		}
		
		if len(nodes) > 0 {
			h := hop.NewHop(hop.NodesOption(nodes...))
			hops = append(hops, h)
		}
	}
	
	if len(hops) == 0 {
		return nil, nil // 返回空链表示直连
	}
	
	return chain.NewChain(chain.HopsOption(hops...)), nil
}

// StartService 启动服务
func StartService(svcCfg ServiceConfig, chains map[string]*chain.Chain) error {
	// 创建监听器
	var ln listener.Listener
	switch svcCfg.Listener.Type {
	case "tcp":
		var err error
		ln, err = tcpln.NewListener(svcCfg.Addr)
		if err != nil {
			return fmt.Errorf("创建 TCP 监听器失败: %v", err)
		}
	default:
		var err error
		ln, err = tcpln.NewListener(svcCfg.Addr)
		if err != nil {
			return fmt.Errorf("创建默认监听器失败: %v", err)
		}
	}
	
	// 创建处理器
	var hdl handler.Handler
	switch svcCfg.Handler.Type {
	case "socks5":
		opts := []handler.Option{}
		
		// 如果指定了链，则使用链
		if svcCfg.Handler.Chain != "" {
			if c, exists := chains[svcCfg.Handler.Chain]; exists && c != nil {
				opts = append(opts, handler.RouterOption(c))
			}
		}
		
		hdl = socks5h.NewHandler(opts...)
	default:
		// 默认使用 socks5
		opts := []handler.Option{}
		if svcCfg.Handler.Chain != "" {
			if c, exists := chains[svcCfg.Handler.Chain]; exists && c != nil {
				opts = append(opts, handler.RouterOption(c))
			}
		}
		hdl = socks5h.NewHandler(opts...)
	}
	
	// 创建服务
	svc := service.NewService(svcCfg.Name, ln, hdl)
	
	// 启动服务
	go func() {
		if err := svc.Serve(); err != nil {
			log.Printf("服务 %s 运行出错: %v", svcCfg.Name, err)
		}
	}()
	
	return nil
}

// StartGostWithConfig 使用配置启动 GOST
func StartGostWithConfig(cfgFile string) error {
	// 这个函数现在移到 main.go 中实现
	// 这里保留一个简单的 SOCKS5 启动函数作为备用
	return StartSocks5Chain("127.0.0.1:1080", "", "", "")
}

// StartSocks5Chain 启动 socks5+chain 代理服务（go-gost v3 正确用法）
func StartSocks5Chain(listenAddr, remoteAddr, username, password string) error {
	// 创建监听器
	ln, err := tcpln.NewListener(listenAddr)
	if err != nil {
		return fmt.Errorf("创建监听器失败: %v", err)
	}
	
	// 创建处理器选项
	opts := []handler.Option{}
	
	// 如果有远程地址，创建链
	if remoteAddr != "" {
		// 创建节点
		node := chain.NewNode("proxy", remoteAddr)
		
		// 设置认证
		if username != "" {
			node = node.WithAuth(auth.NewAuth(
				auth.UserAuthOption(url.UserPassword(username, password)),
			))
		}
		
		// 设置连接器和拨号器
		node = node.WithConnector(socks5c.NewConnector()).WithDialer(tcp.NewDialer())
		
		// 创建跳跃和链
		h := hop.NewHop(hop.NodesOption(node))
		c := chain.NewChain(chain.HopsOption(h))
		
		opts = append(opts, handler.RouterOption(c))
	}
	
	// 创建 SOCKS5 处理器
	hdl := socks5h.NewHandler(opts...)
	
	// 创建服务
	svc := service.NewService("socks5", ln, hdl)
	
	// 启动服务
	go func() {
		if err := svc.Serve(); err != nil {
			log.Printf("SOCKS5 服务运行出错: %v", err)
		}
	}()
	
	return nil
}
