package main

import (
	"time"
)

// messageは一つのメッセージを表します。
type message struct {
	Name    string
	Message string
	When    time.Time
}
