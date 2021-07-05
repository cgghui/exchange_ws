package main

import (
	"context"
	"github.com/cgghui/exchange_ws/db"
	"github.com/cgghui/exchange_ws/extend"
	"github.com/nntaoli-project/goex"
	"github.com/nntaoli-project/goex/builder"
	"log"
	"os"
	"sync"
	"time"
)

const (
	ak       = "2f6df565-f1ce-4bf8-b0ae-a352a144659c" // app key
	sk       = "8010662a-dd23-48b0-aaa8-0ce2fbb9147a" // secret key
	currency = "USDT_QC"                              // 交易对
	timeout  = 60                                     // 订单挂起超时时间
)

const (
	OrderDone    = "done"    // 订单成交成功
	OrderPending = "pending" // 订单挂起中
	OrderClose   = "close"   // 订单已关闭
	OrderTimeout = "timeout" // 订单已超时
	OrderEnd     = "end"     // 订单已结束
)

var RedisDefault = db.ConfigRedis{Addr: "172.22.67.40", Port: 6379, Auth: "", SelectDB: 0}
var Exchange goex.API
var proxyUrl string
var Pair = goex.NewCurrencyPair2(currency)
var ctx = context.Background()

func main() {
	if proxyUrl == "" {
		proxyUrl = os.Getenv("http_proxy")
		if proxyUrl == "" {
			proxyUrl = os.Getenv("https_proxy")
		}
	}
	if proxyUrl == "" {
		Exchange = extend.New(builder.DefaultAPIBuilder.GetHttpClient(), ak, sk)
	} else {
		Exchange = extend.New(builder.DefaultAPIBuilder.HttpProxy(proxyUrl).GetHttpClient(), ak, sk)
	}

	if err := db.ConnectRedis(&RedisDefault); err != nil {
		log.Fatalf("Connect Redis Fail: %s", err.Error())
	}

	g := &sync.WaitGroup{}

	t := time.NewTimer(0)
	for range t.C {
		data, err := db.Redis.HGetAll(ctx, "zb").Result()
		if err != nil {
			log.Printf("从Reids加载数据失败：%s", err.Error())
			continue
		}
		for id, status := range data {
			g.Add(1)
			go Handle(id, status, g)
		}
		g.Wait()
		t.Reset(time.Second)
	}
}

func Handle(id, status string, g *sync.WaitGroup) {
	defer g.Done()
	if status == OrderEnd {
		db.Redis.HDel(ctx, "zb", id)
	}
	if status == OrderPending {

		var (
			info *goex.Order
			err  error
		)

		info, err = Exchange.GetOneOrder(id, Pair)
		if err != nil {
			log.Printf("交易[%s] 获取信息获取失败 Error: %s", id, err.Error())
			return
		}

		// ## 时间
		oTime := time.Unix(int64(info.OrderTime/1000), 0).Local()
		cTime := time.Now()

		// ## 超时
		if info.Status == goex.ORDER_UNFINISH && cTime.Sub(oTime).Seconds() > timeout {
			if _, err = Exchange.CancelOrder(id, Pair); err != nil {
				log.Printf("交易%s[%s] 超时 取消失败，Error: %s", dir(info), id, err.Error())
				return
			}
			// 订单取消成功
			db.Redis.HSet(ctx, "zb", id, OrderTimeout)
			log.Printf("交易%s[%s] 超时 取消成功", dir(info), id)
			return
		}

		// ## 人为取消
		if info.Status == goex.ORDER_CANCEL {
			db.Redis.HSet(ctx, "zb", id, OrderClose)
			log.Printf("交易%s[%s] 订单已取消", dir(info), id)
			return
		}

		// ## 成交
		if info.Status == goex.ORDER_FINISH {
			db.Redis.HSet(ctx, "zb", id, OrderDone)
			log.Printf("交易%s[%s] 成功，提交价：%.4f 成交价：%.4f", dir(info), id, info.Price, info.AvgPrice)
			return
		}

	}
}

func dir(info *goex.Order) string {
	if info.Side == goex.BUY {
		return "[买]"
	} else {
		return "[卖]"
	}
}
