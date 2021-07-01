package zb

import (
	"fmt"
	"github.com/nntaoli-project/goex"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestZb(t *testing.T) {
	ws := NewSpotWs()
	_ = ws.SubscribeTicker(goex.NewCurrencyPair2("USDT_QC"))
	go func() {
		ws.TickerCallback(func(t *goex.Ticker) {
			fmt.Println(t.Last)
		})
	}()
	WaitQuit()
}

func WaitQuit() {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("End.")
}
