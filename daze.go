package daze

import (
	"bufio"
	"bytes"
	"context"
	"crypto/cipher"
	"crypto/rc4"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mohanson/acdb"
	"github.com/mohanson/aget"
	"github.com/mohanson/ddir"
)

// Link copies from src to dst and dst to src until either EOF is reached.
func Link(a, b io.ReadWriteCloser) {
	go func() {
		io.Copy(b, a)
		a.Close()
		b.Close()
	}()
	io.Copy(a, b)
	b.Close()
	a.Close()
}

// ReadWriteCloser is the interface that groups the basic Read, Write and
// Close methods.
type ReadWriteCloser struct {
	io.Reader
	io.Writer
	io.Closer
}

// GravityReader wraps an io.Reader with RC4 crypto.
func GravityReader(r io.Reader, k []byte) io.Reader {
	cr, _ := rc4.NewCipher(k)
	return cipher.StreamReader{S: cr, R: r}
}

// GravityWriter wraps an io.Writer with RC4 crypto.
func GravityWriter(w io.Writer, k []byte) io.Writer {
	cw, _ := rc4.NewCipher(k)
	return cipher.StreamWriter{S: cw, W: w}
}

// Double gravity, double happiness.
func Gravity(conn io.ReadWriteCloser, k []byte) io.ReadWriteCloser {
	cr, _ := rc4.NewCipher(k)
	cw, _ := rc4.NewCipher(k)
	return &ReadWriteCloser{
		Reader: cipher.StreamReader{S: cr, R: conn},
		Writer: cipher.StreamWriter{S: cw, W: conn},
		Closer: conn,
	}
}

// Resolve modifies the net.DefaultResolver(which is the resolver used by the
// package-level Lookup functions and by Dialers without a specified Resolver).
//
// Examples:
//   Resolve("8.8.8.8:53")
//   Resolve("114.114.114.114:53")
func Resolve(addr string) {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "udp", addr)
		},
	}
}

// Dialer contains options for connecting to an address.
type Dialer interface {
	Dial(network string, address string) (io.ReadWriteCloser, error)
}

// IPv4ReservedIPNet returns reserved IPv4 addresses.
//
// Introduction:
//   See https://en.wikipedia.org/wiki/Reserved_IP_addresses
func IPv4ReservedIPNet() []*net.IPNet {
	r := []*net.IPNet{}
	for _, entry := range [][2]string{
		{"00000000", "FF000000"},
		{"0A000000", "FF000000"},
		{"7F000000", "FF000000"},
		{"A9FE0000", "FFFF0000"},
		{"AC100000", "FFF00000"},
		{"C0000000", "FFFFFFF8"},
		{"C00000AA", "FFFFFFFE"},
		{"C0000200", "FFFFFF00"},
		{"C0A80000", "FFFF0000"},
		{"C6120000", "FFFE0000"},
		{"C6336400", "FFFFFF00"},
		{"CB007100", "FFFFFF00"},
		{"F0000000", "F0000000"},
		{"FFFFFFFF", "FFFFFFFF"},
	} {
		i, _ := hex.DecodeString(entry[0])
		m, _ := hex.DecodeString(entry[1])
		r = append(r, &net.IPNet{IP: i, Mask: m})
	}
	return r
}

// IPv6ReservedIPNet returns reserved IPv6 addresses.
//
// Introduction:
//   See https://en.wikipedia.org/wiki/Reserved_IP_addresses
func IPv6ReservedIPNet() []*net.IPNet {
	r := []*net.IPNet{}
	for _, entry := range [][2]string{
		{"00000000000000000000000000000000", "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"},
		{"00000000000000000000000000000001", "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"},
		{"01000000000000000000000000000000", "FFFFFFFFFFFFFFFF0000000000000000"},
		{"0064FF9B000000000000000000000000", "FFFFFFFFFFFFFFFFFFFFFFFF00000000"},
		{"20010000000000000000000000000000", "FFFFFFFF000000000000000000000000"},
		{"20010010000000000000000000000000", "FFFFFFF0000000000000000000000000"},
		{"20010020000000000000000000000000", "FFFFFFF0000000000000000000000000"},
		{"20010DB8000000000000000000000000", "FFFFFFFF000000000000000000000000"},
		{"20020000000000000000000000000000", "FFFF0000000000000000000000000000"},
		{"FC000000000000000000000000000000", "FE000000000000000000000000000000"},
		{"FE800000000000000000000000000000", "FFC00000000000000000000000000000"},
		{"FF000000000000000000000000000000", "FF000000000000000000000000000000"},
	} {
		i, _ := hex.DecodeString(entry[0])
		m, _ := hex.DecodeString(entry[1])
		r = append(r, &net.IPNet{IP: i, Mask: m})
	}
	return r
}

