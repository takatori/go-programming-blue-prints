package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/go-oauth/oauth"
	"github.com/joeshaw/envdecode"
)

var conn net.Conn

// 接続を開く
func dial(netw, addr string) (net.Conn, error) {
	// connが閉じられているかを確認
	if conn != nil {
		conn.Close()
		conn = nil
	}
	netc, err := net.DialTimeout(netw, addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	conn = netc
	return netc, nil
}

var reader io.ReadCloser

func closeConn() {
	// 接続を閉じる
	if conn != nil {
		conn.Close()
	}
	// レスポンスの本体を読み込むために使われるio.ReadCloserも閉じる
	if reader != nil {
		reader.Close()
	}
}

var (
	authClient *oauth.Client
	creds      *oauth.Credentials
)

// 環境変数を読み込んでOAuthオブジェクトをセットアップする
func setupTwitterAuth() {
	// この構造体はここでしか使用されないため、インラインで定義
	var ts struct {
		ConsumerKey    string `env:"SP_TWITTER_KEY,required"`
		ConsumerSecret string `env:"SP_TWITTER_SECRET,required"`
		AccessToken    string `env:"SP_TWITTER_ACCESSTOKEN,required"`
		AccessSecret   string `env:"SP_TWITTER_ACCESSSECRET,required"`
	}
	if err := envdecode.Decode(&ts); err != nil {
		log.Fatalln(err)
	}
	creds = &oauth.Credentials{
		Token:  ts.AccessToken,
		Secret: ts.AccessSecret,
	}
	authClient = &oauth.Client{
		Credentials: oauth.Credentials{
			Token:  ts.ConsumerKey,
			Secret: ts.ConsumerSecret,
		},
	}
}

var (
	authSetupOnce sync.Once
	httpClient    *http.Client
)

// 認証されたリクエストを生成しレスポンスを受け取る
func makeRequest(req *http.Request, params url.Values) (*http.Response, error) {
	authSetupOnce.Do(func() {
		setupTwitterAuth()
		httpClient = &http.Client{
			Transport: &http.Transport{
				Dial: dial,
			},
		}
	})
	formEnc := params.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(formEnc)))
	req.Header.Set("Authorization", authClient.AuthorizationHeader(creds, "POST", req.URL, params))
	return httpClient.Do(req)
}

type tweet struct {
	Text string
}

// Twitterへの接続を開始してストリームから継続的にデータを読み込む
func readFromTwitter(votes chan<- string) { // votesは送信専用チャネル
	options, err := loadOptions()
	if err != nil {
		log.Println("選択肢の読み込みに失敗しました:", err)
		return
	}
	u, err := url.Parse("https://stream.twitter.com/1.1./statuses/filter.json")
	if err != nil {
		log.Println("URLの解析に失敗しました:", err)
		return
	}
	query := make(url.Values)
	query.Set("track", strings.Join(options, ","))
	req, err := http.NewRequest("POST", u.String(), strings.NewReader(query.Encode()))
	if err != nil {
		log.Println("検索のリクエストの作成に失敗しました:", err)
		return
	}
	resp, err := makeRequest(req, query)
	if err != nil {
		log.Println("検索のリクエストに失敗しました:", err)
		return
	}
	reader = resp.Body
	decoder := json.NewDecoder(reader)
	for {
		var tweet tweet
		if err := decoder.Decode(&tweet); err != nil {
			break
		}
		for _, option := range options {
			if strings.Contains(strings.ToLower(tweet.Text), strings.ToLower(option)) {
				log.Println("投票:", option)
				votes <- option
			}
		}
	}
}

// readFromTwitter関数を繰り返し呼び出す
func startTwitterStream(stopchan <-chan struct{}, votes chan<- string) <-chan struct{} {
	stoppedchan := make(chan struct{}, 1)
	go func(
		
	) {
		defer func() {
			stoppedchan <- struct{}{}
		}()
		for {
			select {
			case <-stopchan:
				log.Println("Twitterへの問い合わせを終了します...")
				return
			default:
				log.Println("Twitterに問い合わせます...")
				readFromTwitter(votes)
				log.Println(" (待機中)")
				time.Sleep(10 * time.Second) // 待機してから再接続します
			}
		}
	}()
	return stoppedchan
}
