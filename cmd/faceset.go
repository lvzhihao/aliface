// Copyright © 2017 edwin <edwin.lzh@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/lvzhihao/aliface/face"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// facesetCmd represents the faceset command
var facesetCmd = &cobra.Command{
	Use:   "faceset",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		face.ConnectRedis(viper.GetString("redis_url"))
		conn := face.Redis.Get()
		defer conn.Close()
		//测试redis是否连接成功
		_, err := conn.Do("PING")
		if err != nil {
			log.Fatal(err.Error())
		}

		forceDeleteFaceSet()
		createFaceSet()

		dirname := "./import_success"

		dir, _ := os.Open(dirname)
		fileInfo, _ := dir.Readdir(0)
		for _, file := range fileInfo {
			i := strings.Split(file.Name(), " ")
			na := strings.Split(i[2], ".")
			//dep := i[0]
			//flag := i[1]
			name := na[0]
			res, err := detectFaceToken(dirname + "/" + file.Name())
			if err != nil {
				log.Printf("%s detect: %s", name, err)
			} else {
				addFaceSet(name, res)
				log.Printf("%s detect: success", name)
			}
		}
		detailFaceSet()
	},
}

func addFaceSet(name string, token []string) error {
	apiKey := viper.GetString("face++Key")
	apiSecret := viper.GetString("face++Secret")
	apiUrl := "https://api-cn.faceplusplus.com/facepp/v3/faceset/addface"

	form := url.Values{}
	form.Add("api_key", apiKey)
	form.Add("api_secret", apiSecret)
	form.Add("outer_id", "signin")
	form.Add("face_tokens", strings.Join(token, ","))
	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	result := make(map[string]interface{}, 0)
	err = json.Unmarshal(info, &result)
	if err != nil {
		return err
	}
	if err, ok := result["error_message"]; ok {
		return fmt.Errorf("%s", err)
	}
	conn := face.Redis.Get()
	defer conn.Close()
	for _, t := range token {
		_, err = redis.String(conn.Do("SET", t, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func detectFaceToken(filename string) ([]string, error) {
	apiKey := viper.GetString("face++Key")
	apiSecret := viper.GetString("face++Secret")
	apiUrl := "https://api-cn.faceplusplus.com/facepp/v3/detect"

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fw, err := w.CreateFormFile("image_file", filename)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return nil, err
	}
	fw, _ = w.CreateFormField("api_key")
	fw.Write([]byte(apiKey))
	fw, _ = w.CreateFormField("api_secret")
	fw.Write([]byte(apiSecret))
	w.Close()

	req, _ := http.NewRequest("POST", apiUrl, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	result := make(map[string]interface{}, 0)
	err = json.Unmarshal(info, &result)
	if err != nil {
		return nil, err
	}
	if faces, ok := result["faces"]; ok {
		b, _ := json.Marshal(faces)
		facesRes := make([]map[string]interface{}, 0)
		err := json.Unmarshal(b, &facesRes)
		if err != nil {
			return nil, err
		} else {
			if len(facesRes) > 0 {
				var tokens []string
				for _, v := range facesRes {
					tokens = append(tokens, (v["face_token"]).(string))
				}
				return tokens, nil
			} else {
				return nil, fmt.Errorf("无法识别出人脸")
			}
		}
	} else {
		return nil, fmt.Errorf("error_message: %s", result["error_message"])
	}
}

func detailFaceSet() {
	apiKey := viper.GetString("face++Key")
	apiSecret := viper.GetString("face++Secret")
	apiUrl := "https://api-cn.faceplusplus.com/facepp/v3/faceset/getdetail"

	form := url.Values{}
	form.Add("api_key", apiKey)
	form.Add("api_secret", apiSecret)
	form.Add("outer_id", "signin")
	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	result := make(map[string]interface{}, 0)
	json.Unmarshal(info, &result)
	log.Printf("total count: %0.0f", result["face_count"])
}

func createFaceSet() {
	apiKey := viper.GetString("face++Key")
	apiSecret := viper.GetString("face++Secret")
	createApi := "https://api-cn.faceplusplus.com/facepp/v3/faceset/create"

	form := url.Values{}
	form.Add("api_key", apiKey)
	form.Add("api_secret", apiSecret)
	form.Add("outer_id", "signin")
	req, _ := http.NewRequest("POST", createApi, bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s\n", info)
}

func forceDeleteFaceSet() {
	deleteApi := "https://api-cn.faceplusplus.com/facepp/v3/faceset/delete"
	apiKey := viper.GetString("face++Key")
	apiSecret := viper.GetString("face++Secret")
	form := url.Values{}
	form.Add("api_key", apiKey)
	form.Add("api_secret", apiSecret)
	form.Add("outer_id", "signin")
	form.Add("check_empty", "0")
	req, _ := http.NewRequest("POST", deleteApi, bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s\n", info)
}

func init() {
	RootCmd.AddCommand(facesetCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// facesetCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// facesetCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
