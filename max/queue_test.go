package max

import (
	"fmt"
	"testing"
)

type testItem struct {
	Key      string
	Value    string
	Priority int
}

func Test_PQBasic(test *testing.T) {
	q := NewPriorityQueue(10)

	testData := []*testItem{
		&testItem{"file5", "hash5", 5},
		&testItem{"file1", "hash1", 1},
		&testItem{"file2", "hash2", 2},
		&testItem{"file3", "hash3", 3},
		&testItem{"file4", "hash4", 4},
	}

	for _, data := range testData {
		item := &Item{
			Key:      data.Key,
			Value:    data.Value,
			Priority: data.Priority,
		}
		q.Push(item)
	}
	fmt.Println(q.Len())
	printFirstItem(q)

	key := "file2"
	index := q.IndexByKey(key)
	fmt.Printf("index %d for key %s\n", index, key)
	item := q.ItemByKey(key)
	fmt.Printf("item %v for key %s\n", item, key)

	key2 := "file1"
	q.Update(key2, "new hash 1", 10)
	printFirstItem(q)

	removed := q.Remove(key)
	fmt.Printf("remove for key %s result %v\n", key, removed)
	printFirstItem(q)

	for q.Len() > 0 {
		item := q.Pop()
		fmt.Printf("%v\n", item)
	}
}

func printFirstItem(pq *PriorityQueue) {
	item := pq.FirstItem()
	fmt.Printf("first item : %v\n", item)
}

func Test_PQBasic2(test *testing.T) {
	q := NewPriorityQueue(10)

	testData := []*testItem{
		&testItem{"file5", "hash5", 5},
	}

	for _, data := range testData {
		item := &Item{
			Key:      data.Key,
			Value:    data.Value,
			Priority: data.Priority,
		}
		q.Push(item)
	}
	fmt.Println(q.Len())
	printFirstItem(q)

	item := q.Pop()
	fmt.Printf("%v\n", item)
}
