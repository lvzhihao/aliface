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
	"log"
	"runtime"
	"time"

	"github.com/iris-contrib/graceful"
	"github.com/iris-contrib/middleware/loggerzap"
	"github.com/iris-contrib/middleware/recovery"
	"github.com/kataras/iris"
	"github.com/lvzhihao/aliface/face"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// apiCmd represents the api command
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		runtime.GOMAXPROCS(2)
		face.ConnectRedis(viper.GetString("redis_url"))
		conn := face.Redis.Get()
		defer conn.Close()
		//测试redis是否连接成功
		_, err := conn.Do("PING")
		if err != nil {
			log.Fatal(err.Error())
		}

		app := iris.New(iris.Configuration{IsDevelopment: true})

		//global recovery
		app.Use(loggerzap.New(loggerzap.Config{
			Status: true,
			IP:     true,
			Method: true,
			Path:   true,
		}))
		app.Use(recovery.Handler)
		app.StaticWeb("/public", "./bower_components/")
		app.StaticWeb("/web", "./web/")

		app.Get("/detection", face.DetectionUpload)

		//fetch sign
		app.Post("/api/token", face.Token)

		//rest
		app.Post("/api/detection", face.Detection)

		graceful.Run(viper.GetString("api_host")+":"+viper.GetString("api_port"), time.Duration(10)*time.Second, app)
	},
}

func init() {
	RootCmd.AddCommand(apiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// apiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// apiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
