# What's Daze?

Daze is a tool to help you link to the **Internet**.

\[English\] \[[中文](./README_CN.md)\]

# Usage

Compile or [Download](https://github.com/mohanson/daze/releases) daze:

```sh
$ git clone https://github.com/mohanson/daze
$ cd daze
$ go run cmd/make/main.go develop
```

Build results will be saved in directory `bin/develop`. You can just keep this directory, all other files are not required.

Daze is dead simple to use:

```sh
# server port
# you need a machine that can access the Internet, and enter the following command:
$ daze server -l 0.0.0.0:1081

# client port
# use the following command to link your server(replace $SERVER with your server ip):
$ daze client -s $SERVER:1081 -l 127.0.0.1:1080 -dns 114.114.114.114:53
# now, you are free to visit Internet
$ daze cmd curl https://google.com
```

# For browser, Firefox, Chrome or Edge e.g.

Daze forces any TCP/UDP connection to follow through proxy like SOCKS4, SOCKS5 or HTTP(S) proxy. It can be simply used in browser, take Firefox as an example: Open `Connection Settings` -> `Manual proxy configuration` -> `SOCKSv5 Host=127.0.0.1` and `Port=1080`.

# For android

Daze can work well on **Windows**, **Linux** and **macOS**. In additional, it can also work on **Android**, just it will be a bit complicated.

1. Download [SDK Platform Tools](https://developer.android.com/studio/releases/platform-tools) and make sure you can use `adb` normally.
2. Connect your phone to your computer with USB. Use `adb devices` to list devices.
2. Cross compile daze for android: `GOOS=linux GOARCH=arm go build -o daze github.com/mohanson/daze/cmd/daze`
4. Push binary and open shell: `adb push daze /data/local/tmp/daze`, `adb shell`
5. Open daze client: `cd /data/local/tmp`, `chmod +x daze`, `daze client -s $SERVER:1081 -l 127.0.0.1:1080 -dns 114.114.114.114:53`. Attention, you may wish use `setsid` to run daze in a new session.
6. Set the proxy for phone: WLAN -> Settings -> Proxy -> Fill in `127.0.0.1:1080`
7. Now, you are free to visit Internet.

# Use custom rules

daze use a RULE file to custom your own rules(optional). RULE has the highest priority in filters, so that you should carefully maintain it. This is a RULE document located at "./rule.ls", use `daze client -r ./rule.ls` to apply it.

```
L a.com
R b.com
B c.com
```
- L(ocale) means using local network
- R(emote) means using proxy
- B(anned) means block it

Glob is supported, such as `R *.google.com`.

# More

You can find all the information here by using `daze server -h` and `daze client -h`. The cli provides

- Encrypted data connection
- Confuse
- Specify DNS

Have fun.
