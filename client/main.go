package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/nats-io/go-nats"
)

const envNatsServer = "NATS_SERVER"

var (
	commandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	subj        = commandLine.String("subj", "tweet", "Default subject to subscribe to.")
	natsServer  = commandLine.String("natsServer", nats.DefaultURL, "Default nats server.")
	versionFlag = commandLine.Bool("V", false, "Show version and exit.")

	version  = "N/A"
	revision = "N/A"
)

// User is a phone Tweet.User type
type User struct {
	Name string
}

// Tweet is a phony Tweet type
type Tweet struct {
	User      User
	ID        int64
	CreatedAt string
	Text      string
}

func getNatsServer() string {

	ns := os.Getenv(envNatsServer)

	if ns != "" {
		return ns
	}

	return *natsServer
}

// printMsg displays to stdlog the summary
// of a received tweet.
// * m - incoming Nats message
func printMsg(m *nats.Msg) {
	if m.Data != nil {
		var tweet Tweet
		err := json.Unmarshal(m.Data, &tweet)

		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("Received from [%s]:\n  '%s'\n", tweet.User.Name, tweet.Text)
	}
}

// setupConnOptions populates a slice of Options
// which tell how to connect to a nats server.
// * opts - slice of initial Options
func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectHandler(func(nc *nats.Conn) {
		log.Printf("Disconnected: will attempt reconnects for %.0fm", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Printf("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Fatal("Exiting, no servers available")
	}))
	return opts
}

func main() {
	commandLine.Usage = func() {
		fmt.Fprint(os.Stdout, "Usage of the tweet subscriber:\n")
		commandLine.PrintDefaults()
		os.Exit(0)
	}

	if err := commandLine.Parse(os.Args[1:]); err != nil {
		log.Fatalln(err)
	}

	if *versionFlag {
		fmt.Println("Version:  " + version)
		fmt.Println("Revision: " + revision)
		os.Exit(0)
	}

	opts := []nats.Option{nats.Name("Tweet subscriber")}
	opts = setupConnOptions(opts)

	ns := getNatsServer()

	nc, err := nats.Connect(ns, opts...)
	if err != nil {
		log.Fatalln(err)
	}

	nc.Subscribe(*subj, func(msg *nats.Msg) {
		printMsg(msg)
	})

	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Listening on [%s]", *subj)

	runtime.Goexit()
}
