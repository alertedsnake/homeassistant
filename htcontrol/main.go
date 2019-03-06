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
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"gopkg.in/urfave/cli.v1"
)

const DEFAULT_BROKER = "127.0.0.1:1883"
const DEFAULT_TOPIC_CONTROL = "ht/control"
const DEFAULT_TOPIC_STATUS = "ht/status"
const DEVTYPE_IR = "ir"
const DEVTYPE_CEC = "cec"

// devices -> types
var devices = map[string]string{
	"sonytv":  DEVTYPE_IR,
	"marantz": DEVTYPE_IR,
	"lgtv":    DEVTYPE_CEC,
}

func runIrsend(device string, action string) (err error) {
	cmd := exec.Command("irsend", "send_once", device, action)
	err = cmd.Run()
	if err != nil {
		err = errors.Wrapf(err, "Failed to run command %v", cmd.Args)
	}
	return
}

func runCEC(device string, action string) error {

	// default is to just ping the adapter
	input := "ping"

	switch action {
	case "poweron":
		input = "on 0"
	case "poweroff":
		input = "standby 0"
	}

	cmd := exec.Command("cec-client-4.0.2", "-d", "1", "RPI")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrapf(err, "Failed to create stdin pipe")
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	err = cmd.Run()
	if err != nil {
		err = errors.Wrapf(err, "Failed to run command %v", cmd.Args)
	}

	return nil
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
		devtype, ok := devices[device]
		if !ok {
			log.Errorf("Invalid device '%s', device")
		}

		switch devtype {
		case DEVTYPE_IR:
			if err := runIrsend(device, action); err != nil {
				log.Errorf("Error sending IR message: %v", err)
				return
			}
		case DEVTYPE_CEC:
			if err := runCEC(device, action); err != nil {
				log.Errorf("Error sending CEC command: %v", err)
				return
			}
		}

		// if we just turned the thing on, respond back with a status
		// message saying "on" or "off"
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

	handler := makeMessageHandler(c.GlobalString("status-topic"))

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", string(c.GlobalString("broker"))))
	opts.SetClientID(fmt.Sprintf("%s %s", c.App.Name, c.App.Version))
	opts.SetDefaultPublishHandler(handler)
	if c.GlobalString("username") != "" {
		opts.SetUsername(c.GlobalString("username"))
	}
	if c.GlobalString("password") != "" {
		opts.SetPassword(c.GlobalString("password"))
	}

	opts.OnConnect = func(channel mqtt.Client) {
		// subscribe to our topic
		topic := fmt.Sprintf("%s/#", c.GlobalString("control-topic"))
		if token := channel.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
			log.Fatalf("Subscribe error: %v", token.Error())
		}
		log.Infof("Subscribed to %s", c.GlobalString("control-topic"))
	}
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("Connection error: %v", token.Error())
	}
	log.Infof("Connected to server %v", c.GlobalString("broker"))

	<-channel
	return nil
}

// run in simple send mode
func runSend(c *cli.Context) error {

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", c.GlobalString("broker")))
	opts.SetClientID(fmt.Sprintf("%s %s", c.App.Name, c.App.Version))
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
	topic := fmt.Sprintf("%s/%s", c.GlobalString("status-topic"), device)
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
		},
		{
			Name:   "serve",
			Action: runServer,
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "broker, b",
			Usage:  "MQTT broker",
			EnvVar: "MQTT_broker",
			Value:  DEFAULT_BROKER,
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

		cli.StringFlag{
			Name:  "status-topic",
			Usage: "MQTT topic for status messages",
			Value: DEFAULT_TOPIC_STATUS,
		},
		cli.StringFlag{
			Name:  "control-topic",
			Usage: "MQTT topic for control messages",
			Value: DEFAULT_TOPIC_CONTROL,
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

		log.Debugf("Broker: %s", c.GlobalString("broker"))
		log.Debugf("Username: %s", c.GlobalString("username"))
		log.Debugf("Topic: Control: %s", c.GlobalString("control-topic"))
		log.Debugf("Topic: Status: %s", c.GlobalString("status-topic"))
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
