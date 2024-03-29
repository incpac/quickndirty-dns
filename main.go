package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type result struct {
	Name  string
	Value string
}

type configuration struct {
	Results []result
}

var configPath string
var conf configuration
var port int
var logfile string

// Version is set during compilation.
// It dictates what is returned by qnddns --version.
var Version string

func parseQuery(m *dns.Msg) {

	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			var result string

			for _, r := range conf.Results {
				if strings.ToLower(r.Name) == strings.ToLower(q.Name) {
					result = r.Value
				}
			}

			if result == "" {
				ip, _ := net.LookupIP(q.Name)

				if ip != nil {
					result = ip[0].String()
				}
			}

			if result != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, result))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				} else {
					log.Printf("Failed to create DNS response: %s", err.Error())
				}
			}
		}
	}
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m)
	}

	var src net.IP
	if w.RemoteAddr().Network() == "tcp" {
		src = w.RemoteAddr().(*net.TCPAddr).IP
	} else {
		src = w.RemoteAddr().(*net.UDPAddr).IP
	}

	if m.Answer != nil {
		response := strings.Split(m.Answer[0].String(), "\t")[4]
		log.Printf("Received request from %s for %s Returned %s", src, m.Question[0].Name, response)
	} else {
		log.Printf("Received request from %s for %s Unable to find valid response", src, m.Question[0].Name)
	}

	w.WriteMsg(m)
}

func serve(cmd *cobra.Command, args []string) {

	viper.SetConfigFile(configPath)

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Failed to open configuration file: %s\n", err.Error())
		os.Exit(-1)
	}

	err = viper.Unmarshal(&conf)
	if err != nil {
		log.Fatalf("Failed to unmarshal configuration: %s\n", err.Error())
		os.Exit(-1)
	}

	dns.HandleFunc(".", handleDNSRequest)

	go func() {
		udp := &dns.Server{Addr: ":" + strconv.Itoa(port), Net: "udp"}
		err = udp.ListenAndServe()
		if err != nil {
			log.Fatalf("Failed to start UDP server: %s\n", err.Error())
			os.Exit(-1)
		}
	}()

	go func() {
		tcp := &dns.Server{Addr: ":" + strconv.Itoa(port), Net: "tcp"}
		err = tcp.ListenAndServe()
		if err != nil {
			log.Fatalf("Failed to start TCP server: %s\n", err.Error())
			os.Exit(-1)
		}
	}()

	sc := make(chan os.Signal)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)

	s := <-sc
	log.Fatalf("Signal (%v) received, stopping\n", s)
}

func main() {

	var showVersion bool

	command := &cobra.Command{
		Use:   "qnddns",
		Short: "A quick 'n' dirty DNS server",
		Long:  "A simple DNS server designed for resolving DNS issues",
		Run: func(cmd *cobra.Command, args []string) {
			if showVersion {
				fmt.Printf("%s\n", Version)
			} else {
				if logfile != "" {
					f, _ := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
					defer f.Close()
					log.SetOutput(f)
				}
				serve(cmd, args)
			}
		},
	}

	command.Flags().StringVarP(&configPath, "config", "c", "./config.json", "path to the configuration file")
	command.Flags().IntVarP(&port, "port", "p", 53, "the port to listen on")
	command.Flags().BoolVarP(&showVersion, "version", "v", false, "display the version")
	command.Flags().StringVarP(&logfile, "logfile", "", "", "file to write logs to")

	if err := command.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
}