// CNIPNet returns full ipv4/6 CIDR in CN.
func CNIPNet() []*net.IPNet {
	name := ddir.Join("delegated-apnic-latest")
	f, err := aget.Open(name)
	if err != nil {
		log.Panicln(err)
	}
	defer f.Close()
	r := []*net.IPNet{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "apnic|CN|ipv4"):
			seps := strings.Split(line, "|")
			sep4, err := strconv.Atoi(seps[4])
			if err != nil {
				log.Panicln(err)
			}
			mask := 32 - int(math.Log2(float64(sep4)))
			_, cidr, err := net.ParseCIDR(fmt.Sprintf("%s/%d", seps[3], mask))
			if err != nil {
				log.Panicln(err)
			}
			r = append(r, cidr)
		case strings.HasPrefix(line, "apnic|CN|ipv6"):
			seps := strings.Split(line, "|")
			sep4 := seps[4]
			_, cidr, err := net.ParseCIDR(fmt.Sprintf("%s/%s", seps[3], sep4))
			if err != nil {
				log.Panicln(err)
			}
			r = append(r, cidr)
		}
	}
	return r
}

func IPNetContains(l []*net.IPNet, ip net.IP) bool {
	for _, e := range l {
		if e.Contains(ip) {
			return true
		}
	}
	return false
}

// Locale is the main process of daze. In most cases, it is usually deployed
// as a daemon on a local machine.
type Locale struct {
	Listen string
	Dialer Dialer
}

// Serve traffic in HTTP Proxy/Tunnel format.
//
// Introduction:
//   See https://en.wikipedia.org/wiki/Proxy_server
//   See https://en.wikipedia.org/wiki/HTTP_tunnel
//
// Warning: The performance of HTTP Proxy is very poor, unless you have a good
// reason, please use ServeSocks4 or ServeSocks5 instead. Why the poor
// performance is that I did not implement http persistent connection(a
// well-known name is KeepAlive) because It will trigger some bugs on Firefox.
// Firefox always sends traffic from different sites to the one persistent
// connection. I have been debugging for a long time.
// Fuck.
func (l *Locale) ServeProxy(app io.ReadWriteCloser) error {
	reader := bufio.NewReader(app)

	for {
		if err := func() error {
			r, err := http.ReadRequest(reader)
			if err != nil {
				return err
			}

			var port string
			if r.URL.Port() == "" {
				port = "80"
			} else {
				port = r.URL.Port()
			}

			srv, err := l.Dialer.Dial("tcp", r.URL.Hostname()+":"+port)
			if err != nil {
				return err
			}
			defer srv.Close()
			servReader := bufio.NewReader(srv)

			if r.Method == "CONNECT" {
				log.Println("connect[tunnel]", r.URL.Hostname()+":"+port)
				_, err := app.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
				if err != nil {
					return err
				}
				Link(app, srv)
				return nil
			}

			log.Println("connect[hproxy]", r.URL.Hostname()+":"+port)
			if r.Method == "GET" && r.Header.Get("Upgrade") == "websocket" {
				if err := r.Write(srv); err != nil {
					return err
				}
				Link(app, srv)
				return nil
			}
			if err := r.Write(srv); err != nil {
				return err
			}
			resp, err := http.ReadResponse(servReader, r)
			if err != nil {
				return err
			}
			return resp.Write(app)
		}(); err != nil {
			break
		}
	}
	return nil
}

