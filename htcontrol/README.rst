
htcontrol
=========

A really basic "home theatre control" client program for use with the MQTT_
component in `Home Assistant`_, which can turn on and off my TV and stereo
via a Iguanaworks_ IR device.

The main reason for this is that my RaspberryPi which runs Home Assistant is
not the same one which runs Kodi and has the IR controller.

So.  This progam can do two things:

Server
------
In this mode, we watch a MQTT topic for specific messages, and then run
``irsend`` to send IR signals as if we pushed a remote control button.

Sender
------
In this mode, the program will act as a simple client to publish status messages
via MQTT if certain IR signals have been received.  Because you want to know
when someone turned the device on with a remote rather than via Home Assistant.


Building
========

Have go.  Run ``make``.

Use
===

Home Assistant
--------------

Here's how I have my TV and stereo setup.

.. code-block:: yaml

    switch:
        - platform: mqtt
          name: Stereo
          command_topic: "ht/control/marantz"
          state_topic: "ht/status/marantz"
          state_on: "on"
          state_off: "off"
          payload_on: "poweron"
          payload_off: "poweroff"
          icon: mdi:radio

        - platform: mqtt
          name: TV
          command_topic: "ht/control/sonytv"
          state_topic: "ht/status/sonytv"
          state_on: "on"
          state_off: "off"
          payload_on: "poweron"
          payload_off: "poweroff"
          icon: mdi:television


irexec
------

My ``/etc/lirc/lircrc`` file, which is used with ``irexec``::

    begin
            prog = irexec
            remote = marantz
            button = poweron
            config = /usr/local/bin/htcontrol -b 10.0.3.16:1883 send marantz on
    end
    begin
            prog = irexec
            remote = marantz
            button = poweroff
            config = /usr/local/bin/htcontrol -b 10.0.3.16:1883 send marantz off
    end

    begin
            prog = irexec
            remote = sonytv
            button = poweron
            config = /usr/local/bin/htcontrol -b 10.0.3.16:1883 send sonytv on
    end
    begin
            prog = irexec
            remote = sonytv
            button = poweroff
            config = /usr/local/bin/htcontrol -b 10.0.3.16:1883 send sonytv off
    end

Obviously, provide user/password details, and use the names of your own remotes.

systemd
-------

I'm using this sytstemd unit file::

    [Unit]
    Description=Home Theatre Control
    After=network.target

    [Service]
    Environment=MQTT_broker="10.0.3.16:1883"
    ExecStart=/usr/local/bin/htcontrol serve

    [Install]
    WantedBy=multi-user.target

Of course, with user/password variables too.

.. _Home Assistant: https://www.home-assistant.io/
.. _Iguanaworks: https://www.iguanaworks.net/products/usb-ir-transceiver/
.. _MQTT: https://www.home-assistant.io/components/mqtt/
