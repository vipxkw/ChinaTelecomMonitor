package main

import (
	"ChinaTelecomMonitor/configs"
	"ChinaTelecomMonitor/tools"
	"flag"
	"fmt"
	"github.com/kataras/iris/v12"
	"go.uber.org/zap"
	"time"
)

var (
	goVersion string // 编译时注入
	gitCommit string // 编译时注入
	buildTime string // 编译时注入
)

func main() {
	initFlag()                  // 初始化命令行参数
	if configs.PrintVersion {   // 显示版本信息
		version()
		return
	}

	configs.InitDefaultConfig() // 初始化配置
	initLogger()                // 初始化日志
	go startCron()              // 启动定时任务
	initIris()                  // 初始化Web服务
}

// 初始化命令行参数
func initFlag() {
	flag.StringVar(&configs.Prot, "prot", "", "服务端口")
	flag.StringVar(&configs.DataPath, "data", "", "数据存储路径")
	flag.StringVar(&configs.LogLevel, "log-level", "", "日志级别")
	flag.StringVar(&configs.LogEncoding, "log-encoding", "", "日志格式(json/console)")
	flag.BoolVar(&configs.Dev, "dev", false, "开发模式")
	flag.BoolVar(&configs.PrintVersion, "version", false, "显示版本信息")
	flag.StringVar(&configs.ApiKey, "api-key", "", "API访问密钥(新增)") // 新增：API密钥参数
	flag.Parse()
}

// 显示版本信息
func version() {
	fmt.Printf("China Telecom Monitor\n")
	fmt.Printf("Go Version: %s\n", goVersion)
	fmt.Printf("Git Commit: %s\n", gitCommit)
	fmt.Printf("Build Time: %s\n", buildTime)
}

// 初始化日志
func initLogger() {
	cfg := zap.NewProductionConfig()
	if configs.LogLevel != "" {
		lvl, err := zap.ParseAtomicLevel(configs.LogLevel)
		if err == nil {
			cfg.Level = lvl
		}
	}
	if configs.LogEncoding != "" {
		cfg.Encoding = configs.LogEncoding
	}
	cfg.OutputPaths = []string{
		"stdout",
		fmt.Sprintf("%s/log/app.log", configs.DataPath),
	}
	logger, _ := cfg.Build()
	configs.Logger = logger.Sugar()
}

// 启动定时任务
func startCron() {
	ticker := time.NewTicker(time.Duration(configs.IntervalsTime) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		configs.Logger.Debug("定时任务执行中...")
	}
}

// 初始化Iris Web服务
func initIris() {
	app := iris.New()
	app.Use(requestLogger)   // 请求日志中间件
	app.Use(apiKeyAuth)      // 新增：API密钥验证中间件

	// 多用户接口 - 通过请求参数传递用户名密码和key
	app.Get("/show/flow", flowHandler)
	if configs.Dev {
		app.Get("/show/qryImportantData", qryImportantDataHandler)
		app.Get("/show/userFluxPackage", userFluxPackageHandler)
	}

	configs.Logger.Infof("服务启动成功，监听端口: %s", configs.Prot)
	if err := app.Run(iris.Addr(":" + configs.Prot)); err != nil {
		configs.Logger.Fatalf("服务启动失败: %v", err)
	}
}

// 请求日志中间件
func requestLogger(ctx iris.Context) {
	start := time.Now()
	ctx.Next()
	configs.Logger.Infof(
		"请求: %s %s | 状态: %d | 耗时: %v",
		ctx.Method(),
		ctx.Path(),
		ctx.GetStatusCode(),
		time.Since(start),
	)
}

// 新增：API密钥验证中间件
func apiKeyAuth(ctx iris.Context) {
	// 如果未设置API密钥，跳过验证
	if configs.ApiKey == "" {
		ctx.Next()
		return
	}

	// 从请求参数获取key
	requestKey := ctx.URLParam("key")
	if requestKey == "" {
		ctx.JSON(iris.Map{
			"code": 401,
			"msg":  "缺少必要参数key",
		})
		ctx.StopExecution()
		return
	}

	// 验证密钥是否匹配
	if requestKey != configs.ApiKey {
		configs.Logger.Warnf("无效的API密钥尝试访问: %s %s", ctx.Method(), ctx.Path())
		ctx.JSON(iris.Map{
			"code": 403,
			"msg":  "密钥验证失败",
		})
		ctx.StopExecution()
		return
	}

	ctx.Next()
}

// 流量查询接口
func flowHandler(ctx iris.Context) {
	username := ctx.URLParam("username")
	password := ctx.URLParam("password")

	// 验证用户参数
	if username == "" || password == "" {
		ctx.JSON(iris.Map{
			"code": 400,
			"msg":  "username和password参数不能为空",
		})
		return
	}

	// 执行登录（含频率控制）
	if !tools.ChinaTelecomLogin(username, password) {
		ctx。JSON(iris。Map{
			"code": 403,
			"msg":  "登录频率限制，请稍后再试",
		})
		return
	}

	// 获取数据
	data := tools。GetQryImportantData(username, password)
	if data == nil {
		ctx.JSON(iris.Map{
			"code": 500,
			"msg":  "获取数据失败",
		})
		return
	}

	ctx。JSON(iris。Map{
		"code": 200,
		"data": data,
	})
}

// 原始数据查询接口
func qryImportantDataHandler(ctx iris。Context) {
	username := ctx.URLParam("username")
	password := ctx.URLParam("password")

	if username == "" || password == "" {
		ctx。JSON(iris。Map{
			"code": 400,
			"msg":  "username和password参数不能为空",
		})
		return
	}

	result := tools.GetQryImportantData(username, password)
	ctx.JSON(result)
}

// 流量包查询接口
func userFluxPackageHandler(ctx iris.Context) {
	username := ctx.URLParam("username")
	password := ctx.URLParam("password")

	if username == "" || password == "" {
		ctx.JSON(iris.Map{
			"code": 400，
			"msg":  "username和password参数不能为空"，
		})
		return
	}

	result := tools.GetUserFluxPackage(username, password)
	ctx.JSON(result)
}
