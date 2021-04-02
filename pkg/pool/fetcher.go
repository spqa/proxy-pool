package pool

import (
	"encoding/json"
	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
	"io"
	"net/http"
	"proxy-pool/pkg/log"
	"strconv"
)

type Fetcher interface {
	Get() ([]*entity, error)
	Name() string
}

type ProxyHubFetcher struct{}

func (p ProxyHubFetcher) Name() string {
	return "https://www.proxyhub.me"
}

func (p ProxyHubFetcher) Get() ([]*entity, error) {
	c := colly.NewCollector()

	var entities []*entity
	// Find and visit all links
	c.OnHTML("tbody", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(i int, element *colly.HTMLElement) {
			entity := new(entity)
			element.ForEach("td", func(i int, element *colly.HTMLElement) {
				switch i {
				case 0:
					entity.Ip = element.Text
					break
				case 1:
					port, err := strconv.Atoi(element.Text)
					if err != nil {
						log.Logger.Error("failed to parse port value", zap.String("value", element.Text), zap.Error(err))
						break
					}
					entity.Port = port
					break
				case 2:
					t, err := parseType(element.Text)
					if err != nil {
						log.Logger.Error("failed to parse type", zap.Error(err))
					}
					entity.Type = t
					break
				case 4:
					entity.Country = element.Text
				}
			})
			entities = append(entities, entity)
		})
	})

	c.OnRequest(func(r *colly.Request) {
		log.Logger.Info("visiting url", zap.String("url", r.URL.String()))
	})

	err := c.Visit("https://www.proxyhub.me/")
	return entities, err
}

type ProxyScanFetcher struct {
}

func (p ProxyScanFetcher) Get() ([]*entity, error) {
	client := http.Client{}
	resp, err := client.Get("https://www.proxyscan.io/api/proxy?ping=1000&limit=20&uptime=50&last_check=3600&country=vn,th,sg")
	if err != nil {
		return nil, err
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed []map[string]interface{}
	err = json.Unmarshal(bytes, &parsed)
	if err != nil {
		return nil, err
	}
	var result []*entity
	for _, v := range parsed {
		t, err := parseType((v["Type"].([]interface{}))[0].(string))
		if err != nil {
			return nil, err
		}
		result = append(result, &entity{
			Ip:       v["Ip"].(string),
			Port:     int(v["Port"].(float64)),
			Type:     t,
			Country:  "",
			Latency:  0,
			Username: "",
			Password: "",
		})
	}
	return result, nil
}

func (p ProxyScanFetcher) Name() string {
	return "https://proxyscan.io"
}
