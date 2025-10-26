package main

import (
	"China_Telecom_Monitor/configs"
	"China_Telecom_Monitor/models"
	"China_Telecom_Monitor/tools"
	"flag"
	"fmt"
	"github.com/golang-module/carbon/v2"
	"github.com/kataras/iris/v12"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"strings"
	"sync"
)

var Version = "v2.0.2"
var GoVersion = "not set"
var GitCommit = "not set"
var BuildTime = "not set"

// 多用户数据缓存
var userDataCache = make(map[string]*userCacheData)
var userCacheMutex sync.RWMutex

type userCacheData struct {
	Summary              models.Summary
	QryImportantData     *models.Result[models.ImportantData]
	UserFluxPackageData  *models.Result[models.UserFluxPackageData]
	LastFlowTime         carbon.Carbon
	LastQryTime          carbon.Carbon
	LastUserFluxTime     carbon.Carbon
}

func main() {
	initFlag()
	if configs.PrintVersion {
		version()
		return
	}

	initLogger()

	if checkFlag() {
		return
	}

	initCron()
	initIris()
}

// 初始化配置
func initFlag() {
	flag.StringVar(&configs.Prot, "prot", "8080", "--prot 8080")
	flag.StringVar(&configs.ApiKey, "apiKey", "", "--apiKey xxxxx # API访问密钥")
	flag.IntVar(&configs.LoginIntervalTime, "loginIntervalTime", 43200, "--loginIntervalTime 43200 #电信登录间隔时间（防止被封号），秒")
	flag.Int64Var(&configs.TimeOut, "timeOut", 30, "--timeOut 30 #访问电信接口请求超时时间，秒")
	flag.IntVar(&configs.IntervalsTime, "intervalsTime", 180, "--intervalsTime 180 #接口防止重刷时间")

	flag.StringVar(&configs.LogLevel, "logLevel", "info", "--logLevel info # 日志等级")
	flag.StringVar(&configs.LogEncoding, "logEncoding", "console", "--logEncoding console # 日志输出格式 console 或 json")

	flag.StringVar(&configs.DataPath, "dataPath", "./data", "--dataPath ./data # 数据日志文件保存路径")

	flag.StringVar(&configs.ClientVersion, "clientVersion", "12.2.0", "--clientVersion '12.2.0' # 登录电信客户端版本(电信会限制过低的版本无法进行登录)")

	flag.BoolVar(&configs.Dev, "dev", false, "--dev false # 开发模式,开启后将支持以下接口： /refresh 手动更新流量 和 /show/qryImportantData /show/userFluxPackage 这里两个电信接口")
	flag.BoolVar(&configs.PrintVersion, "version", false, "--version 打印程序构建版本")

	flag.Parse()
}

func version() {
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Go Version: %s\n", GoVersion)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Build Time: %s\n", BuildTime)
}

func checkFlag() bool {
	// 验证必填参数
	if configs.ApiKey == "" {
		configs.Logger.Error("--apiKey 参数不能为空")
		return true
	}
	return false
}

// 初始化 zap日志框架
func initLogger() {
	level := getLevel()
	encoding := configs.LogEncoding

	stdout := &lumberjack.Logger{
		Filename:   configs.DataPath + "/log/stdout.log",
		MaxSize:    10, // 每个日志文件的最大大小，单位为MB
		MaxBackups: 10, // 保留的旧日志文件的最大数量
		MaxAge:     60, // 保留的旧日志文件的最大天数
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // 短路径编码器
	}
	atom := zap.NewAtomicLevelAt(level)
	config := zap.Config{
		Level:         atom,          // 日志级别
		Development:   true,          // 开发模式，堆栈跟踪
		Encoding:      encoding,      // 输出格式 console 或 json
		EncoderConfig: encoderConfig, // 编码器配置
		OutputPaths:      []string{"stdout", stdout.Filename}, // 输出到指定文件
		ErrorOutputPaths: []string{"stderr", stdout.Filename},
	}
	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("logger 初始化失败: %v", err))
	}
	configs.Logger = logger.Sugar()
	configs.Logger.Info("logger 初始化成功")
}

// 获取 日志等级
func getLevel() zapcore.Level {
	levelStr := strings.TrimSpace(strings.ToLower(configs.LogLevel))
	var level zapcore.Level
	switch levelStr {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "dpanic":
		level = zap.DPanicLevel
	case "panic":
		level = zap.PanicLevel
	case "fatal":
		level = zap.FatalLevel
	default:
		level = zap.InfoLevel
	}
	return level
}

// 初始化 定时任务
func initCron() {
	cronApp := cron.New(cron.WithSeconds()) //精确到秒
	spec := "0 0 */2 * * ?"                 //cron表达式，每2小时一次，保障cookie有效
	_, err := cronApp.AddFunc(spec, func() {
		// 定时任务只做必要的维护工作，不进行具体用户数据获取
		configs.Logger.Info("定时任务执行: 维护用户缓存")
	})
	if err != nil {
		configs.Logger.Error(`0x000007 `, err)
		return
	}
	cronApp.Start()
}

