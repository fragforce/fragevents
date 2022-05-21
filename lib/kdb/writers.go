package kdb

import (
	"context"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/segmentio/kafka-go"
	"sync"
)

type AllWriters struct {
	writers map[string]*kafka.Writer
	lock    *sync.Mutex
}

// W aka Writers is a globally shared set of topic:Writer instances
var W AllWriters

func init() {
	W = AllWriters{
		lock:    &sync.Mutex{},
		writers: map[string]*kafka.Writer{},
	}
}

//Get or create the requested writer for the given topic - ctx is only used for new connections!
func (w *AllWriters) Get(ctx context.Context, topic string) (*kafka.Writer, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	var err error
	wr, ok := w.writers[topic]
	if !ok {
		wr, err = NewKafkaWriter(ctx, topic)
		if err != nil {
			return nil, err
		}
		w.writers[topic] = wr
	}
	return wr, nil
}

func (w *AllWriters) Close() error {
	log := df.Log
	w.lock.Lock()
	defer w.lock.Unlock()
	var final error
	for topic, wr := range w.writers {
		log = log.WithField("kafka.writer.topic", topic)
		if err := wr.Close(); err != nil {
			final = err
			log.WithError(err).Error("Problem closing kafka writer")
		} else {
			log.Debug("Closed kafka writer successfully")
		}
	}
	return final
}
