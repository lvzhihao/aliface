package face

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/uber-go/zap"
)

var Redis *redis.Pool

var Logger = zap.New(
	zap.NewTextEncoder(zap.TextTimeFormat(time.RFC3339)),
	//zap.NewJSONEncoder(),
	//zap.Fields(zap.Stack()),
)
var logger = Logger

func ConnectRedis(server string) {
	Redis = &redis.Pool{
		MaxIdle:     10,
		MaxActive:   50,
		Wait:        true,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func ConnectRedisWithPasswd(server, password string) {
	Redis = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("AUTH", password); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func ToFloat(v interface{}) (float64, error) {
	return strconv.ParseFloat(ToString(v), 64)
}

func ToString(v interface{}) (s string) {
	switch v.(type) {
	case nil:
		return ""
	case string:
		s = v.(string)
	case []byte:
		s = string(v.([]byte))
	case io.Reader:
		b, _ := ioutil.ReadAll(v.(io.Reader))
		s = string(b)
	case error:
		s = v.(error).Error()
	default:
		b, err := json.Marshal(v)
		if err == nil {
			s = string(b)
		} else {
			s = fmt.Sprintf("%s", b)
		}
	}
	return
}

func RandStr(len int32) string {
	b := make([]byte, int(math.Ceil(float64(len)/2.0)))
	/*
		GoDoc
		Read generates len(p) random bytes and writes them into p. It always returns len(p) and a nil error. Read should not be called concurrently with any other Rand method.
	*/
	/*
		if _, err := rand.Read(b); err != nil {
			return "", err
		} else {
			return hex.EncodeToString(b)[0:len], nil
		}
	*/
	rand.Seed(time.Now().UnixNano())
	rand.Read(b)
	return hex.EncodeToString(b)[0:len]
}
