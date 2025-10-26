# !/usr/bin/env python3
# -*- coding: utf-8 -*-
# Repo: https://github.com/vipxkw/ChinaTelecomMonitor
# ConfigFile: telecom_config.json
# Modify: 2025-10-26

"""
任务名称
name: 电信套餐用量监控
定时规则
cron: 0 20 * * *
"""

import os
import sys
import json
import datetime
import calendar
import re

# 兼容青龙
try:
    from telecom_class import Telecom
except:
    print("正在尝试自动安装依赖...")
    os.system("pip3 install pycryptodome requests &> /dev/null")
    from telecom_class import Telecom


CONFIG_DATA = {}
NOTIFYS = []
CONFIG_PATH = sys.argv[1] if len(sys.argv) > 1 else "telecom_config.json"


# 发送通知消息
def send_notify(title, body):
    try:
        # 导入通知模块
        import notify

        # 如未配置 push_config 则使用青龙环境通知设置
        if CONFIG_DATA.get("push_config"):
            notify.push_config.update(CONFIG_DATA["push_config"])
            notify.push_config["CONSOLE"] = notify.push_config.get("CONSOLE", True)
        notify.send(title, body)
    except Exception as e:
        if e:
            print("发送通知消息失败！")


# 添加消息
def add_notify(text):
    global NOTIFYS
    NOTIFYS.append(text)
    print("📢", text)
    return text


def create_progress_bar(percentage):
    """创建进度条"""
    bar_length = 10
    filled_length = int(bar_length * percentage / 100)
    bar = '█' * filled_length + '░' * (bar_length - filled_length)
    return f"[{bar}] {percentage:.1f}%"


def format_flow_size(kb, unit='MB', decimal=2):
    """格式化流量大小"""
    if unit == 'GB':
        return round(kb / (1024 * 1024), decimal)
    elif unit == 'MB':
        return round(kb / 1024, decimal)
    else:
        return kb


def mask_phone_number(phone):
    """手机号脱敏"""
    if len(phone) == 11:
        return phone[:3] + "****" + phone[7:]
    return phone


def parse_flow_package_detail(flux_package_str):
    """解析流量包明细并转换为新格式"""
    if not flux_package_str:
        return ""
    
    result = "\n\n📋 流量包明细"
    lines = flux_package_str.strip().split('\n')
    
    current_category = ""
    for line in lines:
        line = line.strip()
        if not line:
            continue
            
        # 识别分类标题 (如 🇨🇳国内通用流量)
        if line.startswith(('🇨🇳', '📺', '🌎')):
            current_category = line
            continue
            
        # 解析流量包条目 (如 🔹[电信无忧卡201905-免费资源]已用52.92MB/共52.92MB)
        if line.startswith('🔹'):
            # 提取流量包名称和数据
            match = re.search(r'\[([^\]]+)\](.+)', line)
            if match:
                package_name = match.group(1)
                data_part = match.group(2)
                
                # 解析已用/共计数据
                if '已用' in data_part and '/共' in data_part:
                    # 提取已用和总计数据
                    used_match = re.search(r'已用([\d.]+)([KMGT]?B)', data_part)
                    total_match = re.search(r'/共([\d.]+)([KMGT]?B)', data_part)
                    
                    if used_match and total_match:
                        used_value = float(used_match.group(1))
                        used_unit = used_match.group(2)
                        total_value = float(total_match.group(1))
                        total_unit = total_match.group(2)
                        
                        # 转换为KB计算百分比
                        used_kb = convert_to_kb(used_value, used_unit)
                        total_kb = convert_to_kb(total_value, total_unit)
                        
                        percentage = (used_kb / total_kb * 100) if total_kb > 0 else 0
                        balance_value = total_value - used_value
                        
                        result += f"\n└─ 📦 {package_name}"
                        result += f"\n├─ 使用情况：{create_progress_bar(percentage)}"
                        result += f"\n└─ 总量：{total_value} {total_unit} | 已用：{used_value} {used_unit} | 余量：{balance_value:.2f} {total_unit}\n\n"
                
                # 处理无限流量的情况
                elif '无限' in data_part:
                    result += f"\n└─ 📦 {package_name}"
                    result += f"\n├─ 使用情况：{data_part}"
                    result += f"\n└─ 类型：无限流量"
    
    return result


