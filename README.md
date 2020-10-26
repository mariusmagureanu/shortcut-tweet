## Tweet streamer POC                    
This is a lite poc project with its sole purpose to fetch tweets and distribute them further to any interested clients.
The project is split into two components. One handling the tweet feed and distribution and another one playing as a simple client interested in the given tweets.

## Prereqs:

 1. Go
 2. Docker
 3. [Nats](https://nats.io)
 
 ## Build
 Run the following from the top-level of the source tree:
 

    $ make
    
 if on MacOS:
 
    $ make osxbuild
  
 This will create a ``bin`` folder under both the ``streamer`` and ``client`` folders containing the appropriate binaries.

## Usage

#### Streamer
 
| Option        | About                                             | Default       | Required       |
| ------------- |:--------------------------------------------------|:-------------:|:--------------:|
|*accessToken*         | Twitter API access token  |  |
|*accessTokenSecret*    | Twitter API access token secret  |  |
|*consumerKey*         | Twitter API consumer key  |      |
|*consumerKeySecret*          | Twitter API consumer key secret| |  |
|*natsServer*            | Default nats server| nats://localhost:4222 |
|*tweetCount*            | Count of tweets to fetch from the Timeline| 2 |
|*subj*            | Subject to use when publishing tweets| tweet |
|*V*            | Show version and exit|  |	 

#### Client
 
| Option        | About                                             | Default       | Required       |
| ------------- |:--------------------------------------------------|:-------------:|:--------------:|
|*natsServer*            | Default nats server| nats://localhost:4222 |
|*subj*            | Subject to use when publishing tweets| tweet |
|*V*            | Show version and exit|  |	

#### Run
Before starting either the ``streamer`` or the ``client`` make sure you have a running nats server.
If not, run the following:

    $ GO111MODULE=on go get github.com/nats-io/nats-server/v2
    $ nats-server -D

The ``streamer`` will look for Twitter api secrets either in the environment variables or in its cli args.  If none found it will generate phony tweets, these are very poor in their content but they serve the purpose.

Get into the streamer folder and run the following:

    $ ./bin/streamer

Get into the client folder and run the following:

    $ ./bin/client

If everything went as expected, the streamer will keep on firing batches of tweets onto nats which will further handle their distribution to the interested clients.

By default, the streamer will push out phony tweets. For fetching tweets from an actual Tweeter acount you will need a developer account along with its associated keys and tokens. If you have such secrets at hand fill them into the ``secrets.env`` file and source it before running the streamer:

    $ source secrets.env

#### Run with Docker

Start the whole poc with Docker by running the following:

    $ source global.env
    $ make docker

This will spawn a streamer container, a nats container and 5 client containers.
