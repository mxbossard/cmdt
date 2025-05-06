package fork

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotForkedWorkScheduling(t *testing.T) {
	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	k := 0
	work := func() {
		k++
	}

	done1, err := sched.schedule("foo", 1, work)
	assert.NoError(t, err)

	done2, err := sched.schedule("foo", 1, work)
	assert.NoError(t, err)

	// Wait work is done
	<-done1
	<-done2

	assert.Equal(t, 2, k)
}

func TestFork2WorkScheduling(t *testing.T) {
	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	k := 0
	work := func() {
		k++
	}

	done1, err := sched.schedule("foo", 2, work)
	assert.NoError(t, err)

	done2, err := sched.schedule("foo", 2, work)
	assert.NoError(t, err)

	// Wait work is done
	<-done1
	<-done2

	assert.Equal(t, 2, k)
}

func TestFork5WorkScheduling(t *testing.T) {
	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	sleepTime := 10 * time.Millisecond
	k := 0
	wg := sync.WaitGroup{}
	work := func() {
		time.Sleep(sleepTime)
		k++
		wg.Done()
	}

	started := time.Now()
	for p := 0; p < 5; p++ {
		wg.Add(1)
		_, err := sched.schedule("bar", 5, work)
		assert.NoError(t, err)
	}

	// Wait work is done
	wg.Wait()

	assert.Less(t, time.Since(started), sleepTime*3)
	assert.Greater(t, time.Since(started), sleepTime)
	assert.Equal(t, 5, k)
}

func TestFork10By5WorkScheduling(t *testing.T) {
	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	sleepTime := 10 * time.Millisecond
	k := 0
	wg := sync.WaitGroup{}
	work := func() {
		time.Sleep(sleepTime)
		k++
		fmt.Printf("job done k:%d\n", k)
		wg.Done()
	}

	started := time.Now()
	for p := 0; p < 10; p++ {
		wg.Add(1)
		_, err := sched.schedule("baz", 5, work)
		fmt.Printf("Scheduled job %d\n", p)
		assert.NoError(t, err)
	}

	// Wait work is done
	wg.Wait()

	assert.Less(t, time.Since(started), sleepTime*3)
	assert.Greater(t, time.Since(started), sleepTime)
	assert.Equal(t, 10, k)
}

func TestFork10By20WorkScheduling(t *testing.T) {
	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	sleepTime := 10 * time.Millisecond
	k := 0
	wg := sync.WaitGroup{}
	work := func() {
		time.Sleep(sleepTime)
		k++
		fmt.Printf("job done k:%d\n", k)
		wg.Done()
	}

	started := time.Now()
	for p := 0; p < 10; p++ {
		wg.Add(2)
		_, err := sched.schedule("pif", 10, work)
		assert.NoError(t, err)
		_, err = sched.schedule("paf", 10, work)
		assert.NoError(t, err)
		//fmt.Printf("Scheduled job %d\n", p)
	}

	// Wait work is done
	wg.Wait()

	assert.Less(t, time.Since(started), sleepTime*3)
	assert.Greater(t, time.Since(started), sleepTime)
	assert.Equal(t, 20, k)
}

func TestFork20By10WorkScheduling(t *testing.T) {
	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	sleepTime := 10 * time.Millisecond
	k := 0
	wg := sync.WaitGroup{}
	work := func() {
		time.Sleep(sleepTime)
		k++
		fmt.Printf("job done k:%d\n", k)
		wg.Done()
	}

	started := time.Now()
	for p := 0; p < 10; p++ {
		wg.Add(2)
		_, err := sched.schedule("puf", 5, work)
		assert.NoError(t, err)
		_, err = sched.schedule("pof", 5, work)
		assert.NoError(t, err)
		//fmt.Printf("Scheduled job %d\n", p)
	}

	// Wait work is done
	wg.Wait()

	assert.Less(t, time.Since(started), sleepTime*5)
	assert.Greater(t, time.Since(started), 2*sleepTime)
	assert.Equal(t, 20, k)
}
