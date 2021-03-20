package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/textproto"
	"os"
	"os/signal"
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

func main() {
	config := getConfig()
	conn := connect(config)
	catchSigterm(conn)

	go loopNicks(conn, config)
	go joinChannels(conn, config)

	tp := textproto.NewReader(bufio.NewReader(conn))
	for {
		status, err := tp.ReadLine()
		if err != nil {
			panic(err)
		}
		read(conn, status)
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

func catchSigterm(conn net.Conn) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		disconnect(conn)
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

func disconnect(conn net.Conn) {
	send(conn, "QUIT :Finished")
	conn.Close()
	os.Exit(0)
}

func send(conn net.Conn, message string) {
	log.Printf("-> %s", message)
	fmt.Fprintf(conn, "%s\r\n", message)
}

func read(conn net.Conn, message string) {
	log.Printf("<- %s", message)

	if strings.HasPrefix(message, "PING") {
		pongCode := strings.Split(message, ":")
		send(conn, fmt.Sprintf("PONG :%s", pongCode[1]))
	}
}

func loopNicks(conn net.Conn, cfg Config) {
	send(conn, "USER _ 0 * :Go")

	log.Println("Starting nicks...")

	for _, nick := range cfg.Nicklist {
		log.Printf("Changing nick to %s", nick)
		send(conn, fmt.Sprintf("NICK %s", nick))
		time.Sleep(30 * time.Second)
	}

	log.Println("End of nicks")

	disconnect(conn)
}

func joinChannels(conn net.Conn, cfg Config) {
	time.Sleep(5 * time.Second)
	if len(cfg.Channels) > 0 {
		for _, channel := range cfg.Channels {
			log.Printf("Joining channel #%s ...", channel)
			send(conn, fmt.Sprintf("JOIN #%s", channel))
		}
	}
}