// Serve traffic in SOCKS4/SOCKS4a format.
//
// Introduction:
//   See https://en.wikipedia.org/wiki/SOCKS
//   See http://ftp.icm.edu.pl/packages/socks/socks4/SOCKS4.protocol
func (l *Locale) ServeSocks4(app io.ReadWriteCloser) error {
	var (
		reader    = bufio.NewReader(app)
		fCode     uint8
		fDstPort  = make([]byte, 2)
		fDstIP    = make([]byte, 4)
		fHostName []byte
		dstHost   string
		dstPort   uint16
		dst       string
		srv       io.ReadWriteCloser
		err       error
	)
	app = ReadWriteCloser{
		Reader: reader,
		Writer: app,
		Closer: app,
	}
	reader.Discard(1)
	fCode, _ = reader.ReadByte()
	io.ReadFull(reader, fDstPort)
	dstPort = binary.BigEndian.Uint16(fDstPort)
	io.ReadFull(reader, fDstIP)
	_, err = reader.ReadBytes(0x00)
	if err != nil {
		return err
	}
	if bytes.Equal(fDstIP[:3], []byte{0x00, 0x00, 0x00}) && fDstIP[3] != 0x00 {
		fHostName, err = reader.ReadBytes(0x00)
		if err != nil {
			return err
		}
		fHostName = fHostName[:len(fHostName)-1]
		dstHost = string(fHostName)
	} else {
		dstHost = net.IP(fDstIP).String()
	}
	dst = dstHost + ":" + strconv.Itoa(int(dstPort))
	log.Println("connect[socks4]", dst)
	switch fCode {
	case 0x01:
		srv, err = l.Dialer.Dial("tcp", dst)
		if err != nil {
			app.Write([]byte{0x00, 0x5b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			return err
		} else {
			defer srv.Close()
			app.Write([]byte{0x00, 0x5a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			Link(app, srv)
			return nil
		}
	case 0x02:
		log.Panicln("unreachable")
	}
	return nil
}

// Serve traffic in SOCKS5 format.
//
// Introduction:
//   See https://en.wikipedia.org/wiki/SOCKS
//   See https://tools.ietf.org/html/rfc1928
func (l *Locale) ServeSocks5(app io.ReadWriteCloser) error {
	var (
		reader   = bufio.NewReader(app)
		fN       uint8
		fCmd     uint8
		fAT      uint8
		fDstAddr []byte
		fDstPort = make([]byte, 2)
		dstHost  string
		dstPort  uint16
		dst      string
		err      error
	)
	app = ReadWriteCloser{
		Reader: reader,
		Writer: app,
		Closer: app,
	}
	reader.Discard(1)
	fN, _ = reader.ReadByte()
	reader.Discard(int(fN))
	app.Write([]byte{0x05, 0x00})
	reader.Discard(1)
	fCmd, _ = reader.ReadByte()
	reader.Discard(1)
	fAT, _ = reader.ReadByte()
	switch fAT {
	case 0x01:
		fDstAddr = make([]byte, 4)
		io.ReadFull(reader, fDstAddr)
		dstHost = net.IP(fDstAddr).String()
	case 0x03:
		fN, _ = reader.ReadByte()
		fDstAddr = make([]byte, int(fN))
		io.ReadFull(reader, fDstAddr)
		dstHost = string(fDstAddr)
	case 0x04:
		fDstAddr = make([]byte, 16)
		io.ReadFull(reader, fDstAddr)
		dstHost = net.IP(fDstAddr).String()
	}
	if _, err = io.ReadFull(app, fDstPort); err != nil {
		return err
	}
	dstPort = binary.BigEndian.Uint16(fDstPort)
	dst = dstHost + ":" + strconv.Itoa(int(dstPort))
	switch fCmd {
	case 0x01:
		return l.ServeSocks5TCP(app, dst)
	case 0x02:
		log.Panicln("unreachable")
	case 0x03:
		return l.ServeSocks5UDP(app)
	}
	return nil
}

// Socks5 TCP protocal.
func (l *Locale) ServeSocks5TCP(app io.ReadWriteCloser, dst string) error {
	log.Println("connect[socks5]", dst)
	srv, err := l.Dialer.Dial("tcp", dst)
	if err != nil {
		app.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return err
	} else {
		defer srv.Close()
		app.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		Link(app, srv)
		return nil
	}
}

// Socks5 UDP protocal.
func (l *Locale) ServeSocks5UDP(app io.ReadWriteCloser) error {
	defer app.Close()
	bndAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	bnd, _ := net.ListenUDP("udp", bndAddr)
	defer bnd.Close()
	bndPort := uint16(bnd.LocalAddr().(*net.UDPAddr).Port)
	r := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	binary.BigEndian.PutUint16(r[8:10], bndPort)
	app.Write(r)

	var (
		buf = make([]byte, 65536)
		srv = map[string]*net.UDPConn{}
	)

	go func() {
		io.Copy(ioutil.Discard, app)
		app.Close()
		bnd.Close()
		for _, v := range srv {
			v.Close()
		}
	}()

	for {
		n, appAddr, err := bnd.ReadFromUDP(buf)
		if err != nil {
			break
		}

		l := 0
		switch buf[3] {
		case 0x01:
			l = 10
		case 0x03:
			l = int(buf[4]) + 7
		case 0x04:
			l = 22
		}

		appHead := make([]byte, l)
		copy(appHead, buf[0:l])
		appData := make([]byte, n-l)
		copy(appData, buf[l:n])

		dstHost := ""
		dstPort := uint16(0)
		switch appHead[3] {
		case 0x01:
			dstHost = net.IP(appHead[4:8]).String()
			dstPort = binary.BigEndian.Uint16(appHead[8:10])
		case 0x03:
			l := appHead[4]
			dstHost = string(appHead[5 : 5+l])
			dstPort = binary.BigEndian.Uint16(appHead[5+l : 7+l])
		case 0x04:
			dstHost = net.IP(appHead[4:20]).String()
			dstPort = binary.BigEndian.Uint16(appHead[20:22])
		}
		dst := dstHost + ":" + strconv.Itoa(int(dstPort))

		ep, b := srv[dst]
		if !b {
			a, err := net.ResolveUDPAddr("udp", dst)
			if err != nil {
				break
			}
			log.Println("connect[socks5]", dst)
			c, err := net.DialUDP("udp", nil, a)
			if err != nil {
				break
			}
			srv[dst] = c
			ep = c

			go func(srv net.Conn, appHead []byte, appAddr *net.UDPAddr) {
				buf := make([]byte, 65536)
				copy(buf, appHead)
				l := len(appHead)
				for {
					n, _, err := c.ReadFromUDP(buf[l:])
					if err != nil {
						break
					}
					bnd.WriteToUDP(buf[:l+n], appAddr)
				}
			}(c, appHead, appAddr)
		}
		ep.Write(appData)
	}
	return nil
}

// We should be very clear about what it does. It judges the traffic type and
// processes it with a different handler(ServeProxy/ServeSocks4/ServeSocks5).
func (l *Locale) Serve(app io.ReadWriteCloser) error {
	var (
		buf = make([]byte, 1)
		err error
	)
	_, err = io.ReadFull(app, buf)
	if err != nil {
		return err
	}
	app = ReadWriteCloser{
		Reader: io.MultiReader(bytes.NewReader(buf), app),
		Writer: app,
		Closer: app,
	}
	if buf[0] == 0x05 {
		return l.ServeSocks5(app)
	}
	if buf[0] == 0x04 {
		return l.ServeSocks4(app)
	}
	return l.ServeProxy(app)
}

// Run.
func (l *Locale) Run() error {
	s, err := net.Listen("tcp", l.Listen)
	if err != nil {
		return err
	}
	defer s.Close()
	log.Println("listen and serve on", l.Listen)

	for {
		c, err := s.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go func() {
			defer c.Close()
			if err := l.Serve(c); err != nil {
				log.Println(err)
			}
		}()
	}
}

// NewLocale returns a Locale.
func NewLocale(listen string, dialer Dialer) *Locale {
	return &Locale{
		Listen: listen,
		Dialer: dialer,
	}
}

// Direct is the default dialer for connecting to an address.
type Direct struct {
}

func (d *Direct) Dial(network string, address string) (io.ReadWriteCloser, error) {
	return net.Dial(network, address)
}

// A RoadMode represents a host's road mode.
type RoadMode int

const (
	MLocale RoadMode = iota
	MRemote
	MFucked
	MPuzzle
)

// RULE file aims to be a minimal configuration file format that's easy to
// read due to obvious semantics.
// There are two parts per line on RULE file: mode and glob. mode are on the
// left of the space sign and glob are on the right. mode is an char and
// describes whether the host should go proxy, glob supported glob-style
// patterns:
//   h?llo matches hello, hallo and hxllo
//   h*llo matches hllo and heeeello
//   h[ae]llo matches hello and hallo, but not hillo
//   h[^e]llo matches hallo, hbllo, ... but not hello
//   h[a-b]llo matches hallo and hbllo
//
// This is a RULE document:
//   L a.com
//   R b.com
//   B c.com
//
// L(ocale)  means using locale network
// R(emote)  means using remote network
// B(anned)  means block it
type Rulels struct {
	Dict map[string]RoadMode
}

func (r *Rulels) Road(host string) RoadMode {
	for p, i := range r.Dict {
		b, err := filepath.Match(p, host)
		if err != nil {
			log.Panicln(err)
		}
		if !b {
			continue
		}
		return i
	}
	return MPuzzle
}

// Load a RULE file.
func (r *Rulels) Load(name string) error {
	f, err := aget.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		seps := strings.Split(line, " ")
		if len(seps) < 2 {
			continue
		}
		switch seps[0] {
		case "#":
		case "L":
			for _, e := range seps[1:] {
				r.Dict[e] = MLocale
			}
		case "R":
			for _, e := range seps[1:] {
				r.Dict[e] = MRemote
			}
		case "B":
			for _, e := range seps[1:] {
				r.Dict[e] = MFucked
			}
		}
	}
	return scanner.Err()
}

// NewRoaderRule returns a new RoaderRule.
func NewRulels() *Rulels {
	return &Rulels{
		Dict: map[string]RoadMode{},
	}
}

type Squire struct {
	Dialer Dialer
	Direct Dialer
	Memory acdb.Client
	Rulels *Rulels
	IPNets []*net.IPNet
}

func (s *Squire) Dial(network string, address string) (io.ReadWriteCloser, error) {
	host, _, err := net.SplitHostPort(address)
	mode := MPuzzle
	if err = s.Memory.Get(host, &mode); err == nil {
		switch mode {
		case MLocale:
			return s.Direct.Dial(network, address)
		case MRemote:
			return s.Dialer.Dial(network, address)
		}
		log.Panicln("unreachable")
	}
	switch s.Rulels.Road(host) {
	case MLocale:
		s.Memory.Set(host, MLocale)
		return s.Direct.Dial(network, address)
	case MRemote:
		s.Memory.Set(host, MRemote)
		return s.Dialer.Dial(network, address)
	case MFucked:
		return nil, fmt.Errorf("daze: %s has been blocked", host)
	case MPuzzle:
	}
	l, err := net.LookupIP(host)
	if err == nil && IPNetContains(s.IPNets, l[0]) {
		s.Memory.Set(host, MLocale)
		return s.Direct.Dial(network, address)
	} else {
		s.Memory.Set(host, MRemote)
		return s.Dialer.Dial(network, address)
	}
}

// NewSquire.
func NewSquire(dialer Dialer) *Squire {
	return &Squire{
		Dialer: dialer,
		Direct: &Direct{},
		Memory: acdb.Lru(1024),
		Rulels: NewRulels(),
	}
}
