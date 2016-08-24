/*
* Copyright 2016 DGraph Labs, Inc.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*         http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package worker

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"

	"github.com/dgraph-io/dgraph/store"
)

// Data represents key-value data stored in RocksDB.
type Data struct {
	Key   []byte
	Value []byte
}

func (d *Data) encode() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	rerr := enc.Encode(*d)
	return b.Bytes(), rerr
}

func (d *Data) decode(data []byte) error {
	r := bytes.NewReader(data)
	dec := gob.NewDecoder(r)
	return dec.Decode(d)
}

// Predicate gets data for a predicate p from another instance and writes it to RocksDB.
func Predicate(p string, idx int) error {
	var err error
	var kvs []store.KV

	pool := pools[idx]
	query := new(Payload)
	query.Data = []byte(p)
	if err != nil {
		return err
	}

	conn, err := pool.Get()
	if err != nil {
		return err
	}
	defer pool.Put(conn)
	c := NewWorkerClient(conn)

	stream, err := c.PredicateData(context.Background(), query)
	if err != nil {
		return err
	}
	for {
		b, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		d := new(Data)
		if err := d.decode(b.Data); err != nil {
			return err
		}
		kvs = append(kvs, store.KV{K: d.Key, V: d.Value})
		// We do a batch write once we have 100 key value pairs.
		if len(kvs) == 100 {
			if err := dataStore.WriteBatch(kvs); err != nil {
				return err
			}
			kvs = nil
		}
	}
	// As we write only every 100 key value pairs, there would be some left usually.
	if len(kvs) > 0 {
		if err := dataStore.WriteBatch(kvs); err != nil {
			return err
		}
	}
	return nil
}
