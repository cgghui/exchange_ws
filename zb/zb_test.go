package zb

import (
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
		ws.TickerCallback(func(tk *goex.Ticker) {
			t.Log(tk.Last)
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
