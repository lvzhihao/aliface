package face

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/kataras/iris"
	"github.com/spf13/viper"
)

type FppSearchReturn struct {
	RequestId    string                  `json:"request_id"`
	Results      []FppSearchReturnResult `json:"results"`
	Thresholds   FppThresholds           `json:"thresholds"`
	ImageId      string                  `json:"image_id"`
	Faces        []FppFace               `json:"faces"`
	TimeUsed     int64                   `json:"time_used"`
	ErrorMessage string                  `json:"error_message"`
}

type FppSearchReturnResult struct {
	FaceToken  string  `json:"face_token"`
	Confidence float64 `json:"confidence"`
	UserId     string  `json:"user_id"`
}

type FppThresholds struct {
	E3 float64 `json:"1e-3"`
	E4 float64 `json:"1e-4"`
	E5 float64 `json:"1e-5"`
}

type FppFace struct {
	FaceToken     string           `json:"face_token"`
	FaceRectangle FppFaceRectangle `json:"face_rectangle"`
}

type FppFaceRectangle struct {
	Hieght int64 `json:"height"`
	Width  int64 `json:"width"`
	Top    int64 `json:"top"`
	Left   int64 `json:"left"`
}

func FacePlusPlus(ctx *iris.Context) {
	var image map[string]string
	err := ctx.ReadJSON(&image)
	if err != nil {
		ctx.JSON(200, map[string]string{"error": "image error"})
	} else {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		imageData, err := base64.StdEncoding.DecodeString(image["image"])
		if err != nil {
			log.Println(err)
			ctx.JSON(200, map[string]interface{}{"error": err})
		}
		f := bytes.NewBuffer(imageData)
		fw, err := w.CreateFormFile("image_file", "upload.jpg")
		if err != nil {
			log.Println(err)
			ctx.JSON(200, map[string]interface{}{"error": err})
		}
		if _, err = io.Copy(fw, f); err != nil {
			log.Println(err)
			ctx.JSON(200, map[string]interface{}{"error": err})
		}
		fw, _ = w.CreateFormField("api_key")
		fw.Write([]byte(viper.GetString("face++Key")))
		fw, _ = w.CreateFormField("api_secret")
		fw.Write([]byte(viper.GetString("face++Secret")))
		fw, _ = w.CreateFormField("outer_id")
		fw.Write([]byte("signin"))
		fw, _ = w.CreateFormField("return_result_count")
		fw.Write([]byte("5"))
		w.Close()

		req, _ := http.NewRequest("POST", "https://api-cn.faceplusplus.com/facepp/v3/search", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		res, err := client.Do(req)
		if err != nil {
			log.Println(err)
			ctx.JSON(200, map[string]interface{}{"error": err})
		}
		info, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
			ctx.JSON(200, map[string]interface{}{"error": err})
		}
		var result FppSearchReturn
		err = json.Unmarshal(info, &result)
		if err != nil {
			log.Println(err)
			ctx.JSON(200, map[string]interface{}{"error": err})
		}
		log.Println(result)
		if len(result.Results) > 0 {
			conn := Redis.Get()
			defer conn.Close()

			var score = 0.0
			var ret = ""
			for _, r := range result.Results {
				var name string
				name, _ = redis.String(conn.Do("GET", r.FaceToken))
				log.Println(name, r.Confidence)
				if r.Confidence > result.Thresholds.E4 && r.Confidence > score {
					score = r.Confidence
					ret = name
				}
			}
			log.Printf("匹配阈值 E5: %f, E4: %f, E3: %f\n", result.Thresholds.E5, result.Thresholds.E4, result.Thresholds.E3)
			log.Printf("匹配E4以上: %s\n", ret)
			ctx.JSON(200, map[string]interface{}{
				"data": map[string]interface{}{
					"local": result.Faces[0].FaceRectangle,
					"score": score,
					"name":  ret,
				},
			})
		} else {
			ctx.JSON(200, map[string]interface{}{"error": "no rsult"})
		}
	}
}
