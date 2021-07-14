package repository

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
)

const (
	riotBucket       = "riot"
	riotVerifyBucket = "verify"
	riotQuotesBucket = "quotes"
	generalBucket    = "general"
)

type Bolt struct {
	db *bolt.DB
}

func NewBolt(db *bolt.DB) *Bolt {
	db.Update(func(t *bolt.Tx) error {
		t.CreateBucketIfNotExists([]byte(generalBucket))
		b, _ := t.CreateBucketIfNotExists([]byte(riotBucket))
		b.CreateBucketIfNotExists([]byte(riotVerifyBucket))
		b.CreateBucketIfNotExists([]byte(riotQuotesBucket))
		return nil
	})
	return &Bolt{db: db}
}

func (b *Bolt) IncrementCommandCount() {
	b.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(generalBucket))
		val := binary.BigEndian.Uint64(b.Get([]byte("commands")))
		cmdBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(cmdBytes, val+1)
		b.Put([]byte("commands"), cmdBytes)
		return nil
	})
}

func (b *Bolt) CommandCount() uint64 {
	var val uint64
	b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(generalBucket))
		val = binary.BigEndian.Uint64(b.Get([]byte("commands")))
		return nil
	})
	return val
}

func (b *Bolt) SetRiotQuote(id, quote string) {
	b.db.Update(func(tx *bolt.Tx) error {
		quotes := tx.Bucket([]byte(riotBucket)).Bucket([]byte(riotQuotesBucket))
		quotes.Put([]byte(fmt.Sprintf("%v", id)), []byte(quote))
		return nil
	})
}

func (b *Bolt) RiotQuote(id string) string {
	quote := ""
	b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(riotBucket)).Bucket([]byte(riotQuotesBucket))
		v := b.Get([]byte(fmt.Sprintf("%v", id)))
		quote = string(v)
		return nil
	})
	return quote
}

func (b *Bolt) SetRiotVerified(discordID, riotID string) {
	b.db.Update(func(tx *bolt.Tx) error {
		verify := tx.Bucket([]byte(riotBucket)).Bucket([]byte(riotVerifyBucket))
		verify.Put([]byte(fmt.Sprintf("%v%v", discordID, riotID)), []byte("1"))
		return nil
	})
}

func (b *Bolt) RiotVerified(discordID, riotID string) bool {
	return b.db.View(func(tx *bolt.Tx) error {
		verify := tx.Bucket([]byte(riotBucket)).Bucket([]byte(riotVerifyBucket))
		if result := verify.Get([]byte(fmt.Sprintf("%v%v", discordID, riotID))); result != nil {
			return nil
		}
		return errors.New("Error: You are not verified for this account")
	}) == nil
}
