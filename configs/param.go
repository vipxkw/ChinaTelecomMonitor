// filePath: ChinaTelecomMonitor/configs/param.go
package configs

import (
	"China_Telecom_Monitor/models"
	"os"
	"go.uber.org/zap"
)

var Prot string
var Username string
var Password string
var LoginIntervalTime int
var TimeOut int64
var IntervalsTime int

var DataPath string
var LogLevel string
var LogEncoding string
var Dev bool
var PrintVersion bool
var ClientVersion string

var Summary models.Summary
var Logger *zap.SugaredLogger

// 从环境变量加载配置
func LoadFromEnv() {
	if env := os.Getenv("TELECOM_USERNAME"); env != "" {
		Username = env
	}
	if env := os.Getenv("TELECOM_PASSWORD"); env != "" {
		Password = env
	}
	if env := os.Getenv("TELECOM_PROT"); env != "" {
		Prot = env
	}
	if env := os.Getenv("TELECOM_CLIENT_VERSION"); env != "" {
		ClientVersion = env
	}
}
