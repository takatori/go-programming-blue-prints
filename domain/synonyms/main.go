package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/takatori/go-programming-blue-prints/thesaurus"
)

func main() {
	// https://words.bighugelabs.com/getkey.php
	// ここからユーザ登録してAPIキーを取得する必要がある
	apiKey := os.Getenv("BHT_APIKEY")
	thesaurus := &thesaurus.BigHuge{APIKey: apiKey}
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		word := s.Text()
		syns, err := thesaurus.Synonyms(word)
		if err != nil {
			log.Fatalf("%qの類語検索に失敗しました: %v\n", word, err)
		}
		if len(syns) == 0 {
			log.Fatalf("%qに類語はありませんでした\n", word)
		}
		for _, syn := range syns {
			fmt.Println(syn)
		}
	}
}
