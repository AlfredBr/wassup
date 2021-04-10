package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/pkg/math"
)

// https://github.com/gofiber/fiber
// https://yalantis.com/blog/how-to-build-websockets-in-go/
// https://yalantis.com/blog/golang-vs-nodejs-comparison/

// https://blog.golang.org/json
// https://github.com/gorilla/websocket/blob/master/examples/chat/main.go
// https://github.com/googollee/go-socket.io
// https://scene-si.org/2017/09/27/things-to-know-about-http-in-go/
// https://dev.to/bcanseco/request-body-encoding-json-x-www-form-urlencoded-ad9

const maxMsgs = 8
const defaultPortNum = 3000
const debugMode = true

var userSymbols map[string]string
var symbolIndex = 0
var userMessages []UserMessage

type UserMessage struct {
	UserId  string `json:"userId"`
	Message string `json:"message"`
	Symbol  string `json:"symbol"`
}

func debug(val string) {
	if debugMode {
		fmt.Printf("%s", val)
	}
}

func getUserId(ctx *fiber.Ctx) string {
	var userMessage UserMessage
	ctx.BodyParser(&userMessage)
	userId := ctx.Cookies("userId")
	if len(userId) > 0 {
		return userId
	}
	userId = uuid.New().String()
	setCookie(ctx, "userId", userId)
	return userId
}
func getUserSymbol(userId string) string {
	symbols := []string{"🔴", "🟡", "🟢", "🟣", "🔵", "🟠", "🟤", "⚪️"}
	symbol := userSymbols[userId]
	if symbol == "" {
		symbol = symbols[symbolIndex%len(symbols)]
		userSymbols[userId] = symbol
		symbolIndex++
	}
	return symbol
}
func getUserMessage(ctx *fiber.Ctx) (UserMessage, error) {
	var userMessage UserMessage
	err := ctx.BodyParser(&userMessage)
	if err != nil {
		errMsg := "error cannot parse"
		ctx.Status(fiber.StatusBadRequest).SendString(errMsg)
		return userMessage, errors.New(errMsg) // TODO: what is correct here?
	}
	if userMessage.Message == "" {
		return userMessage, nil // errors.New("no message")
	}
	return userMessage, nil
}
func setCookie(ctx *fiber.Ctx, name string, value string) {
	cookie := new(fiber.Cookie)
	cookie.Name = name
	cookie.Value = value
	cookie.Expires = time.Now().Add(24 * time.Hour)
	ctx.Cookie(cookie)
}
func cooldownTest(ctx *fiber.Ctx) bool {
	now := time.Now().Unix()
	if cookie := ctx.Cookies("userDate"); cookie != "" {
		if then, err := strconv.ParseInt(cookie, 10, 64); err == nil {
			cooldown := (now - then) >= 1
			//debug(fmt.Sprintf("now= %d\nthen=%d\n", now, then))
			return cooldown
		}
	}
	return false
}
func cooldownError(ctx *fiber.Ctx) {
	const errMsg = "403 forbidden: cooldown failed"
	log.Println(errMsg)
	ctx.Status(fiber.StatusForbidden).SendString(errMsg)
}
func uniquenessTest(ctx *fiber.Ctx, userMessage UserMessage) bool {
	currentMessage := userMessage.Message
	previousMessage := ctx.Cookies("userMessage")
	//debug(fmt.Sprintf("prev='%s'\ncurr='%s'\n", previousMessage, currentMessage))
	unique := previousMessage != currentMessage
	//debug(fmt.Sprintf("unique=%v\n", unique))
	return unique
}
func uniquenessError(ctx *fiber.Ctx, userMessage UserMessage) {
	currentMessage := userMessage.Message
	previousMessage := ctx.Cookies("userMessage")
	//debug(fmt.Sprintf("currentMessage= '%v'\npreviousMessage='%v'\n", currentMessage, previousMessage))
	if previousMessage != "" {
		errMsg := fmt.Sprintf("403 forbidden: message not unique because '%s' = '%s'", previousMessage, currentMessage)
		log.Println(errMsg)
		ctx.Status(fiber.StatusForbidden).SendString(errMsg)
	}
}
func nullHandler(ctx *fiber.Ctx) error {
	ctx.Status(fiber.StatusOK)
	return nil
}
func userMessageHandler(ctx *fiber.Ctx) error {
	if ctx.Method() != "POST" {
		ctx.Status(fiber.StatusForbidden)
		return errors.New("403 not suported")
	}
	if ctx.Path() != "/UserMessage" {
		ctx.Status(fiber.StatusNotFound)
		return errors.New("404 not found")
	}

	userMessage, err := getUserMessage(ctx)
	if err != nil {
		return err
	}
	if len(userMessage.Message) > 0 {
		userDate := strconv.FormatInt(time.Now().Unix(), 10)
		setCookie(ctx, "userDate", userDate)
		if strings.HasPrefix(userMessage.Message, "/n") {
			// todo: custom symbol command
			return nil
		}
		if strings.HasPrefix(userMessage.Message, "/reset") {
			// todo: reset command
			return nil
		}
		if strings.HasPrefix(userMessage.Message, "/info") {
			// todo: info command
			return nil
		}
		if strings.HasPrefix(userMessage.Message, "/") {
			// unknown command
			return nil
		}
		if !cooldownTest(ctx) {
			cooldownError(ctx)
			return nil
		}
		if !uniquenessTest(ctx, userMessage) {
			uniquenessError(ctx, userMessage)
			return nil
		}
		userId := getUserId(ctx)
		userSymbol := getUserSymbol(userId)
		setCookie(ctx, "userMessage", userMessage.Message)
		userMessageOut := UserMessage{userId, userMessage.Message, userSymbol}
		fmt.Printf("/UserMessage: %+v\n", userMessageOut)
		userMessages = append(userMessages, userMessageOut)
		// todo: send broadcast message via websocket
		//writer, _ := tws.NextWriter(websocket.TextMessage)
		//writer.Write([]byte("ABC"))
	}

	userMessages = userMessages[math.Max(0, len(userMessages)-maxMsgs):]
	ctx.Status(fiber.StatusOK).JSON(userMessages)

	return nil
}
func getPort() int {
	portEnv := os.Getenv("PORT")
	portNum, err := strconv.ParseInt(portEnv, 10, 32)
	if err != nil || portNum == 0 {
		portNum = defaultPortNum
	}
	return int(portNum)
}

