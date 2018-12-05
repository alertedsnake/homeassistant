/*
Really basic "home theatre control" client.

The idea is we watch a topic for specific messages, and then run lirc to
send IR signals as if we pushed a remote control button.

Or, act as a simple client to send status messages back if certain IR
signals have been received.  Because you want to know when someone turned
the device on with a remote rather than via the HT system.
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"gopkg.in/urfave/cli.v1"
)

const DEFAULT_BROKER = "127.0.0.1:1883"
const DEFAULT_TOPIC_CONTROL = "ht/control/#"
const DEFAULT_TOPIC_STATUS = "ht/status"

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Infof("%s: %s", msg.Topic(), msg.Payload())
	action := string(msg.Payload())

	// verify payload - ignore anything but these
	if action != "on" && action != "off" {
		return
	}

	// Split up the topic, should be ht/control/device
	parts := strings.Split(msg.Topic(), "/")
	device := parts[2]

	// handle this by sending 'poweron' or 'poweroff'
	fullaction := fmt.Sprintf("power%s", action)
	cmd := exec.Command("irsend", "send_once", device, fullaction)
	err := cmd.Run()
	if err != nil {
		log.Errorf("Failed to run command %v: %v", cmd.Args, err)
		return
	}

	// respond back with a status message
	topic := fmt.Sprintf("%s/%s", c.getString("topic"), device)
	token := client.Publish(topic, 0, false, msg.Payload())
	token.Wait()
}

// run in server mode
func runServer(c *cli.Context) error {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)

	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s", c.GlobalString("broker")))
	opts.SetClientID("htcontrol 0.1.0")
	opts.SetDefaultPublishHandler(f)
	if c.GlobalString("username") != "" {
		opts.SetUsername(c.GlobalString("username"))
	}
	if c.GlobalString("password") != "" {
		opts.SetPassword(c.GlobalString("password"))
	}

	opts.OnConnect = func(channel mqtt.Client) {
		// subscribe to our topic
		if token := channel.Subscribe(c.getString("topic"), 0, f); token.Wait() && token.Error() != nil {
			log.Error("%v", token.Error())
		}
	}
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("%v", token.Error())
	}
	log.Infof("Connected to server %v", c.GlobalString("broker"))

	<-channel
	return nil
}

// run in simple send mode
func runSend(c *cli.Context) error {

	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s", c.getString("broker")))
	opts.SetClientID("htcontrol 0.1.0")
	opts.SetDefaultPublishHandler(f)
	if c.GlobalString("username") != "" {
		opts.SetUsername(c.GlobalString("username"))
	}
	if c.GlobalString("password") != "" {
		opts.SetPassword(c.GlobalString("password"))
	}
	opts.SetKeepAlive(2 * time.Second)

	// the args
	device := string(c.Args().Get(0))
	status := string(c.Args().Get(1))

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("Connection error: %v", token.Error())
	}

	// send to the 'status' topic
	topic := fmt.Sprintf("%s/%s", c.String("topic"), device)
	log.Debugf("%s -> %s", topic, status)
	if token := client.Publish(topic, 0, false, status); token.Wait() && token.Error() != nil {
		return fmt.Errorf("Publish error: %v", token.Error())
	}
	client.Disconnect(250)
	return nil
}

// run the CLI
func runCli() error {
	app := cli.NewApp()
	app.Version = VERSION
	app.Compiled = time.Now()
	app.Name = "HTControl"
	app.Commands = []cli.Command{
		{
			Name:   "send",
			Action: runSend,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:    "topic",
					Usage:   "MQTT topic",
					Default: DEFAULT_TOPIC_STATUS,
				},
			},
		},
		{
			Name:   "serve",
			Action: runServer,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:    "topic",
					Usage:   "MQTT topic",
					Default: DEFAULT_TOPIC_CONTROL,
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:    "broker, b",
			Usage:   "MQTT broker",
			EnvVar:  "MQTT_broker",
			Default: DEFAULT_BROKER,
		},
		cli.StringFlag{
			Name:   "username, u",
			Usage:  "MQTT username",
			EnvVar: "MQTT_username",
		},
		cli.StringFlag{
			Name:   "password, p",
			Usage:  "MQTT password",
			EnvVar: "MQTT_password",
		},

		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable verbose debugging",
		},
	}

	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	}

	return app.Run(os.Args)
}

func main() {
	// I like pretty logs, so...
	formatter := new(prefixed.TextFormatter)
	formatter.FullTimestamp = true
	log.SetFormatter(formatter)
	log.SetLevel(log.InfoLevel)

	// run the program
	err := runCli()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