// 获取流量信息
func cronSummary(username, password string) {
	t := carbon.Now()
	qryImportantData := tools.GetQryImportantData(username, password)
	userCache := getUserCacheData(username)
	userCache.Summary = tools.ToSummary(qryImportantData, username, t)
	userCache.LastFlowTime = carbon.Now()
}

// 初始化访问接口
func initIris() {
	irisApp := iris.New()
	irisApp.Use(middleware)
	irisApp.Use(apiKeyAuth)

	irisApp.Handle(iris.MethodGet, "/show/flow", flow)
	irisApp.Handle(iris.MethodGet, "/show/detail", packageDetail)
	irisApp.Handle(iris.MethodGet, "/show/flowPackage", flowPackage)

	if configs.Dev {
		irisApp.Handle(iris.MethodGet, "/refresh", refresh)
		irisApp.Handle(iris.MethodGet, "/show/qryImportantData", qryImportantData)
		irisApp.Handle(iris.MethodGet, "/show/userFluxPackage", userFluxPackage)
	}

	err := irisApp.Run(iris.Addr(":" + configs.Prot))
	if err != nil {
		configs.Logger.Error("InitIris error", err)
	}
}

func middleware(ctx iris.Context) {
	configs.Logger.Infof("iris access Path: %s | IP: %s", ctx.Path(), ctx.RemoteAddr())
	ctx.Next()
}

// API密钥验证中间件
func apiKeyAuth(ctx iris.Context) {
	// 从请求参数获取用户名、密码和key
	username := ctx.URLParam("username")
	password := ctx.URLParam("password")
	requestKey := ctx.URLParam("key")

	// 验证参数是否齐全
	if username == "" || password == "" || requestKey == "" {
		ctx.JSON(iris.Map{
			"code": 401,
			"msg":  "缺少必要参数username、password或key",
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

// 获取或创建用户缓存数据
func getUserCacheData(username string) *userCacheData {
	userCacheMutex.Lock()
	defer userCacheMutex.Unlock()
	
	if data, exists := userDataCache[username]; exists {
		return data
	}
	
	data := &userCacheData{}
	userDataCache[username] = data
	return data
}

// 获取流量信息接口
func flow(ctx iris.Context) {
	username := ctx.URLParam("username")
	password := ctx.URLParam("password")

	userCache := getUserCacheData(username)
	if carbon.Now().Lt(userCache.LastFlowTime.AddSeconds(configs.IntervalsTime)) {
		ctx.JSON(iris.Map{"code": 200, "data": desensitization(userCache.Summary)})
		return
	}

	cronSummary(username, password)
	summary := desensitization(userCache.Summary)
	ctx.JSON(iris.Map{"code": 200, "data": summary})
}

func packageDetail(ctx iris.Context) {
	ctx.JSON(&models.DetailRequest{
		Result:          410,
		ParaFieldResult: "接口已失效",
	})
}

func flowPackage(ctx iris.Context) {
	ctx.JSON(&models.FlowPackage{
		Result: 410,
		Msg:    "接口已失效",
	})
}

func qryImportantData(ctx iris.Context) {
	username := ctx.URLParam("username")
	password := ctx.URLParam("password")

	userCache := getUserCacheData(username)
	if carbon.Now().Lt(userCache.LastQryTime.AddSeconds(configs.IntervalsTime)) {
		ctx.JSON(&userCache.QryImportantData)
		return
	}

	userCache.QryImportantData = tools.GetQryImportantData(username, password)
	userCache.LastQryTime = carbon.Now()
	ctx.JSON(&userCache.QryImportantData)
}

func userFluxPackage(ctx iris.Context) {
	username := ctx.URLParam("username")
	password := ctx.URLParam("password")

	userCache := getUserCacheData(username)
	if carbon.Now().Lt(userCache.LastUserFluxTime.AddSeconds(configs.IntervalsTime)) {
		ctx.JSON(&userCache.UserFluxPackageData)
		return
	}

	userCache.UserFluxPackageData = tools.GetUserFluxPackage(username, password)
	userCache.LastUserFluxTime = carbon.Now()
	ctx.JSON(&userCache.UserFluxPackageData)
}

func desensitization(summary models.Summary) models.Summary {
	if configs.Dev {
		if len(summary.Username) == 11 {
			summary.Username = summary.Username[0:3] + "****" + summary.Username[7:11]
		}
	} else {
		if len(summary.Username) > 0 {
			summary.Username = ""
		}
	}
	return summary
}

// 手动触发获取流量信息
func refresh(ctx iris.Context) {
	username := ctx.URLParam("username")
	password := ctx.URLParam("password")

	go cronSummary(username, password)
	ctx.JSON(iris.Map{"code": 200, "msg": "已触发指定用户流量更新"})
}
