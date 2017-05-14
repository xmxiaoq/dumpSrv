package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/spf13/afero"
	"github.com/spf13/cast"
	"github.com/uber-go/zap"
	"github.com/ungerik/go-dry"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/cors"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/websocket"
	irisLogger "gopkg.in/kataras/iris.v6/middleware/logger"
)

var logger zap.Logger
var FS = afero.Afero{Fs: afero.NewOsFs()}

func initLogger() {
	var logPath string
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("os.Executable()", err)
	} else {
		logPath = exe + ".log"
	}

	fmt.Println("log to", logPath)

	l := &lumberjack.Logger{Filename: logPath}
	logger = zap.New(
		zap.NewJSONEncoder(),
		//zap.NewTextEncoder(zap.TextTimeFormat(time.RFC3339)),
		zap.DebugLevel,
		//zap.DiscardOutput,
		zap.Output(zap.Tee(zap.AddSync(l), zap.AddSync(os.Stdout))),
	)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for {
			<-c
			l.Rotate()
		}
	}()
}

func init() {
	initLogger()

	_, _ = FS.Exists(os.Args[0] + ".log")
	_ = cast.ToString(100)
	_ = dry.SyncMap{}
}

func main() {
	logger.Info("Failed to fetch URL.404",
		zap.String(`"url"`, `this is = "url"`),
		zap.Int("attempt", 10),
		zap.Duration("backoff", time.Duration(100)),
	)

	app := iris.New()
	app.Adapt(iris.DevLogger()) // enable all (error) logs
	app.Adapt(httprouter.New()) // select the httprouter as the servemux
	//app.Adapt(view.HTML("./templates", ".html")) // select the html engine to serve templates

	ws := websocket.New(websocket.Config{
		// the path which the websocket client should listen/registered to,
		Endpoint: "/ws",
		// to enable binary messages (useful for protobuf):
		BinaryMessages: true,
		MaxMessageSize: 4096,
	})

	app.Adapt(ws) // adapt the websocket server, you can adapt more than one with different Endpoint

	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})

	app.Adapt(crs) // this line should be added

	customLogger := irisLogger.New(irisLogger.Config{
		// Status displays status code
		Status: true,
		// IP displays request's remote address
		IP: true,
		// Method displays the http method
		Method: true,
		// Path displays the request path
		Path: true,
	})

	app.Use(customLogger)

	//app.StaticWeb("/js", "./static/js") // serve our custom javascript code
	//app.StaticWeb("/pb", "./pb")
	//
	//app.Get("/", func(ctx *iris.Context) {
	//	ctx.MustRender("client.html", mp.ClientPage{Title: "Client Page", Host: ctx.Host()})
	//})

	app.Post("/msg", func(ctx *iris.Context) {
		logger.Info(ctx.FormValue("msg"))
	})

	ws.OnConnection(func(c websocket.Connection) {
		fmt.Println("ws: new connection, ID =", c.ID())
		//page := &ClientPage{Title: "title", Host: "host"}
		//bts := make([]byte, 0, page.Msgsize())
		//bts, err := page.MarshalMsg(bts[0:0])
		//if err != nil {
		//	fmt.Println(err)
		//}
		//c.EmitMessage(bts)

		c.OnMessage(func(data []byte) {
			logger.Info(string(data))
		})

		c.OnDisconnect(func() {
			fmt.Printf("\nConnection with ID: %s has been disconnected!", c.ID())
		})
	})

	app.Listen(":8080")
}
