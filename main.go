////////////////////////////////////////////////////////////////////////////
// Program: owc-insight
// Purpose: OpenWeChat Insight
// Authors: Tong Sun (c) 2020-2021, All rights reserved
////////////////////////////////////////////////////////////////////////////

package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/caarlos0/env"
	//"github.com/eatMoreApple/openwechat"
	"github.com/skip2/go-qrcode"
	"github.com/eatmoreapple/openwechat"
)

////////////////////////////////////////////////////////////////////////////
// Constant and data type/structure definitions

const desc = "OpenWeChat Insight"

type envConfig struct {
	LogLevel  string `env:"OWCI_LOG"`
	KaWait    int    `env:"OWCI_KA_WAIT" envDefault:"160"`   // keep-alive wait (in min)
	KaVariety int    `env:"OWCI_KA_VARIETY" envDefault:"30"` // ka variant (in min)
	KaBoost   int    `env:"OWCI_KA_BOOST" envDefault:"6"`    // ka boost, shorter-factor to devide the above two by
}

////////////////////////////////////////////////////////////////////////////
// Global variables definitions

var (
	progname = "owc-insight"
	version  = "0.2.0"
	date     = "2021-08-20"

	e envConfig

	timeStarted  time.Time
	lastReceived time.Time
	lastError    time.Time
	// lastReceived sync
	lrSync = &sync.Mutex{}

	chatie *openwechat.Mp

	ErrLoginFailed     = errors.New("login failed")
	ErrClientCheckLost = errors.New("ClientCheck lost")
)

////////////////////////////////////////////////////////////////////////////
// Function definitions

/*

   NOT WORKING!

type ResponseHooker struct{}

func (r ResponseHooker) BeforeRequest(req *http.Request) {}

func (r ResponseHooker) AfterRequest(response *http.Response, err error) {
	fmt.Println(response.Request.URL.Path)
	fmt.Println(response.Request.Header)
}

*/

//==========================================================================
// Main

func main() {
	// == Config handling
	err := env.Parse(&e)
	abortOn("Env config parsing error", err)
	if e.LogLevel != "" {
		di, err := strconv.ParseInt(e.LogLevel, 10, 8)
		abortOn("OWCI_LOG (int) parse error", err)
		debug = int(di)
	}

	logIf(0, desc,
		"Version", version,
		"Built-on", date,
	)
	logIf(0, "Copyright (C) 2020-2021, Tong Sun", "License", "MIT")
	logIf(0, "Program parameters",
		"log-level", e.LogLevel,
		"keep-alive-wait-min", e.KaWait,
		"keep-alive-variant-min", e.KaVariety,
		"keep-alive-boost-factor", e.KaBoost,
	)

	bot := openwechat.DefaultBot()
	//bot.Caller.Client.AddHttpHook(ResponseHooker{})
	// 注册登陆二维码回调
	bot.UUIDCallback = ConsoleQrCode
	// 注册消息处理函数
	bot.MessageHandler = textMessageHandle

	// 创建热存储容器对象
	reloadStorage := openwechat.NewJsonFileHotReloadStorage("storage.json")
	// 执行热登陆, 不定长参数设置为true, 可在登录凭证失效后进行扫码登录
	err = bot.HotLogin(reloadStorage, true)
	_abortOn("Can't start bot", err, 9)
	// 获取登陆的用户
	self, err := bot.GetCurrentUser()
	abortOn("Can't get self", err)
	logIf(0, "logged-on", "user", self)

	// == Start Scheduled Executor
	rand.Seed(time.Now().Unix())
	timeStarted = time.Now()
	lastReceived = timeStarted
	lastError = timeStarted
	go periodicHotReload(bot, self, reloadStorage)
	go periodicDogFeed(bot, self)

	postLogin(self)

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}

//==========================================================================
// support functions

func ConsoleQrCode(uuid string) {
	q, _ := qrcode.New("https://login.weixin.qq.com/l/"+uuid, qrcode.Low)
	fmt.Println(q.ToSmallString(true))
}

func postLogin(self *openwechat.Self) {
	mps := getMps(self, true, 1)
	logIf(1, "mps", "list", fmt.Sprintf("%v", mps))
	for k, mp := range mps {
		logIf(5, "公众号", "id", k, "rec", fmt.Sprintf("%#v\n", mp.User))
	}
	chatie = mps.SearchByNickName(1, "Chatie")[0]
	logIf(1, "keep-alive-with", "chatie", chatie.User)

	groups := getGroups(self, true, 2)
	logIf(2, "groups", "list", fmt.Sprintf("%v", groups))

	friends := getFriends(self, true, 3)
	logIf(3, "friends", "list", fmt.Sprintf("%v", friends))

	// WX ClientCheck from 微信团队 will come within seconds after login
	// wait for ~2 minutes to confirm their arrival
	go wxHandshakeCheck()
}

// 获取当前用户所有的公众号
func getMps(self *openwechat.Self, update bool, logLevel int) openwechat.Mps {
	mps, err := self.Mps(update)
	abortOn("Can't get mps", err)
	return mps
}

// 获取所有的群组(最新的)
func getGroups(self *openwechat.Self, update bool, logLevel int) openwechat.Groups {
	groups, err := self.Groups(update)
	abortOn("Can't get groups", err)
	return groups
}

// 获取所有的好友(最新的好友)
func getFriends(self *openwechat.Self, update bool, logLevel int) openwechat.Friends {
	friends, err := self.Friends(update)
	abortOn("Can't get friends", err)
	return friends
}
