package quark

import (
	"github.com/imroc/req/v3"
	"net/http"
	"time"
)

var defaultUa = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) quark-cloud-drive/2.5.20 Chrome/100.0.4896.160 Electron/18.3.5.4-b478491100 Safari/537.36 Channel/pckk_other_ch"

type SessionRefresh func(session string)

type QuarkClient struct {
	pus           string
	puus          string
	sessionClient *req.Client
	defaultClient *req.Client
	pusRefresh    SessionRefresh
	puusRefresh   SessionRefresh
}

func NewClient(pus, puus string) *QuarkClient {
	client := &QuarkClient{
		pus:           pus,
		puus:          puus,
		sessionClient: initSessionClient(pus, puus),
		defaultClient: initDefaultClient(),
	}
	return client
}

func (c *QuarkClient) refreshPus(pus string) *req.Client {
	c.pus = pus
	if c.pusRefresh != nil {
		c.pusRefresh(pus)
	}
	return c.sessionClient.SetCommonCookies(&http.Cookie{Name: "__pus", Value: pus})
}

func (c *QuarkClient) refreshPuus(puus string) *req.Client {
	c.puus = puus
	if c.puusRefresh != nil {
		c.puusRefresh(puus)
	}
	return c.sessionClient.SetCommonCookies(&http.Cookie{Name: "__puus", Value: puus})
}

func initSessionClient(pus, puus string) *req.Client {
	sessionClient := req.C().
		SetCommonHeaders(map[string]string{
			"User-Agent": defaultUa,
			"Accept":     "application/json, text/plain, */*",
			"Referer":    "https://pan.quark.cn",
		}).
		SetCommonQueryParam("pr", "ucpro").
		SetCommonQueryParam("fr", "pc").
		SetCommonCookies(&http.Cookie{Name: "__pus", Value: pus}, &http.Cookie{Name: "__puus", Value: puus}).
		SetTimeout(30 * time.Minute).SetBaseURL("https://drive.quark.cn/1/clouddrive")
	return sessionClient
}

func initDefaultClient() *req.Client {
	sessionClient := req.C().SetTimeout(30 * time.Minute)
	return sessionClient
}