def convert_to_kb(value, unit):
    """将流量值转换为KB"""
    unit_dict = {"B": 1/1024, "KB": 1, "MB": 1024, "GB": 1024*1024, "TB": 1024*1024*1024}
    return value * unit_dict.get(unit, 1)


def generate_usage_status(data):
    """生成使用状态提醒"""
    message = "\n📈 使用状态"
    
    # 计算百分比
    total_percentage = (data['flowUse'] / data['flowTotal'] * 100) if data['flowTotal'] > 0 else 0
    general_percentage = (data['commonUse'] / data['commonTotal'] * 100) if data['commonTotal'] > 0 else 0
    
    # 总流量状态提醒
    if total_percentage >= 90:
        message += "\n⚠️ 总流量即将用完，请注意控制使用"
    elif total_percentage >= 80:
        message += "\n🔶 总流量使用较多，建议适当控制"
    elif total_percentage >= 50:
        message += "\n🟡 总流量使用正常"
    else:
        message += "\n🟢 总流量充足"
    
    # 通用流量状态提醒
    if general_percentage >= 90:
        message += "\n⚠️ 通用流量即将用完"
    elif general_percentage >= 80:
        message += "\n🔶 通用流量使用较多"
    
    # 余额提醒
    balance_num = data.get('balance', 0) / 100
    if balance_num < 10:
        message += "\n💸 余额不足，建议及时充值"
    elif balance_num < 20:
        message += "\n💰 余额较低，请关注"
    
    return message


def format_notify_message(summary, flux_package_str=""):
    """格式化通知消息"""
    # 计算百分比
    voice_percentage = (summary['voiceUsage'] / summary['voiceTotal'] * 100) if summary['voiceTotal'] > 0 else 0
    total_percentage = (summary['flowUse'] / summary['flowTotal'] * 100) if summary['flowTotal'] > 0 else 0
    common_percentage = (summary['commonUse'] / summary['commonTotal'] * 100) if summary['commonTotal'] > 0 else 0
    special_percentage = (summary['specialUse'] / summary['specialTotal'] * 100) if summary['specialTotal'] > 0 else 0
    
    # 格式化数据
    balance = round(summary['balance'] / 100, 2)
    
    # 语音数据
    voice_total = summary['voiceTotal']
    voice_used = summary['voiceUsage']
    voice_balance = summary['voiceBalance']
    
    # 总流量数据 (MB)
    total_flow_mb = format_flow_size(summary['flowTotal'], 'MB')
    total_used_mb = format_flow_size(summary['flowUse'], 'MB')
    total_balance_mb = format_flow_size(summary['flowTotal'] - summary['flowUse'], 'MB')
    
    # 通用流量数据 (MB)
    common_total_mb = format_flow_size(summary['commonTotal'], 'MB')
    common_used_mb = format_flow_size(summary['commonUse'], 'MB')
    common_balance_mb = format_flow_size(summary['commonTotal'] - summary['commonUse'], 'MB')
    
    # 专用流量数据
    special_total_kb = summary['specialTotal']
    special_used_kb = summary['specialUse']
    special_balance_kb = summary['specialTotal'] - summary['specialUse']
    
    # 构建消息
    message = f"""\n\n📱 电信使用量查询结果

👤 用户信息
├─ 手机号码：{mask_phone_number(summary['phonenum'])}
└─ 查询时间：{summary['createTime']}

💰 账户余额：{balance}元

📞 语音通话
├─ 使用情况：{create_progress_bar(voice_percentage)}
└─ 总量：{voice_total}分钟 | 已用：{voice_used}分钟 | 余量：{voice_balance}分钟

📊 总流量统计
├─ 使用情况：{create_progress_bar(total_percentage)}
└─ 总量：{total_flow_mb} MB | 已用：{total_used_mb} MB | 余量：{total_balance_mb} MB

🌐 通用流量
├─ 使用情况：{create_progress_bar(common_percentage)}
└─ 总量：{common_total_mb} MB | 已用：{common_used_mb} MB | 余量：{common_balance_mb} MB

🎯 专用流量
├─ 使用情况：{create_progress_bar(special_percentage)}
└─ 总量：{special_total_kb} KB | 已用：{special_used_kb} KB | 余量：{special_balance_kb} KB"""

    # 流量明细
    if summary.get('flowItems'):
        message += "\n\n📋 流量明细"
        for item in summary['flowItems']:
            item_percentage = (item['use'] / item['total'] * 100) if item['total'] > 0 else 0
            item_total_mb = format_flow_size(item['total'], 'MB')
            item_used_mb = format_flow_size(item['use'], 'MB')
            item_balance_mb = format_flow_size(item['balance'], 'MB')
            
            message += f"\n└─ 📦 {item['name']}"
            message += f"\n├─ 使用情况：{create_progress_bar(item_percentage)}"
            message += f"\n└─ 总量：{item_total_mb} MB | 已用：{item_used_mb} MB | 余量：{item_balance_mb} MB"

    # 流量包明细 - 使用新格式
    if flux_package_str:
        message += parse_flow_package_detail(flux_package_str)

    # 使用状态
    message += generate_usage_status(summary)
    
    message += f"\n\n📅 更新时间：{summary['createTime']}"
    
    return message


