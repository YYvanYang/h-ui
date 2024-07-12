package service

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"h-ui/dao"
	"h-ui/model/bo"
	"h-ui/model/constant"
	"h-ui/proxy"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func Hysteria2Auth(conPass string) (string, error) {
	if !Hysteria2IsRunning() {
		return "", errors.New("hysteria2 is not running")
	}

	now := time.Now().UnixMilli()
	account, err := dao.GetAccount("con_pass = ? and deleted = 0 and (quota < 0 or quota > download + upload) and ? < expire_time and ? > kick_util_time", conPass, now, now)
	if err != nil {
		return "", err
	}

	// 限制设备数
	onlineUsers, err := Hysteria2Online()
	if err != nil {
		return "", err
	}
	device, exist := onlineUsers[*account.Username]
	if exist && *account.DeviceNo < device {
		return "", errors.New("device limited")
	}

	return *account.Username, nil
}

func Hysteria2Online() (map[string]int64, error) {
	if !Hysteria2IsRunning() {
		return map[string]int64{}, nil
	}
	apiPort, err := GetHysteria2ApiPort()
	if err != nil {
		return nil, errors.New("get hysteria2 apiPort err")
	}
	jwtSecretConfig, err := dao.GetConfig("key = ?", constant.JwtSecret)
	if err != nil {
		return nil, err
	}
	onlineUsers, err := proxy.NewHysteria2Api(apiPort).OnlineUsers(*jwtSecretConfig.Value)
	if err != nil {
		return nil, err
	}
	return onlineUsers, nil
}

func Hysteria2Kick(ids []int64, kickUtilTime int64) error {
	if !Hysteria2IsRunning() {
		return errors.New("hysteria2 is not running")
	}
	if err := dao.UpdateAccount(ids, map[string]interface{}{"kick_util_time": kickUtilTime}); err != nil {
		return err
	}

	accounts, err := dao.ListAccount("id in ?", ids)
	if err != nil {
		return err
	}
	var keys []string
	for _, item := range accounts {
		keys = append(keys, *item.Username)
	}
	apiPort, err := GetHysteria2ApiPort()
	if err != nil {
		return errors.New("get hysteria2 apiPort err")
	}
	jwtSecretConfig, err := dao.GetConfig("key = ?", constant.JwtSecret)
	if err != nil {
		return err
	}
	if err = proxy.NewHysteria2Api(apiPort).KickUsers(keys, *jwtSecretConfig.Value); err != nil {
		return err
	}
	return nil
}

func Hysteria2SubscribeUrl(accountId int64, protocol string, host string) (string, error) {
	account, err := dao.GetAccount("id = ?", accountId)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s//%s/hui/%s", protocol, host, *account.ConPass), nil
}

func Hysteria2Subscribe(conPass string, clientType string, host string) (string, string, error) {
	hysteria2Config, err := GetHysteria2Config()
	if err != nil {
		return "", "", err
	}
	if hysteria2Config.Listen == nil || *hysteria2Config.Listen == "" {
		return "", "", errors.New("hysteria2 config is empty")
	}

	account, err := dao.GetAccount("con_pass = ?", conPass)
	if err != nil {
		return "", "", err
	}

	userInfo := ""
	configYaml := ""
	if clientType == constant.Shadowrocket || clientType == constant.ClashforWindows {
		userInfo = fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d",
			*account.Upload,
			*account.Download,
			*account.Quota,
			*account.ExpireTime/1000)

		clashConfig := bo.ClashConfig{}
		var ClashConfigInterface []interface{}

		port, err := strconv.ParseUint(strings.Split(*hysteria2Config.Listen, ":")[1], 10, 64)
		if err != nil {
			return "", "", err
		}

		hysteria2 := bo.Hysteria2{
			Name:     "hysteria2",
			Type:     "hysteria2",
			Server:   strings.Split(host, ":")[0],
			Port:     uint(port),
			Password: conPass,
		}
		if hysteria2Config.Obfs != nil &&
			hysteria2Config.Obfs.Type != nil &&
			*hysteria2Config.Obfs.Type == "salamander" &&
			hysteria2Config.Obfs.Salamander != nil &&
			hysteria2Config.Obfs.Salamander.Password != nil &&
			*hysteria2Config.Obfs.Salamander.Password != "" {
			hysteria2.Obfs = "salamander"
			hysteria2.ObfsPassword = *hysteria2Config.Obfs.Salamander.Password
		}

		if hysteria2Config.Bandwidth != nil {
			if hysteria2Config.Bandwidth.Up != nil &&
				*hysteria2Config.Bandwidth.Up != "" {
				up, err := strconv.ParseUint(strings.Split(*hysteria2Config.Bandwidth.Up, " ")[0], 10, 64)
				if err != nil {
					return "", "", err
				}

				hysteria2.Up = int(up)
			}
			if hysteria2Config.Bandwidth.Down != nil &&
				*hysteria2Config.Bandwidth.Down != "" {
				down, err := strconv.ParseUint(strings.Split(*hysteria2Config.Bandwidth.Down, " ")[0], 10, 64)
				if err != nil {
					return "", "", err
				}

				hysteria2.Down = int(down)
			}
		}

		ClashConfigInterface = append(ClashConfigInterface, hysteria2)

		proxyGroups := make([]bo.ProxyGroup, 0)
		proxyGroup := bo.ProxyGroup{
			Name:    "PROXY",
			Type:    "select",
			Proxies: []string{"hysteria2"},
		}

		proxyGroups = append(proxyGroups, proxyGroup)
		clashConfig.ProxyGroups = proxyGroups
		clashConfig.Proxies = ClashConfigInterface

		clashConfigYaml, err := yaml.Marshal(&clashConfig)
		if err != nil {
			return "", "", err
		}
		configYaml = string(clashConfigYaml)
	}

	return userInfo, configYaml, nil
}

