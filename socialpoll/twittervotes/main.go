package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bitly/go-nsq"

	"gopkg.in/mgo.v2"
)

func main() {

	// シグナルのハンドリング
	var stoplock sync.Mutex
	stop := false
	stopChan := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)
	go func() {
		<-signalChan
		stoplock.Lock()
		stop = true
		stoplock.Unlock()
		log.Println("停止します...")
		stopChan <- struct{}{}
		closeConn()
	}()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// データベースへの接続を開始
	if err := dialdb(); err != nil {
		log.Fatalln("MongoDBへのダイヤルに失敗しました:", err)
	}
	defer closedb()

	// 処理を開始します
	votes := make(chan string)                                // 投票結果のためのチャネル
	publisherStoppedChan := publishVotes(votes)               // 停止のために使われるチャネル
	twitterStoppedChan := startTwitterStream(stopChan, votes) // 停止用シグナルのチャネル

	// 更新のためのgoroutineの開始
	// 定期的にcloseConnによる切断で
	// 調査の選択肢を毎回データベースから読み直し
	// 状態を最新に保つ
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			closeConn()
			stoplock.Lock()
			if stop {
				stoplock.Unlock()
				break
			}
			stoplock.Unlock()
		}
	}()

	<-twitterStoppedChan
	close(votes)
	<-publisherStoppedChan
}

var db *mgo.Session

// MongoDBに接続する
func dialdb() error {
	var err error
	log.Println("MongoDBにダイアル中: localhost")
	db, err = mgo.Dial("localhost")
	return err
}

// MongoDBへの接続を閉じる
func closedb() {
	db.Close()
	log.Println("データベース接続が閉じられました")
}

type poll struct {
	Options []string
}

// 投票を表すオブジェクトを読みこんで
// ドキュメントの中から投票の選択肢を全て取り出す
func loadOptions() ([]string, error) {
	var options []string
	iter := db.DB("ballots").C("polls").Find(nil).Iter()
	var p poll
	for iter.Next(&p) {
		options = append(options, p.Options...)
	}
	iter.Close()
	return options, iter.Err()
}

// 投票結果をNSQのトピックにパブリッシュする
func publishVotes(votes <-chan string) <-chan struct{} {
	stopchan := make(chan struct{}, 1)
	pub, _ := nsq.NewProducer("localhost:4150", nsq.NewConfig())
	go func() {
		for vote := range votes {
			pub.Publish("votes", []byte(vote)) // 投票内容をパブリッシュする
		}
		log.Println("Publisher: 停止中です")
		pub.Stop()
		log.Println("Publisher: 停止しました")
		stopchan <- struct{}{}
	}()
	return stopchan
}
