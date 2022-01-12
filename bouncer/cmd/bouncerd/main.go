package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"github.com/platform9/pf9-qbert/bouncer/pkg/api"
	"github.com/platform9/pf9-qbert/bouncer/pkg/authn"
	"github.com/platform9/pf9-qbert/bouncer/pkg/keystone"
	"github.com/platform9/pf9-qbert/bouncer/pkg/policy"

	"golang.org/x/crypto/bcrypt"
)

const (
	URLv1                   = "/v1"
	defaultAuthTTL          = time.Duration(5) * time.Minute
	defaultUnauthTTL        = time.Duration(1) * time.Minute
	defaultKeystoneTimeout  = time.Duration(30) * time.Second
	defaultCacheSize        = 2048
	defaultBcryptCost       = 7
	defaultLogStatsInterval = time.Duration(10) * time.Minute
	minLogStatsInterval     = time.Duration(1) * time.Minute
	maxLogStatsInterval     = time.Duration(30) * time.Minute
)

var (
	addr                      string
	keystoneURL               string
	projectID                 string
	caFile, certFile, keyFile string
	authTTL, unauthTTL        time.Duration
	keystoneTimeout           time.Duration
	cacheSize                 int
	bcryptCost                int
	tlsConfig                 *tls.Config
	logStatsInterval          time.Duration
)

func init() {
	flag.DurationVar(&authTTL, "auth-ttl", defaultAuthTTL, "The `duration` to cache a response to authenticated request")
	flag.DurationVar(&unauthTTL, "unauth-ttl", defaultUnauthTTL, "The `duration` to cache a response to an unauthenticated request")
	flag.StringVar(&caFile, "ca-file", "", "The `path` of the CA certificate file")
	flag.StringVar(&certFile, "cert-file", "", "The `path` of the server certificate file")
	flag.StringVar(&keyFile, "key-file", "", "The `path` of the server certificate key file")
	flag.DurationVar(&keystoneTimeout, "keystone-timeout", defaultKeystoneTimeout, "The `duration` to wait for a response from Keystone")
	flag.IntVar(&cacheSize, "cache-size", defaultCacheSize, "The maximum `number` of distinct tokenIDs or credentials in the cache")
	flag.IntVar(&bcryptCost, "bcrypt-cost", defaultBcryptCost, fmt.Sprintf("The `cost` of hashing a credentials password. Cost outside interval [%d, %d] reverts to the default.", bcrypt.MinCost, bcrypt.MaxCost))
	flag.DurationVar(&logStatsInterval, "log-stats-interval", defaultLogStatsInterval, fmt.Sprintf("The interval `duration` on which metrics are written to the log. Duration outside interval [%s, %s] reverts to the default.", minLogStatsInterval, maxLogStatsInterval))
	flag.Parse()
	if flag.NArg() != 3 {
		usage()
		os.Exit(1)
	}
	switch definedPkiFlags() {
	case 0:
		break
	case 3:
		configureTLS()
	default:
		log.Fatal("Must set all or none of the [ca-file, cert-file, key-file] flags")
	}
	addr = flag.Arg(0)
	keystoneURL = flag.Arg(1)
	projectID = flag.Arg(2)
	if bcryptCost < bcrypt.MinCost || bcryptCost > bcrypt.MaxCost {
		log.Printf("bcrypt-cost %d outside interval [%d, %d]; reverting to default cost %d\n", bcryptCost, bcrypt.MinCost, bcrypt.MaxCost, defaultBcryptCost)
		bcryptCost = defaultBcryptCost
	}
	if logStatsInterval < minLogStatsInterval || logStatsInterval > maxLogStatsInterval {
		log.Printf("log-stats-interval %s outside interval [%s, %s]; reverting to default duration %s\n", logStatsInterval, minLogStatsInterval, maxLogStatsInterval, defaultLogStatsInterval)
		logStatsInterval = defaultLogStatsInterval
	}
}

