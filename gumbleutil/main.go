package gumbleutil // import "github.com/5pm-HDH/gumble/gumbleutil"

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/5pm-HDH/gumble/gumble"
	"net"
	"os"
	"strconv"
)

// Main aids in the creation of a basic command line gumble bot. It accepts the
// following flag arguments:
//  --server
//  --username
//  --password
//  --insecure,
//  --certificate
//  --key
func Main(listeners ...gumble.EventListener) {
	server := flag.String("server", "localhost:64738", "Mumble server address")
	username := flag.String("username", "gumble-bot", "client username")
	password := flag.String("password", "", "client password")
	insecure := flag.Bool("insecure", false, "skip server certificate verification")
	certificateFile := flag.String("certificate", "", "user certificate file (PEM)")
	keyFile := flag.String("key", "", "user certificate key file (PEM)")

	if !flag.Parsed() {
		flag.Parse()
	}

	host, port, err := net.SplitHostPort(*server)
	if err != nil {
		host = *server
		port = strconv.Itoa(gumble.DefaultPort)
	}

	keepAlive := make(chan bool)

	config := gumble.NewConfig()
	config.Username = *username
	config.Password = *password
	address := net.JoinHostPort(host, port)

	var tlsConfig tls.Config

	if *insecure {
		tlsConfig.InsecureSkipVerify = true
	}
	if *certificateFile != "" {
		if *keyFile == "" {
			keyFile = certificateFile
		}
		if certificate, err := tls.LoadX509KeyPair(*certificateFile, *keyFile); err != nil {
			fmt.Printf("%s: %s\n", os.Args[0], err)
			os.Exit(1)
		} else {
			tlsConfig.Certificates = append(tlsConfig.Certificates, certificate)
		}
	}
	config.Attach(AutoBitrate)
	for _, listener := range listeners {
		config.Attach(listener)
	}
	config.Attach(Listener{
		Disconnect: func(e *gumble.DisconnectEvent) {
			keepAlive <- true
		},
	})
	_, err = gumble.DialWithDialer(new(net.Dialer), address, config, &tlsConfig)
	if err != nil {
		fmt.Printf("%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	<-keepAlive
}
