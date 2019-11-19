package lib

import (
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	c *http.Client
	o sync.Once
)

func getClient() *http.Client {
	o.Do(func() {
		c = &http.Client{}
		jar, _ := cookiejar.New(nil)
		//绑定session
		c.Jar = jar
	})
	return c
}

func Login(userInfo *UserInfo) error {
	client := getClient()

	// 请求获得必要参数
	resp, err := client.Get("https://pass.neu.edu.cn/tpass/login?service=https%3A%2F%2Fipgw.neu.edu.cn%2Fsrun_cas.php%3Fac_id%3D1")
	if err != nil {
		return errors.New(ErrorCheckConnect)
	}

	// 读取响应内容
	res, err := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()
	body := string(res)

	// 读取lt post_url
	ltExp := regexp.MustCompile(`name="lt" value="(.+?)"`)
	lt := ltExp.FindAllStringSubmatch(body, -1)[0][1]

	postUrlExp := regexp.MustCompile(`id="loginForm" action="(.+?)"`)
	postUrl := postUrlExp.FindAllStringSubmatch(body, -1)[0][1]

	// 拼接data
	data := "rsa=" + userInfo.Username + userInfo.Password + lt +
		"&ul=" + strconv.Itoa(len(userInfo.Username)) +
		"&pl=" + strconv.Itoa(len(userInfo.Password)) +
		"&lt=" + lt +
		"&execution=e1s1" +
		"&_eventId=submit"

	// 构造请求
	req, _ := http.NewRequest("POST", "https://pass.neu.edu.cn"+postUrl, strings.NewReader(data))

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", "pass.neu.edu.cn")
	req.Header.Add("Origin", "https://pass.neu.edu.cn")
	req.Header.Add("Referer", "https://pass.neu.edu.cn/tpass/login?service=https%3A%2F%2Fipgw.neu.edu.cn%2Fsrun_cas.php%3Fac_id%3D3")

	// 发送请求
	resp, err = client.Do(req)
	if err != nil {
		return errors.New(ErrorCheckConnect)
	}

	// 读取响应内容
	res, err = ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()
	body = string(res)

	// 挂载IP信息
	ipExp := regexp.MustCompile(`get_online_info\('(.+?)'\)`)
	ip := ipExp.FindAllStringSubmatch(body, -1)
	if len(ip) == 0 {
		return errors.New(ErrorLoginFail)
	}
	userInfo.IP = ip[0][1]

	// 挂载SID信息
	sidExp := regexp.MustCompile(`do_drop\('(.+?)'\)`)
	sidList := sidExp.FindAllStringSubmatch(body, -1)
	sid := sidList[len(sidList)-1][1]
	if sid == "" {
		return errors.New(ErrorLoginFail)
	}
	userInfo.SID = sid

	return nil
}

func GetAccountInfo(userInfo *UserInfo) error {
	// 检查是否登陆
	if userInfo.IP == "" {
		return errors.New(ErrorNotLoginYet)
	}

	// 获取client实例
	client := getClient()

	// 构造请求
	k := strconv.Itoa(rand.Intn(100000 + 1))
	url := "https://ipgw.neu.edu.cn/include/auth_action.php?k=" + k
	data := "action=get_online_info&key=" + k

	req, _ := http.NewRequest("POST", url, strings.NewReader(data))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Add("Host", "pass.neu.edu.cn")
	req.Header.Add("Origin", "https://pass.neu.edu.cn")
	req.Header.Add("Referer", "https://ipgw.neu.edu.cn/srun_cas.php?ac_id=1")

	// 发送请求
	resp, err := client.Do(req)
	// 发送请求
	if err != nil {
		return errors.New(ErrorCheckConnect)
	}

	// 读取响应内容
	res, err := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()
	body := string(res)

	// 解析响应
	split := strings.Split(body, ",")
	if len(split) != 6 {
		return errors.New(ErrorLoginFail)
	}
	userInfo.Used, err = strconv.Atoi(split[0])
	userInfo.Time, err = strconv.Atoi(split[1])
	userInfo.Balance, err = strconv.ParseFloat(split[2], 64)

	return nil
}