func main() {
	k, err := keystone.New(keystoneURL, keystoneTimeout)
	if err != nil {
		log.Fatal("initialize keystone:", err)
	}
	r := policy.New()
	n, err := authn.New(k, projectID, authTTL, unauthTTL, cacheSize, bcryptCost, r)
	if err != nil {
		log.Fatal("initialize authenticator:", err)
	}
	printConfig()

	nTimer := metrics.NewTimer()
	m := http.NewServeMux()
	m.Handle(URLv1, timedHandler(n, nTimer))
	m.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})

	go logTimer(nTimer, logStatsInterval)

	var serverErr error
	s := &http.Server{Addr: addr, Handler: m, TLSConfig: tlsConfig}
	if tlsConfig != nil {
		serverErr = s.ListenAndServeTLS("", "")
	} else {
		serverErr = s.ListenAndServe()
	}
	log.Fatal("serve http:", serverErr)
}

func configureTLS() {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal("load cert-file and key-file:", err)
	}

	caPool := x509.NewCertPool()
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatal("read ca-file:", err)
	}
	if ok := caPool.AppendCertsFromPEM(caCert); !ok {
		log.Fatal("append ca-file cert to pool", err)
	}

	certs := make([]tls.Certificate, 1)
	certs[0] = cert
	tlsConfig = &tls.Config{RootCAs: caPool, Certificates: certs}
}

func printConfig() {
	var tlsCfgMsg string
	if tlsConfig != nil {
		tlsCfgMsg = fmt.Sprintf(" ca-file: %s. cert-file: %s. key-file: %s.", caFile, certFile, keyFile)
	}
	log.Printf("version: %s. listening on: %s. keystone-url: %s. project-id: %s. auth-ttl: %s. unauth-ttl: %s. keystone-timeout: %s. cache-size: %d. bcrypt-cost: %d. log-stats-interval: %s%s", api.Version, addr, keystoneURL, projectID, authTTL, unauthTTL, keystoneTimeout, cacheSize, bcryptCost, logStatsInterval, tlsCfgMsg)
}

func usage() {
	cmd := os.Args[0]
	msg := `Authentication Webhook for Platform9 Kubernetes clusters.
Version: %s
Usage: %s [OPTIONS] addr keystone-url project-id
`
	fmt.Printf(msg, api.Version, cmd)
	flag.PrintDefaults()
}

func definedPkiFlags() int {
	var count int
	if caFile != "" {
		count++
	}
	if certFile != "" {
		count++
	}
	if keyFile != "" {
		count++
	}
	return count
}

func timedHandler(h http.Handler, t metrics.Timer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		h.ServeHTTP(w, r)
		duration := time.Now().Sub(startTime)
		t.Update(duration)
	})
}

func logTimer(t metrics.Timer, freq time.Duration) {
	scale := time.Millisecond
	du := float64(scale)
	duSuffix := scale.String()[1:]
	buf := new(bytes.Buffer)

	for _ = range time.Tick(freq) {
		tSnapshot := t.Snapshot()
		ps := tSnapshot.Percentiles([]float64{0.50, 0.95, 0.99})
		buf.Reset()
		buf.WriteString(fmt.Sprintf("count: %d.", tSnapshot.Count()))
		buf.WriteString(fmt.Sprintf(" mean: %.f%s.", tSnapshot.Mean()/du, duSuffix))
		buf.WriteString(fmt.Sprintf(" median: %.f%s.", ps[0]/du, duSuffix))
		buf.WriteString(fmt.Sprintf(" 95%%: %.f%s.", ps[1]/du, duSuffix))
		buf.WriteString(fmt.Sprintf(" 99%%: %.f%s.", ps[2]/du, duSuffix))
		buf.WriteString(fmt.Sprintf(" 1m-rate: %.f.", tSnapshot.Rate1()))
		buf.WriteString(fmt.Sprintf(" 5m-rate: %.f.", tSnapshot.Rate5()))
		buf.WriteString(fmt.Sprintf(" 15m-rate: %.f.", tSnapshot.Rate15()))
		log.Println(buf.String())
	}
}
