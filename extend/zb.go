package extend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nntaoli-project/goex"
	"strings"
	"sync"
)

type req struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

type resp struct {
	Date    string          `json:"date"`
	Channel string          `json:"channel"`
	Ticker  json.RawMessage `json:"ticker"`
	ct      [2]string
}

func (r *resp) ParseChannel() bool {
	if r.ct[0] == "" && r.ct[1] == "" {
		if strings.Contains(r.Channel, "_") {
			ret := strings.SplitN(r.Channel, "_", 2)
			r.ct[0] = ret[0]
			r.ct[1] = ret[1]
			return true
		}
		return false
	}
	return true
}

var symbolCheck = []string{"usdt", "usd", "pax", "btc", "eth", "etc"}

func (r *resp) GetPair() goex.CurrencyPair {
	for _, symbol := range symbolCheck {
		if strings.Contains(r.ct[0], symbol) {
			var s bytes.Buffer
			symbolX := strings.Trim(r.ct[0], symbol)
			if strings.HasSuffix(r.ct[0], symbol) {
				// usdt在后
				s.WriteString(symbolX)
				s.WriteString("_")
				s.WriteString(symbol)
			} else {
				// usdt在前
				s.WriteString(symbol)
				s.WriteString("_")
				s.WriteString(symbolX)
			}
			return goex.NewCurrencyPair2(strings.ToUpper(s.String()))
		}
	}
	return goex.UNKNOWN_PAIR
}

type SpotWs struct {
	c            *goex.WsConn
	once         sync.Once
	wsBuilder    *goex.WsBuilder
	reqId        int
	depthCallFn  func(depth *goex.Depth)
	tickerCallFn func(ticker *goex.Ticker)
	tradeCallFn  func(trade *goex.Trade)
}

func NewSpotWsZB() *SpotWs {
	spotWs := &SpotWs{}
	spotWs.wsBuilder = goex.NewWsBuilder().
		WsUrl("wss://api.zb.land/websocket").
		ProtoHandleFunc(spotWs.handle).AutoReconnect()
	spotWs.reqId = 1

	return spotWs
}

func (s *SpotWs) connect() {
	s.once.Do(func() {
		s.c = s.wsBuilder.Build()
	})
}

func (s *SpotWs) DepthCallback(f func(depth *goex.Depth)) {
	s.depthCallFn = f
}

func (s *SpotWs) TickerCallback(f func(ticker *goex.Ticker)) {
	s.tickerCallFn = f
}

func (s *SpotWs) TradeCallback(f func(trade *goex.Trade)) {
	s.tradeCallFn = f
}

func (s *SpotWs) SubscribeDepth(pair goex.CurrencyPair) error {
	defer func() {
		s.reqId++
	}()

	s.connect()

	return s.c.Subscribe(req{
		Event:   "addChannel",
		Channel: pair.ToLower().ToSymbol("") + "_depth",
	})
}

func (s *SpotWs) SubscribeTicker(pair goex.CurrencyPair) error {
	defer func() {
		s.reqId++
	}()

	s.connect()

	return s.c.Subscribe(req{
		Event:   "addChannel",
		Channel: pair.ToLower().ToSymbol("") + "_ticker",
	})
}

func (s *SpotWs) SubscribeTrade(pair goex.CurrencyPair) error {
	panic("implement me")
}

func (s *SpotWs) handle(data []byte) error {
	var r resp
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	if ok := r.ParseChannel(); !ok {
		return fmt.Errorf("Parse channel filed error: %s ", r.Channel)
	}
	switch r.ct[1] {
	case "ticker":
		return s.tickerHandle(&r)
	case "depth":
		return s.depthHandle(&r)
	}
	return nil
}

func (s *SpotWs) depthHandle(_ *resp) error {
	panic("implement me")
	return nil
}

func (s *SpotWs) tickerHandle(r *resp) error {
	var (
		tickerData map[string]string
		ticker     goex.Ticker
	)

	err := json.Unmarshal(r.Ticker, &tickerData)
	if err != nil {
		return err
	}

	ticker.Pair = r.GetPair()
	ticker.Vol = goex.ToFloat64(tickerData["vol"])
	ticker.Last = goex.ToFloat64(tickerData["last"])
	ticker.Sell = goex.ToFloat64(tickerData["sell"])
	ticker.Buy = goex.ToFloat64(tickerData["buy"])
	ticker.High = goex.ToFloat64(tickerData["high"])
	ticker.Low = goex.ToFloat64(tickerData["low"])
	ticker.Date = goex.ToUint64(r.Date)

	s.tickerCallFn(&ticker)

	return nil
}
