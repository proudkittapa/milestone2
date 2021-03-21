//ref https://github.com/Lebonesco/go_lru_cache/blob/master/main.go
package cacheFile

import (
	"errors"
	"fmt"
)

const SIZE = 5 // size of cache

// maps data to node in Queue
type Hash (map[string]string)

// type hash map[int]byte

type Cache struct {
	Hash Hash
}

func NewCache() Cache {
	return Cache{Hash: Hash{}}
}

func (c *Cache) Check(str string) (string, error) {
	if _, ok := c.Hash[str]; ok {
		return c.Hash[str], nil
	} else {
		return "", errors.New("key doesn't exists")
	}
}

func (c *Cache) Remove(key string) {
	fmt.Printf("remove key: %s\n", key)
	delete(c.Hash, key)
}

func (c *Cache) Add(key string, value string) {
	c.Hash[key] = value
}

func (c *Cache) Display() {
	for key, _ := range c.Hash {
		fmt.Printf("{%s}\n", key)
	}
}
