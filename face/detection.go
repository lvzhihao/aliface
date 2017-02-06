package face

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/kataras/iris"
	"github.com/spf13/viper"
)

var ConstObjs []MemberObject

var inputs []Input

type Input struct {
	Image InputImage `json:"image"`
	Type  InputType  `json:"type"`
}

type InputImage struct {
	DataType  int64  `json:"dataType"`
	DataValue string `json:"dataValue"`
}

type InputType struct {
	DataType  int64 `json:"dataType"`
	DataValue int64 `json:"dataValue"`
}

type Result struct {
	Outputs []Output `json:"outputs"`
}

type Output struct {
	OutputLabel string      `json:"outputLabel"`
	OutputValue OutputValue `json:"outputValue"`
	OutputMulti interface{} `json:"outputMulti"`
}

type OutputValue struct {
	DataType  int64  `json:"dataType"`
	DataValue string `json:"dataValue"`
}

type DataValue struct {
	Errno  int64     `json:"errno"`
	Number int64     `json:"number"`
	Rect   []float64 `json:"rect"`
	Raw    []float64 `json:"raw"`
	Dense  []float64 `json:"dense"`
}

type MemberObject struct {
	Name  string
	Flag  string
	Dep   string
	Value DataValue
}

type CheckMember struct {
	Score  float64
	Member MemberObject
}

func init() {
	ConstObjs = make([]MemberObject, 0)
}

func DetectionUpload(ctx *iris.Context) {
	ctx.Render("upload/index.html", nil)
}

func Token(ctx *iris.Context) {
	var obj map[string]interface{}
	ctx.ReadJSON(&obj)
	log.Println(obj)
	ctx.JSON(200, map[string]string{"sucess": "sss"})
}

func Detection(ctx *iris.Context) {
	var dense []float64
	err := ctx.ReadJSON(&dense)
	if err != nil {
		log.Println(err)
		ctx.JSON(200, map[string]string{"error": "dense error"})
		return
	} else {
		//log.Println(dense)
		ret := CheckDetection(dense)
		scores := make(map[float64][]string, 0)
		scoresSort := make([]float64, 0)
		for _, cm := range ret {
			if _, ok := scores[cm.Score]; !ok {
				scores[cm.Score] = make([]string, 0)
			}
			scores[cm.Score] = append(scores[cm.Score], cm.Member.Name)
			scoresSort = append(scoresSort, cm.Score)
			sort.Float64s(scoresSort)
		}
		var limit int = 10
		var retName = "none"
		//log.Println(scoresSort)
		for _, s := range scoresSort {
			if s <= 300 && retName == "none" {
				retName = scores[s][0]
			}
			log.Println(strings.Join(scores[s], ",") + " score:" + strconv.Itoa(int(s)))
			limit--
			if limit == 0 {
				break
			}
		}
		log.Println("maybe: ", retName)
		ctx.JSON(200, map[string]string{"data": retName})
		return
	}
}

func CheckDetection(dense []float64) (ret []CheckMember) {
	start := time.Now().UnixNano()
	if len(ConstObjs) == 0 {
		conn := Redis.Get()
		defer conn.Close()
		vals, err := redis.StringMap(conn.Do("HGETALL", "detection"))
		for _, val := range vals {
			var obj MemberObject
			json.Unmarshal([]byte(val), &obj)
			ConstObjs = append(ConstObjs, obj)
		}
		if err != nil {
			return nil
		}
	}
	ch := make(chan CheckMember, len(ConstObjs))
	for _, obj := range ConstObjs {
		go func(obj MemberObject, ch chan CheckMember) {
			var score float64
			sch := make(chan float64, 256)
			for i := 0; i < 256; i++ {
				go func(i int, dense []float64, obj MemberObject, sch chan float64) {
					//log.Printf("%f - %f = %v", dense[i], obj.Value.Dense[i], math.Pow(dense[i]-obj.Value.Dense[i], 2))
					sch <- math.Pow(dense[i]-obj.Value.Dense[i], 2)
				}(i, dense, obj, sch)
			}
			for i := 0; i < 256; i++ {
				select {
				case fs := <-sch:
					score += fs
				}
			}
			ch <- CheckMember{Score: score, Member: obj}
		}(obj, ch)
	}
	for i := 0; i < len(ConstObjs); i++ {
		select {
		case obj := <-ch:
			ret = append(ret, obj)
			//log.Println(obj.Score)
		}
	}
	log.Printf("%d ms\n", (time.Now().UnixNano()-start)/1000/1000)
	return ret
}

func ImportDetection(name, flag, dep, dataValue string) error {
	b, err := AliFeatureDetection(dataValue)
	if err != nil {
		return err
	}
	//var ret map[string]interface{}
	var ret Result
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return err
	}
	var value DataValue
	err = json.Unmarshal([]byte(ret.Outputs[0].OutputValue.DataValue), &value)
	if err != nil {
		return err
	}
	if value.Number != 1 {
		return fmt.Errorf(name + " number error: " + strconv.Itoa(int(value.Number)))
	}
	var obj = MemberObject{
		Name:  name,
		Flag:  flag,
		Dep:   dep,
		Value: value,
	}
	b, _ = json.Marshal(obj)
	objStr := string(b)
	conn := Redis.Get()
	defer conn.Close()
	_, err = redis.Int(conn.Do("HSET", "detection", flag, objStr))
	if err != nil {
		return err
	}
	//log.Println(obj)
	return nil
}

func AliFeatureDetection(dataValue string) ([]byte, error) {
	var post = map[string]interface{}{"inputs": []interface{}{map[string]interface{}{"image": map[string]interface{}{"dataType": 50, "dataValue": dataValue}}}}
	b, _ := json.Marshal(post)
	req, _ := http.NewRequest("POST", "https://dm-24.data.aliyun.com/rest/160601/face/feature_detection.json", bytes.NewReader(b))
	req.Header.Set("Authorization", "APPCODE "+viper.GetString("app_code"))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	//log.Println(res)
	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	//log.Printf("result, %s\n", info)
	return info, nil

}
