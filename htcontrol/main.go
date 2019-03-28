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

	"github.com/alertedsnake/homeassistant/htcontrol/config"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"gopkg.in/urfave/cli.v1"
)

const (
	DEFAULT_BROKER        = "127.0.0.1:1883"
	DEFAULT_TOPIC_CONTROL = "ht/control"
	DEFAULT_TOPIC_STATUS  = "ht/status"
)

var (
	VERSION = "0.1.0-alpha1"
	CFGARGS = []string{"broker", "username", "password", "control-topic", "status-topic"}
	cfg     *config.Config
)

func runIrsend(device string, action string) (err error) {
	cmd := exec.Command("irsend", "send_once", device, action)
	err = cmd.Run()
	if err != nil {
		err = errors.Wrapf(err, "Failed to run command %v", cmd.Args)
	}
	return
}

// make a message handler
func makeMessageHandler(status_topic string) mqtt.MessageHandler {

	var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		action := string(msg.Payload())

		log.Infof("%s: %s", msg.Topic(), msg.Payload())

		// Split up the topic, should be ht/control/device
		parts := strings.Split(msg.Topic(), "/")
		device := parts[2]

		// handle this by sending the action to the remote
		if err := runIrsend(device, action); err != nil {
			log.Errorf("Error sending IR message: %v", err)
			return
		}

		// respond back with a status // message saying "on" or "off"
		if action == "poweron" || action == "poweroff" {
			topic := fmt.Sprintf("%s/%s", status_topic, device)
			token := client.Publish(topic, 0, false, strings.Replace(action, "power", "", 1))
			token.Wait()
		}
	}
	return f
}

// run in server mode
func runServer(c *cli.Context) error {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)

	handler := makeMessageHandler(cfg.GetString("status-topic"))

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", string(cfg.GetString("broker"))))
	opts.SetClientID(fmt.Sprintf("%s %s", c.App.Name, c.App.Version))
	opts.SetDefaultPublishHandler(handler)
	if cfg.GetString("username") != "" {
		opts.SetUsername(cfg.GetString("username"))
	}
	if cfg.GetString("password") != "" {
		opts.SetPassword(cfg.GetString("password"))
	}

	opts.OnConnect = func(channel mqtt.Client) {
		// subscribe to our topic
		topic := fmt.Sprintf("%s/#", cfg.GetString("control-topic"))
		if token := channel.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
			log.Fatalf("MQTT Subscribe error: %v", token.Error())
		}
		log.Infof("Subscribed to %s", cfg.GetString("control-topic"))
	}
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("MQTT Connection error: %v", token.Error())
	}
	log.Infof("Connected to server %v", cfg.GetString("broker"))

	<-channel
	return nil
}

// run in simple send mode
func runSend(c *cli.Context) error {

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", cfg.GetString("broker")))
	opts.SetClientID(fmt.Sprintf("%s %s", c.App.Name, c.App.Version))
	if cfg.GetString("username") != "" {
		opts.SetUsername(cfg.GetString("username"))
	}
	if cfg.GetString("password") != "" {
		opts.SetPassword(cfg.GetString("password"))
	}
	opts.SetKeepAlive(2 * time.Second)

	// the args
	device := string(c.Args().Get(0))
	status := string(c.Args().Get(1))

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("MQTT Connection error: %v", token.Error())
	}

	// send to the 'status' topic
	topic := fmt.Sprintf("%s/%s", cfg.GetString("status-topic"), device)
	log.Debugf("%s -> %s", topic, status)
	if token := client.Publish(topic, 0, false, status); token.Wait() && token.Error() != nil {
		return fmt.Errorf("MQTT Publish error: %v", token.Error())
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
		},
		{
			Name:   "serve",
			Action: runServer,
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Config file",
		},
		cli.StringFlag{
			Name:  "broker, b",
			Usage: "MQTT broker",
		},
		cli.StringFlag{
			Name:  "username, u",
			Usage: "MQTT username",
		},
		cli.StringFlag{
			Name:  "password, p",
			Usage: "MQTT password",
		},

		cli.StringFlag{
			Name:  "status-topic",
			Usage: "MQTT topic for status messages",
		},
		cli.StringFlag{
			Name:  "control-topic",
			Usage: "MQTT topic for control messages",
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
		cfg = config.New()
		cfg.Set("status-topic", DEFAULT_TOPIC_STATUS)
		cfg.Set("control-topic", DEFAULT_TOPIC_CONTROL)
		cfg.Set("broker", DEFAULT_BROKER)

		// read config file
		if c.GlobalString("config") != "" {
			var err error
			cfg, err = config.Load(c.GlobalString("config"))
			if err != nil {
				return err
			}
		}

		// allow command-line overrides
		for _, val := range CFGARGS {
			if c.GlobalString(val) != "" {
				cfg.Set(val, c.GlobalString(val))
			}
		}

		log.Debugf("Broker: %s", cfg.GetString("broker"))
		log.Debugf("Username: %s", cfg.GetString("username"))
		log.Debugf("Topic: Control: %s", cfg.GetString("control-topic"))
		log.Debugf("Topic: Status: %s", cfg.GetString("status-topic"))
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
	if err := runCli(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
