# What's Daze?

Daze 是一款帮助你连接至**互联网**的工具.

\[[English](./README.md)]\] \[中文\]

# 使用

使用 daze 该死的简单:

```sh
$ go get -u -v github.com/mohanson/daze/cmd/daze

# 服务端
# 你需要一台能正确连接互联网的机器, 并输入以下命令
$ daze server -l 0.0.0.0:51958

# 客户端
# 使用如下命令连接至你的服务端
$ daze client -s $SERVER:51958 -l 127.0.0.1:51959 -dns 114.114.114.114:53
# 现在, 你即可自由地访问互联网
$ daze cmd curl https://google.com
```

# 在浏览器中使用, Firefox, Chrome 或 Edge 等

Daze 通过代理技术, 如 SOCKS4, SOCKS5 和 HTTP(S) 代理转发任何本机的 TCP/UDP 流量. 在浏览器中使用 Daze 非常简单, 以 Firefox 为例: `选项` -> `网络代理` -> `手动配置代理` -> 填写 `SOCKS 主机=127.0.0.1` 和 `Port=51959`, 同时勾选 `SOCKS v5`. 注意的是, 在大部分情况下, 请同时勾选 `使用 SOCKS v5 时代理 DNS 查询`.

# 在 android 中使用

Daze 可以在 **Windows**, **Linux** 和 **macOS** 下正常工作. 另外, 它同样适用于 **Android**, 只是配置起来稍显复杂.

1. 下载 [SDK Platform Tools](https://developer.android.com/studio/releases/platform-tools) 并确保你能正常使用 `adb` 命令.
2. 使用 USB 连接你的手机和电脑. 使用 `adb devices` 可显示已连接的设备, 确保连接成功.
2. 交叉编译: `GOOS=linux GOARCH=arm go build -o daze github.com/mohanson/daze/cmd/daze`
4. 推送二进制文件至手机并进入 Shell: `adb push daze /data/local/tmp/daze`, `adb shell`
5. 启动 daze 客户端: `cd /data/local/tmp`, `chmod +x daze`, `daze client -s $SERVER:51958 -l 127.0.0.1:51959 -dns 114.114.114.114:53`. 注意的是, 你可能需要使用 `setsid` 命令将客户端程序托管至后台运行.
6. 设置代理: 连接任意 Wifi -> 设置 -> 代理 -> 填写 `127.0.0.1:51959`
7. 现在, 你即可自由地访问互联网

# 在 python 中使用

`daze cmd` 可以代理大部分应用比如 `curl` 或 `wget`(它们均使用了 `libcurl`). 你同样可以方便的代理自己的 python 代码. 你所需要做的一切就是安装 `pysocks` 并且使用 `daze cmd` 运行代码.

```sh
$ pip install pysocks requests
```

将下面的代码写入一个新的文件, 如 "google.py":

```py
import requests
r = requests.get('https://google.com')
print(r.status_code)
```

使用 `daze cmd python google.py` 而非 `python google.py`

```sh
$ daze cmd python google.py
# 200
```