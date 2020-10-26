package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/nats-io/go-nats"
)

const (
	envConsumerKey       = "TWEETER_API_CONSUMER_KEY"
	envConsumerKeySecret = "TWEETER_API_CONSUMER_KEY_SECRET"
	envAccessToken       = "TWEETER_API_ACCESS_TOKEN"
	envAccessTokenSecret = "TWEETER_API_ACCESS_TOKEN_SECRET"

	envNatsServer = "NATS_SERVER"
)

var (
	commandLine       = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	consumerKey       = commandLine.String("consumerKey", "", "Tweeter api consumer key.")
	consumerKeySecret = commandLine.String("consumerKeySecret", "", "Tweeter api consumer key secret.")
	accessToken       = commandLine.String("accessToken", "", "Tweeter api access token.")
	accessTokenSecret = commandLine.String("accessTokenSecret", "", "Tweeter api access token secret.")
	natsServer        = commandLine.String("natsServer", nats.DefaultURL, "Default nats server.")
	tweetSubject      = commandLine.String("subj", "tweet", "Default subject name to publish towards nats.")
	versionFlag       = commandLine.Bool("V", false, "Show version and exit")
	tweetCount        = commandLine.Int("tweetCount", 2, "Set how many tweets should be retrieved.")

	version  = "N/A"
	revision = "N/A"
)

// secrets is a helper type which contains
// secrets and tokens used to authenticate
// against the twitter api.
type secrets struct {
	consumerKey       string
	consumerKeySecret string
	accessToken       string
	accessTokenSecret string
}

// getPhonyTweets creates twitter.Tweet objects and
// populates a couple of their properties after which
// they're being sent onto a tweet chanel every 2 seconds.
// * tweetChan - channel to send Tweets onto.
func getPhonyTweets(tweetChan chan twitter.Tweet, wg *sync.WaitGroup) {

	ticker := time.NewTicker(2 * time.Second)

	user := twitter.User{}
	user.Name = "John Doe"

	wg.Done()

	for {
		select {
		case <-ticker.C:
			tc := rand.Intn(10)

			for i := 0; i < tc; i++ {
				tweet := twitter.Tweet{}
				tweet.User = &user

				tweet.ID = time.Now().Unix()
				tweet.CreatedAt = time.Now().Local().String()
				tweet.Text = fmt.Sprintf("%s - %d", "foo bar baz", i)

				tweetChan <- tweet
			}
		}
	}
}

// getTweeterSecrets looks for Tweeter api secrets
// and tokens and errors out if they're not found
// in either cli options or env vars.
func getTweeterSecrets() (secrets, error) {
	ts := secrets{}

	ts.consumerKey = os.Getenv(envConsumerKey)
	ts.consumerKeySecret = os.Getenv(envConsumerKeySecret)
	ts.accessToken = os.Getenv(envAccessToken)
	ts.accessTokenSecret = os.Getenv(envAccessTokenSecret)

	if *consumerKey != "" {
		ts.consumerKey = *consumerKey
	}

	if *consumerKeySecret != "" {
		ts.consumerKeySecret = *consumerKeySecret
	}

	if *accessToken != "" {
		ts.accessToken = *accessToken
	}

	if *accessTokenSecret != "" {
		ts.accessTokenSecret = *accessTokenSecret
	}

	if ts.consumerKey == "" || ts.consumerKeySecret == "" || ts.accessToken == "" || ts.accessTokenSecret == "" {
		return ts, errors.New("incomplete tweeter api secrets")
	}

	return ts, nil
}

// getTweeterClient returns a twitter.Client instance which
// handles the entire communication with the Tweeter api.
// * ts - secrets instance containing auth tokens against the Twitter api.
func getTweeterClient(ts secrets) *twitter.Client {
	config := oauth1.NewConfig(ts.consumerKey, ts.consumerKeySecret)
	token := oauth1.NewToken(ts.accessToken, ts.accessTokenSecret)

	httpClient := config.Client(oauth1.NoContext, token)
	return twitter.NewClient(httpClient)
}

