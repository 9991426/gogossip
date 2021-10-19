package main

import (
	"sync"
)

// BulkWaitGroup provides additional controls over a WaitGroup, to allow for
// easier use in concurrency.
type BulkWaitGroup struct {
	sync.WaitGroup
	waitergroup sync.WaitGroup
	count       int64
	waiters     int64
	ready       chan struct{}
	start_lock  sync.Mutex
	wait_lock   sync.Mutex
}

// Start adds count to both the main WaitGroup and the waitergroup, if the
// WaitGroup has already been reset. If there is an existing value, it does
// nothing.
func (b *BulkWaitGroup) Add(count int) {
	dbgPrint(2, "start")
	b.start_lock.Lock()

	if b.ready == nil {
		dbgPrint(2, "make chan")
		b.ready = make(chan struct{}, 1)
		b.ready <- struct{}{}
	}

	if b.count == 0 {
		dbgPrint(2, "STARTER")
		// Waiting for the ready signal ensures this WaitGroup task does not start
		// until all the other waiters have cleared the previous one.
		<-b.ready
		dbgPrint(2, "ready")

		b.count = int64(count)
		b.WaitGroup.Add(count)
		b.waitergroup.Add(count)
	}

	b.start_lock.Unlock()
}

// Done calls WaitGroup.Done(), wile decrementing the count.
func (b *BulkWaitGroup) Done() {
	dbgPrint(2, "done")
	b.start_lock.Lock()
	b.count -= 1
	b.start_lock.Unlock()

	b.WaitGroup.Done()
}

// Wait synchronizes the goroutines waiting on the WaitGroup. First, it waits
// until all goroutines have started waiting, using waitergroup. Then, it waits
// until the main WaitGroup completes. Finally, when all waiters have received
// the completion signal, it resets the BulkWaitGroup and gives the ready signal.
func (b *BulkWaitGroup) Wait() {
	b.wait_lock.Lock()
	b.waiters += 1
	b.waitergroup.Done()
	b.wait_lock.Unlock()

	b.waitergroup.Wait()
	b.WaitGroup.Wait()

	dbgPrint(2, "update waiters")

	b.wait_lock.Lock()
	b.waiters -= 1
	dbgPrint(2, "wait", b.waiters)
	if b.waiters == 0 {
		dbgPrint(2, "send ready")
		b.ready <- struct{}{}
	}
	b.wait_lock.Unlock()
}
