module github.com/alertedsnake/homeassistant/htcontrol

require (
	github.com/alertedsnake/homeassistant/htcontrol/config v0.0.0
	github.com/eclipse/paho.mqtt.golang v1.1.1
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.0
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	golang.org/x/net v0.0.0-20190327214358-63eda1eb0650 // indirect
	gopkg.in/urfave/cli.v1 v1.20.0
)

replace github.com/alertedsnake/homeassistant/htcontrol/config => ./config