func Hysteria2Url(clientType string, accountId int64, hostname string) (string, error) {
	hysteria2Config, err := GetHysteria2Config()
	if err != nil {
		return "", err
	}
	if hysteria2Config.Listen == nil || *hysteria2Config.Listen == "" {
		return "", errors.New("hysteria2 config is empty")
	}

	account, err := dao.GetAccount("id = ?", accountId)
	if err != nil {
		return "", err
	}

	urlConfig := ""
	if hysteria2Config.Obfs != nil &&
		hysteria2Config.Obfs.Type != nil &&
		*hysteria2Config.Obfs.Type == "salamander" &&
		hysteria2Config.Obfs.Salamander != nil &&
		hysteria2Config.Obfs.Salamander.Password != nil &&
		*hysteria2Config.Obfs.Salamander.Password != "" {
		urlConfig += fmt.Sprintf("&obfs=salamander&obfs-password=%s", *hysteria2Config.Obfs.Salamander.Password)
	}

	var insecure int64 = 0
	if hysteria2Config.TLS != nil &&
		hysteria2Config.TLS.Cert != nil &&
		*hysteria2Config.TLS.Cert != "" &&
		hysteria2Config.TLS.Key != nil &&
		*hysteria2Config.TLS.Key != "" {
		insecure = 1
	}
	urlConfig += fmt.Sprintf("&insecure=%d", insecure)

	if hysteria2Config.ACME != nil &&
		hysteria2Config.ACME.Domains != nil &&
		len(hysteria2Config.ACME.Domains) > 0 {
		if clientType == constant.Shadowrocket {
			urlConfig += fmt.Sprintf("&peer=%s", hysteria2Config.ACME.Domains[0])
		} else {
			urlConfig += fmt.Sprintf("&sni=%s", hysteria2Config.ACME.Domains[0])
		}
	}

	if hysteria2Config.Bandwidth != nil &&
		hysteria2Config.Bandwidth.Down != nil &&
		*hysteria2Config.Bandwidth.Down != "" {
		urlConfig += fmt.Sprintf("&downmbps=%s", url.QueryEscape(*hysteria2Config.Bandwidth.Down))
	}

	hysteria2ConfigPortHopping, err := dao.GetConfig("key = ?", constant.Hysteria2ConfigPortHopping)
	if err != nil {
		return "", err
	}
	port := *hysteria2Config.Listen
	if *hysteria2ConfigPortHopping.Value != "" {
		if clientType == constant.Shadowrocket {
			urlConfig += fmt.Sprintf("&mport=%s", *hysteria2ConfigPortHopping.Value)
		} else {
			port = ":" + *hysteria2ConfigPortHopping.Value
		}
	}

	hysteria2ConfigRemark, err := dao.GetConfig("key = ?", constant.Hysteria2Remark)
	if err != nil {
		return "", err
	}
	if *hysteria2ConfigRemark.Value != "" {
		urlConfig += fmt.Sprintf("#%s", *hysteria2ConfigRemark.Value)
	}
	if urlConfig != "" {
		urlConfig = "?" + strings.TrimPrefix(urlConfig, "&")
	}
	return fmt.Sprintf("hysteria2://%s@%s%s", *account.ConPass, hostname, port) + urlConfig, nil
}
