package max

import (
	"container/heap"
	"fmt"
	"sync"
)

type Item struct {
	Key      interface{}
	Value    interface{}
	Priority int
	Index    int
}

type ItemSlice []*Item

func (this ItemSlice) Len() int {
	return len(this)
}

// smaller priority comes first
func (this *ItemSlice) Less(i, j int) bool {
	return (*this)[i].Priority < (*this)[j].Priority
}

func (this *ItemSlice) Swap(i, j int) {
	(*this)[i], (*this)[j] = (*this)[j], (*this)[i]
	(*this)[i].Index = i
	(*this)[j].Index = j
}

func (this *ItemSlice) Push(x interface{}) {
	n := len(*this)
	item := x.(*Item)
	item.Index = n
	(*this) = append((*this), item)
}

func (this *ItemSlice) Pop() interface{} {
	old := *this
	n := len(old)
	item := old[n-1]
	item.Index = -1
	(*this) = old[0 : n-1]
	return item
}
func (this *ItemSlice) Update(item *Item, value interface{}, priority int) {
	item.Value = value
	item.Priority = priority
	heap.Fix(this, item.Index)
}

type PriorityQueue struct {
	Items      ItemSlice
	Size       int
	PushNotify chan struct{}
	lock       sync.RWMutex
}

func NewPriorityQueue(size int) *PriorityQueue {
	queue := &PriorityQueue{
		Items:      make([]*Item, 0, size),
		Size:       size,
		PushNotify: make(chan struct{}, size),
	}
	heap.Init(&queue.Items)
	return queue
}

func (this *PriorityQueue) Push(item *Item) error {
	if item == nil {
		return fmt.Errorf("item is nil")
	}

	this.lock.Lock()
	if this.Items.Len() >= this.Size {
		this.lock.Unlock()
		return fmt.Errorf("queue is full")
	}

	if this.indexByKey(item.Key) != -1 {
		this.lock.Unlock()
		return fmt.Errorf("item exist for key %v", item.Key)
	}
	heap.Push(&this.Items, item)
	this.lock.Unlock()

	go this.NotifyPush()
	return nil
}

func (this *PriorityQueue) Pop() *Item {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.Items.Len() == 0 {
		return nil
	}
	return heap.Pop(&this.Items).(*Item)
}

func (this *PriorityQueue) Len() int {
	return this.Items.Len()
}

func (this *PriorityQueue) FirstItem() *Item {
	this.lock.RLock()
	defer this.lock.RUnlock()

	if this.Items.Len() == 0 {
		return nil
	}
	return this.Items[0]
}

func (this *PriorityQueue) Update(key interface{}, value interface{}, priority int) (updated bool) {
	this.lock.Lock()
	defer this.lock.Unlock()

	item := this.itemByKey(key)
	if item == nil {
		return false
	}
	this.Items.Update(item, value, priority)
	return true
}

func (this *PriorityQueue) IndexByKey(key interface{}) int {
	this.lock.RLock()
	defer this.lock.RUnlock()

	for index, item := range this.Items {
		if item.Key == key {
			return index
		}
	}
	return -1
}

func (this *PriorityQueue) indexByKey(key interface{}) int {
	for index, item := range this.Items {
		if item.Key == key {
			return index
		}
	}
	return -1
}

func (this *PriorityQueue) ItemByKey(key interface{}) *Item {
	this.lock.RLock()
	defer this.lock.RUnlock()

	item := this.itemByKey(key)
	return item
}

func (this *PriorityQueue) itemByKey(key interface{}) *Item {
	index := this.indexByKey(key)
	if index == -1 {
		return nil
	}
	return this.Items[index]
}

func (this *PriorityQueue) Remove(key interface{}) (removed bool) {
	this.lock.Lock()
	defer this.lock.Unlock()

	index := this.indexByKey(key)
	if index == -1 {
		return false
	}
	this.Items = append(this.Items[0:index], this.Items[index+1:]...)
	heap.Init(&this.Items)
	return true
}

func (this *PriorityQueue) GetPushNotifyChan() chan struct{} {
	return this.PushNotify
}

func (this *PriorityQueue) NotifyPush() {
	this.PushNotify <- struct{}{}
}