def parse_users_from_env():
    """从环境变量解析用户信息"""
    users = []
    telecom_user_env = os.environ.get("TELECOM_USER", "")
    
    if not telecom_user_env:
        return users
    
    # 分割多用户
    user_configs = telecom_user_env.split("@")
    
    for user_config in user_configs:
        if not user_config.strip():
            continue
            
        parts = user_config.split(",")
        
        if len(parts) >= 2:
            phonenum = parts[0].strip()
            password = parts[1].strip()
            flux_package = parts[2].strip().lower() == "true" if len(parts) >= 3 else True
            
            if phonenum and password:
                users.append({
                    "phonenum": phonenum,
                    "password": password,
                    "flux_package": flux_package
                })
    
    return users


def process_user(user_info):
    """处理单个用户"""
    phonenum = user_info["phonenum"]
    password = user_info["password"]
    flux_package_enabled = user_info["flux_package"]
    
    print(f"\n===============处理用户 {mask_phone_number(phonenum)}===============")
    
    telecom = Telecom()
    
    def auto_login():
        if not phonenum.isdigit():
            print(f"自动登录：手机号设置错误，跳过用户 {mask_phone_number(phonenum)}")
            return False
        else:
            print(f"自动登录：{mask_phone_number(phonenum)}")
        
        # 记录登录失败次数，避免风控
        user_key = f"loginFailTime_{phonenum}"
        login_fail_time = CONFIG_DATA.get(user_key, 0)
        
        if login_fail_time < 5:
            data = telecom.do_login(phonenum, password)
            if data.get("responseData").get("resultCode") == "0000":
                print(f"自动登录：成功")
                login_info = data["responseData"]["data"]["loginSuccessResult"]
                login_info["phonenum"] = phonenum
                login_info["createTime"] = datetime.datetime.now().strftime(
                    "%Y-%m-%d %H:%M:%S"
                )
                CONFIG_DATA[f"login_info_{phonenum}"] = login_info
                CONFIG_DATA[user_key] = 0
                telecom.set_login_info(login_info)
                return True
            else:
                login_fail_time = int(
                    data.get("responseData", {})
                    .get("data", {})
                    .get("loginFailResult", {})
                    .get("loginFailTime", login_fail_time + 1)
                )
                CONFIG_DATA[user_key] = login_fail_time
                print(f"自动登录：已连续失败{login_fail_time}次，跳过用户 {mask_phone_number(phonenum)}")
                return False
        else:
            print(f"自动登录：已连续失败{login_fail_time}次，为避免风控跳过用户 {mask_phone_number(phonenum)}")
            return False

    # 读取缓存Token
    login_info = CONFIG_DATA.get(f"login_info_{phonenum}", {})
    if login_info and login_info.get("phonenum"):
        print(f"尝试使用缓存登录：{mask_phone_number(login_info['phonenum'])}")
        telecom.set_login_info(login_info)
    else:
        if not auto_login():
            return None

    # 获取主要信息
    important_data = telecom.qry_important_data()
    if important_data.get("responseData"):
        print(f"获取主要信息：成功")
    elif important_data["headerInfos"]["code"] == "X201":
        print(f"获取主要信息：失败 {important_data['headerInfos']['reason']}")
        if not auto_login():
            return None
        important_data = telecom.qry_important_data()

    # 简化主要信息
    try:
        summary = telecom.to_summary(important_data["responseData"]["data"])
    except Exception as e:
        print(f"简化主要信息出错：{e}")
        return None
        
    if summary:
        print(f"简化主要信息：成功")
        CONFIG_DATA[f"summary_{phonenum}"] = summary

    # 获取流量包明细
    flux_package_str = ""
    if flux_package_enabled:
        user_flux_package = telecom.user_flux_package()
        if user_flux_package:
            print("获取流量包明细：成功")
            packages = user_flux_package["responseData"]["data"]["productOFFRatable"][
                "ratableResourcePackages"
            ]
            for package in packages:
                package_icon = (
                    "🇨🇳"
                    if "国内" in package["title"]
                    else "📺" if "专用" in package["title"] else "🌎"
                )
                flux_package_str += f"\n{package_icon}{package['title']}\n"
                for product in package["productInfos"]:
                    if product["infiniteTitle"]:
                        # 无限流量
                        flux_package_str += f"""🔹[{product['title']}]{product['infiniteTitle']}{product['infiniteValue']}{product['infiniteUnit']}/无限\n"""
                    else:
                        flux_package_str += f"""🔹[{product['title']}]{product['leftTitle']}{product['leftHighlight']}{product['rightCommon']}\n"""

    # 格式化通知消息
    notify_str = format_notify_message(summary, flux_package_str)
    
    return notify_str


