package tools

import (
	"ChinaTelecomMonitor/configs"
	"encoding/json"
	"os"
	"path/filepath"
)

// Token 存储用户的登录令牌信息
type Token struct {
	ChinaTelecomToken string `json:"chinaTelecomToken"` // 电信接口Token
	LoginLastTime     int64  `json:"loginLastTime"`     // 上次登录时间戳
}

// 获取用户Token文件路径
func getUserTokenPath(username string) string {
	return filepath。Join(configs。DataPath, "tokens", username+".json")
}

// GetUserToken 获取指定用户的Token
func GetUserToken(username string) *Token {
	path := getUserTokenPath(username)
	data, err := os。ReadFile(path)
	if err != nil {
		configs。Logger.Debugf("用户 %s 未找到Token文件: %v", username, err)
		return nil
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		configs.Logger.Errorf("解析用户 %s Token失败: %v", username, err)
		return nil
	}
	return &token
}

// SetUserToken 保存用户Token
func SetUserToken(username, tokenStr string, loginTime int64) {
	token := Token{
		ChinaTelecomToken: tokenStr,
		LoginLastTime:     loginTime,
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		configs.Logger.Errorf("序列化用户 %s Token失败: %v", username, err)
		return
	}

	// 确保目录存在
	_ = os.MkdirAll(filepath.Dir(getUserTokenPath(username)), 0755)

	if err := os.WriteFile(getUserTokenPath(username), data, 0644); err != nil {
		configs.Logger.Errorf("保存用户 %s Token失败: %v", username, err)
		return
	}
	configs.Logger.Infof("用户 %s Token保存成功", username)
}

// DeleteUserToken 删除用户Token
func DeleteUserToken(username string) {
	path := getUserTokenPath(username)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		configs.Logger.Errorf("删除用户 %s Token失败: %v", username, err)
	}
}
