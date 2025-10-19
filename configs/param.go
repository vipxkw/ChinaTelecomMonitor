package configs

import (
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

var (
	Prot              string        // 服务端口
	LoginIntervalTime int           // 登录间隔时间(秒)
	TimeOut           int64         // 超时时间(秒)
	IntervalsTime     int           // 定时任务间隔(秒)
	DataPath          string        // 数据存储路径
	LogLevel          string        // 日志级别
	LogEncoding       string        // 日志格式(json/console)
	Dev               bool          // 开发模式
	PrintVersion      bool          // 显示版本信息
	ClientVersion     string        // 客户端版本
	Logger            *zap。SugaredLogger // 日志实例
	ApiKey            string        // API访问密钥(新增)
)

// 初始化默认配置
func InitDefaultConfig() {
	// 设置默认数据路径
	if DataPath == "" {
		DataPath = "./data"
	}
	// 设置默认端口
	if Prot == "" {
		Prot = "8080"
	}
	// 设置默认登录间隔(12小时)
	if LoginIntervalTime == 0 {
		LoginIntervalTime = 43200
	}
	// 设置默认超时时间
	if TimeOut == 0 {
		TimeOut = 30
	}
	// 设置默认定时任务间隔
	if IntervalsTime == 0 {
		IntervalsTime = 60
	}
	// 设置默认日志级别
	if LogLevel == "" {
		LogLevel = "info"
	}
	// 设置默认日志格式
	if LogEncoding == "" {
		LogEncoding = "console"
	}
	// 设置默认客户端版本
	if ClientVersion == "" {
		ClientVersion = "12.2.0"
	}

	// 从环境变量读取API密钥(新增)
	if ApiKey == "" {
		ApiKey = os.Getenv("API_KEY")
	}

	// 创建必要目录
	_ = os.MkdirAll(filepath.Join(DataPath, "log"), 0755)
	_ = os.MkdirAll(filepath.Join(DataPath, "tokens"), 0755)

	// 验证API密钥是否设置(新增)
	if ApiKey == "" {
		Logger.Warn("警告：API_KEY未设置，接口安全验证将失效")
	}
}
