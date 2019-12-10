package main

import (
	"fmt"
	"github.com/boltdb/bolt"
	"time"
)

var db *bolt.DB

const dbName = "pixelbot.db"

// main buckets
const (
	riotBucket    = "riot"
	generalBucket = "general"
)

// riot sub buckets
const (
	riotVerifyBucket = "verify"
	riotQuotesBucket = "quotes"
)

// general sub buckets
const (
	generalServersBucket = "servers"
)

var dbInitMap = map[string][]string{
	riotBucket:    {riotVerifyBucket, riotQuotesBucket},
	generalBucket: {generalServersBucket},
}

func initDB() error {
	var err error
	db, err = bolt.Open(dbName, 0666, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		fmt.Println("Error opening riot.db")
		return err
	}
	return db.Update(func(tx *bolt.Tx) error {
		for b1, b2s := range dbInitMap{
			b, err := tx.CreateBucketIfNotExists([]byte(b1))
			if err != nil {
				fmt.Println("Error making bucket:",b1,err)
				return err
			}
			for _, b2 := range b2s {
				_, err = b.CreateBucketIfNotExists([]byte(b2))
				if err != nil {
					fmt.Println("Error making sub bucket:",b1,b2,err)
					return err
				}
			}
		}
		b := tx.Bucket([]byte(generalBucket))
		val := b.Get([]byte("commands"))
		if len(val) == 0 {
			b.Put([]byte("commands"), make([]byte,8))
		}
		return nil
	})
}