var tws *websocket.Conn

func main() {
	log.Println("--GoLang/GoFiber--")

	userSymbols = make(map[string]string)

	app := fiber.New()

	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Use("/socket.io/", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Use(logger.New())
	app.Static("/", ".")
	app.Post("/UserMessage", userMessageHandler)
	app.Get("/socket.io/", nullHandler)
	app.Get("/ws", websocket.New(func(ws *websocket.Conn) {
		tws = ws
		log.Println(ws.Locals("allowed")) // true
	}))

	app.Get("/zzws", websocket.New(func(c *websocket.Conn) {
		// c.Locals is added to the *websocket.Conn
		log.Println(c.Locals("allowed"))  // true
		log.Println(c.Params("id"))       // 123
		log.Println(c.Query("v"))         // 1.0
		log.Println(c.Cookies("session")) // ""

		// websocket.Conn bindings
		// https://pkg.go.dev/github.com/fasthttp/websocket?tab=doc#pkg-index
		var (
			mt  int
			msg []byte
			err error
		)
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
				break
			}
			log.Printf("recv: %s", msg)

			if err = c.WriteMessage(mt, msg); err != nil {
				log.Println("write:", err)
				break
			}
		}
	}))

	addr := fmt.Sprintf(":%v", getPort())
	url := fmt.Sprintf("http://localhost%s", addr)
	log.Printf("listening: %s\n", url)
	log.Fatal(app.Listen(addr))
}
