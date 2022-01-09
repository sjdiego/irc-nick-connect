package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/textproto"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Hostname string
	Port     int
	Nicklist []string
	Channels []string
}

var (
	conn   net.Conn
	config Config
)

func main() {
	config = getConfig()
	conn = connect(config)
	catchSigterm()

	send("USER _ 0 * :Go")

	go loopNicks()
	go joinChannels()

	tp := textproto.NewReader(bufio.NewReader(conn))
	for {
		status, err := tp.ReadLine()
		if err != nil {
			panic(err)
		}
		read(status)
	}
}

func getConfig() Config {
	configFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		panic(err)
	}
	var config Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		panic(err)
	}
	log.Printf("Config OK")
	return config
}

func catchSigterm() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		disconnect()
	}()
}

func connect(cfg Config) net.Conn {
	log.Printf("Trying connection to %s ...", cfg.Hostname)
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Hostname, cfg.Port))
	if err != nil {
		panic(err)
	}
	log.Printf("Connected successfully to %s", cfg.Hostname)
	return conn
}

func disconnect() {
	send("QUIT :Finished")
	conn.Close()
	os.Exit(0)
}

func send(message string) {
	log.Printf("-> %s", message)
	fmt.Fprintf(conn, "%s\r\n", message)
}

func read(message string) {
	raw := getRaw(message)
	if raw > 0 {
		parseRaw(raw, message)
	}

	if strings.HasPrefix(message, "PING") {
		replyPing(message)
	}
}

func getRaw(message string) int {
	// :stirling.chathispano.com 433 * :El nick n está en uso.
	if !strings.HasPrefix(message, ":") {
		return 0
	}

	parts := strings.Split(message, " ")

	raw, err := strconv.Atoi(parts[1])

	if err != nil {
		return 0
	}

	return raw
}

func parseRaw(raw int, message string) {
	switch raw {
	case 433, // Nick in use
		451, // Not registered
		464: // Invalid password
		setNick(randomString(8))
		return
	case 372, 375, 376: // MOTD
		return
	default:
		log.Printf("<- %s", message)
	}
}

func replyPing(message string) {
	pongCode := strings.Split(message, ":")
	send(fmt.Sprintf("PONG :%s", pongCode[1]))
}

func setNick(nick string) {
	nickName := strings.Split(nick, ":")
	log.Printf("Changing nick to %s", nickName[0])
	send(fmt.Sprintf("NICK %s", nick))
}

func loopNicks() {
	log.Println("Starting nicks...")

	for _, nick := range config.Nicklist {
		setNick(nick)
		time.Sleep(30 * time.Second)
	}

	log.Println("End of nicks")

	disconnect()
}

func joinChannels() {
	if len(config.Channels) > 0 {
		time.Sleep(5 * time.Second)
		for _, channel := range config.Channels {
			log.Printf("Joining channel #%s ...", channel)
			send(fmt.Sprintf("JOIN #%s", channel))
		}
	}
}

func randomString(n int) string {
	var letters = []rune("aeiouAEIOU")

	s := make([]rune, n)

	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}

	return string(s)
}
