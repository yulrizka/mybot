package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/uber-go/zap"
	"github.com/yulrizka/bot"
)

var (
	botName              string
	log                  zap.Logger
	logLevel             int
	httpTimeout          = 15 * time.Second
	telegramInBufferSize = 10000
	startedAt            time.Time
	adminID              = ""

	// compiled time information
	VERSION   = ""
	BUILDTIME = ""
)

type logger struct {
	zap.Logger
}

func (l logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

func init() {
	log = logger{zap.NewJSON(zap.AddCaller(), zap.AddStacks(zap.FatalLevel))}
}

func main() {
	logLevel := zap.LevelFlag("v", zap.InfoLevel, "log level: all, debug, info, warn, error, panic, fatal, none")
	flag.StringVar(&botName, "botname", "satpam_bot", "bot name")
	flag.StringVar(&adminID, "admin", "", "admin id")
	flag.Parse()

	// setup logger
	log.SetLevel(*logLevel)
	bot.SetLogger(log)
	log.Info("STARTED", zap.String("version", VERSION), zap.String("buildtime", BUILDTIME))

	key := os.Getenv("TELEGRAM_KEY")
	if key == "" {
		log.Fatal("TELEGRAM_KEY can not be empty")
	}

	startedAt = time.Now()
	telegram := bot.NewTelegram(key)
	plugin := satpamBot{t: telegram}
	if err := telegram.AddPlugin(&plugin); err != nil {
		log.Fatal("Failed AddPlugin", zap.Error(err))
	}
	plugin.start()
	telegram.Start()

}

type satpamBot struct {
	// channel to communicate with telegram
	in  chan interface{}
	out chan bot.Message

	t *bot.Telegram
}

func (*satpamBot) Name() string {
	return "satpamBot"
}

func (b *satpamBot) Init(out chan bot.Message) (in chan interface{}, err error) {
	b.in = make(chan interface{}, telegramInBufferSize)
	b.out = out

	return b.in, nil
}

func (b *satpamBot) start() {
	go b.handleInbox()
}

// handleInbox handles incomming chat message
func (b *satpamBot) handleInbox() {
	for {
		select {
		case rawMsg := <-b.in:
			if rawMsg == nil {
				log.Fatal("handleInbox input channel is closed")
			}
			switch msg := rawMsg.(type) {
			case *bot.Message:
				if msg.Date.Before(startedAt) {
					// ignore message that is received before the process started
					log.Debug("message before started at", zap.Object("msg", msg), zap.String("startedAt", startedAt.String()), zap.String("date", msg.Date.String()))
					continue
				}
				log.Debug("handleInbox got message", zap.Object("msg", msg))

				if msg.From.ID != adminID {
					continue
				}

				msgType := msg.Chat.Type
				if msgType == bot.Private {
					log.Debug("Got private message", zap.Object("msg", msg))
					if msg.From.ID == adminID {
						// TODO
					}
					continue
				}

				// ## Handle Commands ##
				switch msg.Text {
				case "/leave", "/leave@" + botName:
					if b.cmdLeave(msg) {
						continue
					}
				}
			}
		}
	}
}

func (b *satpamBot) cmdLeave(msg *bot.Message) bool {
	b.t.Leave(msg.Chat.ID)
	fmt.Printf("leaving %s\n", msg.Chat.ID)
	return true
}
