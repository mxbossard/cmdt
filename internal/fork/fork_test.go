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
	suite := "foo1"

	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	k := 0
	work := func() {
		k++
	}

	done1, err := sched.schedule(suite, 1, work)
	assert.NoError(t, err)

	done2, err := sched.schedule(suite, 1, work)
	assert.NoError(t, err)

	// Wait work is done
	<-done1
	<-done2

	assert.Equal(t, 2, k)

	err = sched.clearSuite(suite)
	assert.NoError(t, err)
}

func TestFork2WorkScheduling(t *testing.T) {
	suite := "foo2"

	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	k := 0
	work := func() {
		k++
	}

	done1, err := sched.schedule(suite, 2, work)
	assert.NoError(t, err)

	done2, err := sched.schedule(suite, 2, work)
	assert.NoError(t, err)

	// Wait work is done
	<-done1
	<-done2

	assert.Equal(t, 2, k)

	err = sched.clearSuite(suite)
	assert.NoError(t, err)
}

func TestFork5WorkScheduling(t *testing.T) {
	suite := "foo5"

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
		_, err := sched.schedule(suite, 5, work)
		assert.NoError(t, err)
	}

	// Wait work is done
	wg.Wait()

	assert.Less(t, time.Since(started), sleepTime*3)
	assert.Greater(t, time.Since(started), sleepTime)
	assert.Equal(t, 5, k)

	err = sched.clearSuite(suite)
	assert.NoError(t, err)
}

func TestFork10By5WorkScheduling(t *testing.T) {
	suite := "foo10by5"

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
		_, err := sched.schedule(suite, 5, work)
		// fmt.Printf("Scheduled job %d\n", p)
		assert.NoError(t, err)
	}

	// Wait work is done
	wg.Wait()

	assert.Less(t, time.Since(started), sleepTime*3)
	assert.Greater(t, time.Since(started), sleepTime)
	assert.Equal(t, 10, k)

	err = sched.clearSuite(suite)
	assert.NoError(t, err)
}

func TestFork10By20WorkScheduling(t *testing.T) {
	suite1 := "foo10by20A"
	suite2 := "foo10by20B"

	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	sleepTime := 10 * time.Millisecond
	k := 0
	wg := sync.WaitGroup{}
	work := func() {
		time.Sleep(sleepTime)
		k++
		// fmt.Printf("job done k:%d\n", k)
		wg.Done()
	}

	started := time.Now()
	for p := 0; p < 10; p++ {
		wg.Add(2)
		_, err := sched.schedule(suite1, 10, work)
		assert.NoError(t, err)
		_, err = sched.schedule(suite2, 10, work)
		assert.NoError(t, err)
		//fmt.Printf("Scheduled job %d\n", p)
	}

	// Wait work is done
	wg.Wait()

	assert.Less(t, time.Since(started), sleepTime*4)
	assert.Greater(t, time.Since(started), sleepTime)
	assert.Equal(t, 20, k)

	err = sched.clearSuite(suite1)
	assert.NoError(t, err)

	err = sched.clearSuite(suite2)
	assert.NoError(t, err)
}

func TestFork20By4WorkScheduling(t *testing.T) {
	suite1 := "bar20by4A"
	suite2 := "bar20by4B"

	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	sleepTime := 10 * time.Millisecond
	k := 0
	wg := sync.WaitGroup{}
	work := func() {
		time.Sleep(sleepTime)
		k++
		//fmt.Printf("job done k:%d\n", k)
		wg.Done()
	}

	started := time.Now()
	for p := 0; p < 10; p++ {
		wg.Add(2)
		_, err := sched.schedule(suite1, 2, work)
		assert.NoError(t, err)
		_, err = sched.schedule(suite2, 2, work)
		assert.NoError(t, err)
		//fmt.Printf("Scheduled job %d\n", p)
	}

	// Wait work is done
	wg.Wait()

	assert.Less(t, time.Since(started), sleepTime*7)
	assert.Greater(t, time.Since(started), 5*sleepTime)
	assert.Equal(t, 20, k)

	err = sched.clearSuite(suite1)
	assert.NoError(t, err)

	err = sched.clearSuite(suite2)
	assert.NoError(t, err)
}

func TestWaitQueueComplete(t *testing.T) {
	suite := "fooWait"

	sched, err := instance()
	require.NoError(t, err)
	require.NotNil(t, sched)

	sleepTime := 10 * time.Millisecond
	k := 0
	work := func() {
		time.Sleep(sleepTime)
		k++
	}

	started := time.Now()
	for p := 0; p < 20; p++ {
		_, err := sched.schedule(suite, 5, work)
		assert.NoError(t, err)
	}

	// Wait work is done
	sched.waitQueueComplete(suite)

	assert.Equal(t, 20, k)
	assert.Less(t, time.Since(started), sleepTime*6)
	assert.Greater(t, time.Since(started), sleepTime*4)

	err = sched.clearSuite(suite)
	assert.NoError(t, err)
}