// getTweets returns a slice of Tweet objects.
// * count   - fetch a specific amount of tweets.
// * sinceID - fetch tweets which occured after the given Id
//             in order to avoid returning dups.
func getTweets(tc *twitter.Client, count int, sinceID int64) ([]twitter.Tweet, error) {

	timeLineParams := twitter.HomeTimelineParams{Count: count, SinceID: sinceID}
	tweets, _, err := tc.Timelines.HomeTimeline(&timeLineParams)

	return tweets, err
}

// pollTweets fetches twitter.Tweet objects from Twitter at
// a given interval and sends them over onto a tweet channel.
// * tc           - twitter client to handle Twitter api communication.
// * pollInterval - interval to fetch tweets.
// * tweetChan    - channel to send tweets onto.
func pollTweets(tc *twitter.Client, pollInterval time.Duration, tweetChan chan twitter.Tweet, wg *sync.WaitGroup) {
	ticker := time.NewTicker(pollInterval)
	done := make(chan bool)

	var sinceID int64

	wg.Done()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			tweets, err := getTweets(tc, *tweetCount, sinceID)

			if err != nil {
				log.Fatalln(err)
			}

			if len(tweets) > 0 {
				sinceID = tweets[0].ID
			}

			for _, tweet := range tweets {
				tweetChan <- tweet
			}
		}
	}
}

// setupConnOptions creates a slice of Options
// used when connecting againsta Nats server.
// * opts - initial Options slice.
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

// connectToNats connects to a Nats server and returns
// a connection with a json encoder.
// * natsServer - nats server url. (e.g. http://localhost:4222)
// * opts       - connection options
func connectToNats(natsServer string, opts []nats.Option) (*nats.EncodedConn, error) {

	nc, err := nats.Connect(natsServer, opts...)

	if err != nil {
		return nil, err
	}

	nec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)

	if err != nil {
		return nil, err
	}

	return nec, err
}

// publishTweets reads the tweeter channel and sends out
// incoming tweets to Nats using the specified subject.
// * nec - json encoded Nats connection
// * tweetChan - channel of Tweet objects
// * subj - subject used to publish the tweets with, clients
//          intersted in these tweets will need to subscribe
//          to this subject.
func publishTweets(nec *nats.EncodedConn, tweetChan chan twitter.Tweet, subj string) {

	for {
		select {
		case tweet := <-tweetChan:
			log.Printf("Publishing from [%s]:\n  '%s'\n", tweet.User.Name, tweet.Text)
			nec.Publish(subj, tweet)
		}
	}
}

// getNatsServer looks for a nats server in the
// env vars, if none specified it will use the
// one set in the cli options or the default one.
func getNatsServer() string {

	ns := os.Getenv(envNatsServer)

	if ns != "" {
		return ns
	}

	return *natsServer
}

func main() {

	commandLine.Usage = func() {
		fmt.Fprint(os.Stdout, "Usage of the tweet streamer:\n")
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

	opts := []nats.Option{nats.Name("Tweeter demo publisher")}
	opts = setupConnOptions(opts)

	ns := getNatsServer()
	conn, err := connectToNats(ns, opts)

	if err != nil {
		log.Fatalln(err)
	}

	tweetChan := make(chan twitter.Tweet, *tweetCount)
	defer close(tweetChan)

	var (
		wg    sync.WaitGroup
		phony bool
	)

	wg.Add(1)

	ts, err := getTweeterSecrets()

	if err != nil {
		phony = true
		log.Println("Could not fetch API secrets, will continue with phony tweets.")
	}

	if phony {
		go getPhonyTweets(tweetChan, &wg)
	} else {
		tc := getTweeterClient(ts)
		go pollTweets(tc, 2*time.Second, tweetChan, &wg)
	}

	wg.Wait()

	publishTweets(conn, tweetChan, *tweetSubject)
}
