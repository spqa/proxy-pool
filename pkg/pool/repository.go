package pool

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
	"h12.io/socks"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Type string

const (
	Socks4 Type = "socks4"
	Socks5 Type = "socks5"
	Http   Type = "http"
	Https  Type = "https"
)

var (
	ErrEmptyPool = errors.New("empty pool")
)

func parseType(t string) (Type, error) {
	t = strings.TrimSpace(t)
	t = strings.ToLower(t)
	switch t {
	case "socks4":
		return Socks4, nil
	case "socks5":
		return Socks5, nil
	case "http":
		return Http, nil
	case "https":
		return Https, nil
	default:
		return "", errors.New("unknown protocol " + t)
	}
}

const indexKey = "index:proxy:set"

type entity struct {
	Ip       string `mapstructure:"ip"`
	Port     int    `mapstructure:"port"`
	Type     Type   `mapstructure:"type"`
	Country  string `mapstructure:"country"`
	Latency  int    `mapstructure:"latency"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (e *entity) GetProxyUri() string {
	if e.Type == Socks5 || e.Type == Socks4 {
		return fmt.Sprintf("%v://%v:%v?timeout=5s", e.Type, e.Ip, e.Port)
	}
	if e.Username != "" && e.Password != "" {
		return fmt.Sprintf("%v://%v:%v@%v:%v", "http", e.Username, e.Password, e.Ip, e.Port)
	} else {
		return fmt.Sprintf("%v://%v:%v", "http", e.Ip, e.Port)
	}
}

func (e *entity) GetDialFunc() func(string, string) (net.Conn, error) {
	switch e.Type {
	case Socks4:
		return socks.Dial(e.GetProxyUri())
	case Socks5:
		return socks.Dial(e.GetProxyUri())
	case Http:
		return func(network, addr string) (net.Conn, error) {
			// create connect req to pass to upper proxy
			connectReq := &http.Request{
				Method: "CONNECT",
				URL:    &url.URL{Opaque: addr},
				Host:   addr,
				Header: map[string][]string{
					"Proxy-Authorization": {"Basic " + base64.URLEncoding.EncodeToString([]byte(e.Username+":"+e.Password))},
				},
			}
			// open connection to upper proxy
			c, err := net.DialTimeout(network, fmt.Sprintf("%v:%v", e.Ip, e.Port), time.Second*5)
			if err != nil {
				return nil, err
			}
			// send the connect req to connection
			err = connectReq.Write(c)
			if err != nil {
				return nil, err
			}
			// Read response.
			// Okay to use and discard buffered reader here, because
			// TLS server will not speak until spoken to.
			br := bufio.NewReader(c)
			resp, err := http.ReadResponse(br, connectReq)
			if err != nil {
				_ = c.Close()
				return nil, err
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				resp, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return nil, err
				}
				_ = c.Close()
				return nil, errors.New("proxy refused connection" + string(resp))
			}
			return c, nil
		}
	case Https:
		return func(network, addr string) (net.Conn, error) {
			connectReq := &http.Request{
				Method: "CONNECT",
				URL:    &url.URL{Opaque: addr},
				Host:   addr,
				Header: map[string][]string{
					"Proxy-Authorization": {"Basic " + base64.URLEncoding.EncodeToString([]byte(e.Username+":"+e.Password))},
				},
			}

			c, err := net.Dial(network, fmt.Sprintf("%v:%v", e.Ip, e.Port))
			if err != nil {
				return nil, err
			}
			c = tls.Client(c, &tls.Config{
				InsecureSkipVerify: true,
			})

			err = connectReq.Write(c)
			if err != nil {
				return nil, err
			}
			// Read response.
			// Okay to use and discard buffered reader here, because
			// TLS server will not speak until spoken to.
			br := bufio.NewReader(c)
			resp, err := http.ReadResponse(br, connectReq)
			if err != nil {
				c.Close()
				return nil, err
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 500))
				if err != nil {
					return nil, err
				}
				c.Close()
				return nil, errors.New("proxy refused connection" + string(body))
			}
			return c, nil
		}
	default:
		return nil
	}
}

type repository struct {
	redis *redis.Client
}

func NewRepository(client *redis.Client) *repository {
	return &repository{redis: client}
}

func (r repository) saveMany(ctx context.Context, entities []*entity) error {
	pipeline := r.redis.Pipeline()
	for _, e := range entities {
		m := map[string]interface{}{
			"ip":       e.Ip,
			"port":     strconv.Itoa(e.Port),
			"type":     string(e.Type),
			"country":  e.Country,
			"latency":  strconv.Itoa(e.Latency),
			"username": e.Username,
			"password": e.Password,
		}
		// save to hash
		pipeline.HMSet(ctx, buildKeyName(e), m)
		// add index
		pipeline.SAdd(ctx, indexKey, buildKeyName(e))
	}
	_, err := pipeline.Exec(ctx)
	return err
}

func (r repository) delete(ctx context.Context, entity *entity) error {
	pipeline := r.redis.Pipeline()
	// remove from index
	pipeline.SRem(ctx, indexKey, buildKeyName(entity))
	// remove hash
	pipeline.Del(ctx, buildKeyName(entity))
	_, err := pipeline.Exec(ctx)
	return err
}

func (r repository) getByRandom(ctx context.Context, count int64) ([]*entity, error) {
	// get random key name from index
	key, err := r.redis.SRandMemberN(ctx, indexKey, count).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrEmptyPool
		}
		return nil, err
	}

	// load data of that key
	var entities []*entity
	for _, k := range key {
		result, err := r.redis.HGetAll(ctx, k).Result()
		if err != nil {
			return nil, err
		}
		e := entity{}
		decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:           &e,
			WeaklyTypedInput: true,
		})
		err = decoder.Decode(result)
		if err != nil {
			return nil, err
		}
		entities = append(entities, &e)
	}

	return entities, nil
}

func buildKeyName(e *entity) string {
	return fmt.Sprintf("proxy:%v:%v", e.Ip, e.Port)
}
