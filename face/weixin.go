package face

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/kataras/iris"
	"github.com/lvzhihao/wechat/core"
	"github.com/spf13/viper"
	"github.com/uber-go/zap"
)

type JSSDKConfig struct {
	AppId     string
	TimeStamp int64
	NonceStr  string
	Sign      string
}

type Token struct {
	Token       string
	JsApiTicket string
	Expire      int64
}

var GlobalToken Token
var GlobalTokenLk sync.Mutex

func init() {
	GlobalToken = Token{
		Token:  "",
		Expire: 0,
	}
}

func getToken() Token {
	GlobalTokenLk.Lock()
	defer GlobalTokenLk.Unlock()
	if GlobalToken.Expire < time.Now().Unix() {
		for {
			t, err := core.GetAccessToken(viper.GetString("appid"), viper.GetString("appsecret"))
			if err == nil {
				logger.Info("Fetch AccessToken", zap.Object("token", t))
				GlobalToken.Token = t.AccessToken
				GlobalToken.Expire = time.Now().Unix() + t.ExpiresIn - 10 //fix
				rsp, _ := http.Get("https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=" + t.AccessToken + "&type=jsapi")
				b, _ := ioutil.ReadAll(rsp.Body)
				var result map[string]interface{}
				json.Unmarshal(b, &result)
				GlobalToken.JsApiTicket = ToString(result["ticket"])
				break
			} else {
				logger.Warn("Fetch AccessToken", zap.Error(err))
			}
		} //获取token
	}
	return GlobalToken
}

func WeixinUpload(ctx *iris.Context) {
	config := JSSDKConfig{
		AppId:     viper.GetString("appid"),
		TimeStamp: time.Now().Unix(),
		NonceStr:  RandStr(10),
		Sign:      "",
	}
	orgStr := "jsapi_ticket=" + getToken().JsApiTicket
	orgStr += "&noncestr=" + config.NonceStr
	orgStr += "&timestamp=" + strconv.Itoa(int(config.TimeStamp))
	orgStr += "&url=http://192.168.51.254:8977/weixin"
	config.Sign = fmt.Sprintf("%x", sha1.Sum([]byte(orgStr)))

	ctx.Render("weixin/upload.html", iris.Map{"config": config})
}