def main():
    global CONFIG_DATA
    start_time = datetime.datetime.now()
    print(f"===============程序开始===============")
    print(f"⏰ 执行时间: {start_time.strftime('%Y-%m-%d %H:%M:%S')}")
    print()
    
    # 读取配置
    if os.path.exists(CONFIG_PATH):
        print(f"⚙️ 正从 {CONFIG_PATH} 文件中读取配置")
        with open(CONFIG_PATH, "r", encoding="utf-8") as file:
            CONFIG_DATA = json.load(file)
    if not CONFIG_DATA.get("user"):
        CONFIG_DATA["user"] = {}

    # 解析用户信息
    users = parse_users_from_env()
    
    if not users:
        # 兼容旧版本单用户配置
        if TELECOM_USER := os.environ.get("TELECOM_USER"):
            if "," not in TELECOM_USER and "@" not in TELECOM_USER:
                phonenum, password = (
                    TELECOM_USER[:11],
                    TELECOM_USER[11:],
                )
                flux_package = os.environ.get("TELECOM_FLUX_PACKAGE", "true").lower() != "false"
                users = [{
                    "phonenum": phonenum,
                    "password": password,
                    "flux_package": flux_package
                }]
        elif TELECOM_USER := CONFIG_DATA.get("user", {}):
            phonenum = TELECOM_USER.get("phonenum", "")
            password = TELECOM_USER.get("password", "")
            flux_package = os.environ.get("TELECOM_FLUX_PACKAGE", "true").lower() != "false"
            if phonenum and password:
                users = [{
                    "phonenum": phonenum,
                    "password": password,
                    "flux_package": flux_package
                }]
    
    if not users:
        exit("未设置账号密码，退出")

    print(f"共检测到 {len(users)} 个用户")

    # 处理所有用户
    for i, user_info in enumerate(users, 1):
        print(f"\n处理第 {i}/{len(users)} 个用户")
        result = process_user(user_info)
        if result:
            add_notify(result)

    # 通知
    if NOTIFYS:
        notify_body = "\n\n" + "="*50 + "\n\n".join(NOTIFYS)
        print(f"===============推送通知===============")
        send_notify("📢【电信套餐用量监控】", notify_body)
        print()

    update_config()


def update_config():
    # 更新配置
    with open(CONFIG_PATH, "w", encoding="utf-8") as file:
        json.dump(CONFIG_DATA, file, ensure_ascii=False, indent=2)


if __name__ == "__main__":
    main()
