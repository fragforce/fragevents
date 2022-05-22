package kdb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/segmentio/kafka-go"
	"github.com/spf13/viper"
	"net/url"
	"time"
)

func init() {
	viper.SetDefault("kafka.conn.timeout", 10*time.Second)
	viper.SetDefault("kafka.conn.idle", 300)
}

func newTLSConfig() (*tls.Config, error) {
	log := df.Log

	roots := x509.NewCertPool()

	ok := roots.AppendCertsFromPEM([]byte(viper.GetString("runtime.kafka_trusted_cert")))
	if !ok {
		err := errors.New("invalid kafka trusted cert")
		log.WithError(err).Error("Invalid kafka trusted cert")
		return nil, err
	}

	cert, err := tls.X509KeyPair(
		[]byte(viper.GetString("runtime.kafka_client_cert")),
		[]byte(viper.GetString("runtime.kafka_client_cert_key")),
	)
	if err != nil {
		log.WithError(err).Error("Problem loading kafka client key pair")
		return nil, err
	}

	t := tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, // Hostnames always wrong - use cert func
		RootCAs:            roots,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		MinVersion:         tls.VersionTLS13,
		Renegotiation:      tls.RenegotiateNever,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			vOpts := x509.VerifyOptions{Roots: roots}
			// Assume one cert
			for _, rawCert := range rawCerts {
				c, err := x509.ParseCertificate(rawCert)
				if err != nil {
					return err
				}
				_, err = c.Verify(vOpts)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	log.Trace("Created tls config")
	return &t, nil
}

func newKafkaTransport(ctx context.Context) (t *kafka.Transport, err error) {
	log := df.Log.WithContext(ctx)

	tlsConfig, err := newTLSConfig()
	if err != nil {
		log.WithError(err).Error("Problem creating new tls config")
		return nil, err
	}

	t = &kafka.Transport{
		DialTimeout: viper.GetDuration("kafka.conn.timeout"),
		IdleTimeout: viper.GetDuration("kafka.conn.idle"),
		//MetadataTTL: 0,
		ClientID: fmt.Sprintf(
			"%v-%v",
			viper.GetString("runtime.app_name"),
			viper.GetString("runtime.dyno_id"),
		),
		TLS: tlsConfig,
		//SASL:     nil,
		//Resolver: nil,
		Context: ctx,
	}

	log.Trace("Created new kafka transport")
	return
}

func NewKafkaWriter(ctx context.Context, topic string) (writer *kafka.Writer, err error) {
	log := df.Log.WithField("kafka.topic", topic).WithContext(ctx)

	transport, err := newKafkaTransport(ctx)
	if err != nil {
		log.WithError(err).Error("Problem creating kafka transport")
		return nil, err
	}

	kURLs := viper.GetStringSlice("kafka.urls")
	addrs := make([]string, len(kURLs))
	for _, kURL := range kURLs {
		log := log.WithField("url.raw", kURLs)
		u, err := url.ParseRequestURI(kURL)
		if err != nil {
			log.WithError(err).Error("Problem making url into url")
			return nil, err
		}
		addrs = append(addrs, u.Host)
	}

	writer = &kafka.Writer{
		Addr:                   kafka.TCP(addrs...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		MaxAttempts:            0,
		BatchSize:              0,
		BatchBytes:             0,
		BatchTimeout:           0,
		ReadTimeout:            0,
		WriteTimeout:           0,
		RequiredAcks:           0,
		Async:                  false,
		Completion:             nil,
		Compression:            0,
		Logger:                 nil,
		ErrorLogger:            nil,
		Transport:              transport,
		AllowAutoTopicCreation: false, // We can't do this in Heroku
	}

	log.Trace("Created kafka writer obj")

	return
}
