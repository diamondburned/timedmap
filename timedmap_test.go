package timedmap

import (
	"runtime"
	"testing"
	"time"
)

func BenchmarkMap(b *testing.B) {
	tm := New()

	for i := 0; i < b.N; i++ {
		tm.Set("hime", "arikawa", 99999999999999)
		_ = tm.GetValue("hime").(string)
	}
}

func BenchmarkConcurrentRead(b *testing.B) {
	tm := New()
	tm.Set("hime", "arikawa", 99999999999)

	b.SetParallelism(runtime.NumCPU() * 2)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tm.GetValue("hime")
		}
	})
}

func BenchmarkConcurrentWrite(b *testing.B) {
	tm := New()

	b.SetParallelism(runtime.NumCPU() * 2)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tm.Set("hime", "arikawa", 99999999999)
		}
	})
}

const cleanupTick = 10 * time.Millisecond

func newTmap(t *testing.T) *Map {
	tm := New()
	cl := NewCleaner(cleanupTick)
	cl.AddCleanable(tm)
	t.Cleanup(cl.Stop)
	return tm
}

func TestFlush(t *testing.T) {
	var tm = newTmap(t)

	for i := 0; i < 10; i++ {
		tm.Set(i, 1, time.Hour)
	}
	tm.Flush()
	if s := tm.Size(); s > 0 {
		t.Fatalf("size was %d > 0", s)
	}
}

func TestSet(t *testing.T) {
	tm := newTmap(t)

	key := "tKeySet"
	val := "tValSet"

	tm.Set(key, val, 20*time.Millisecond)
	vl, ok := tm.get(key)
	if !ok {
		t.Fatal("key was not set")
	}

	if vl.Value.(string) != val {
		t.Fatal("value was not like set")
	}

	time.Sleep(20*time.Millisecond + cleanupTick)

	if v := tm.GetValue(key); v != nil {
		t.Fatal("key was not deleted after expire")
	}

	tm.Flush()
}

func TestGetValue(t *testing.T) {
	tm := newTmap(t)

	key := "tKeyGetVal"
	val := "tValGetVal"

	tm.Set(key, val, 50*time.Millisecond)

	if tm.GetValue("keyNotExists") != nil {
		t.Fatal("non existent key was not nil")
	}

	v := tm.GetValue(key)
	if v == nil {
		t.Fatal("value was nil")
	}
	if vStr := v.(string); vStr != val {
		t.Fatalf("got value was %s != 'tValGetVal'", vStr)
	}

	time.Sleep(60 * time.Millisecond)

	v = tm.GetValue(key)
	if v != nil {
		t.Fatal("key was not deleted after expiration time")
	}

	tm.Set(key, val, 1*time.Microsecond)

	time.Sleep(2 * time.Millisecond)

	if tm.GetValue(key) != nil {
		t.Fatal("expired key was not removed by get func")
	}

	tm.Flush()
}

func TestGetExpire(t *testing.T) {
	tm := newTmap(t)

	key := "tKeyGetExp"
	val := "tValGetExp"

	tm.Set(key, val, 50*time.Millisecond)
	ct := time.Now().Add(50 * time.Millisecond)

	ex, ok := tm.GetExpires(key)
	if !ok {
		t.Fatal(key, "does not exist.")
	}

	if d := ct.Sub(ex); d > 1*time.Millisecond {
		t.Fatalf("expire date diff was %d > 1 millisecond", d)
	}

	tm.Flush()
}

func TestContains(t *testing.T) {
	tm := newTmap(t)

	key := "tKeyCont"

	tm.Set(key, 1, 30*time.Millisecond)

	if tm.Contains("keyNotExists") {
		t.Fatal("non existing key was detected as containing")
	}

	if !tm.Contains(key) {
		t.Fatal("containing key was detected as not containing")
	}

	time.Sleep(50 * time.Millisecond)
	if tm.Contains(key) {
		t.Fatal("expired key was detected as containing")
	}

	tm.Flush()
}

func TestRemove(t *testing.T) {
	tm := newTmap(t)

	key := "tKeyRem"

	tm.Set(key, 1, time.Hour)
	tm.Remove(key)

	if _, ok := tm.get(key); ok {
		t.Fatal("key still exists after remove")
	}

	tm.Flush()
}

func TestExtend(t *testing.T) {
	tm := newTmap(t)

	const key = "tKeyRef"

	if ok := tm.Extend("keyNotExists", time.Hour); ok {
		t.Fatal("Non-existing key was refreshed.")
	}

	tm.Set(key, 1, 20*time.Millisecond)

	if ok := tm.Extend(key, 30*time.Millisecond); !ok {
		t.Fatal("Failed to refresh key.")
	}

	time.Sleep(30 * time.Millisecond)

	if v := tm.GetValue(key); v == nil {
		t.Fatal("Key was not extended.")
	}

	time.Sleep(20*time.Millisecond + cleanupTick)

	if _, ok := tm.get(key); ok {
		t.Fatal("key was not deleted after refreshed time")
	}

	tm.Flush()
}

func TestSize(t *testing.T) {
	var tm = newTmap(t)

	for i := 0; i < 25; i++ {
		tm.Set(i, 1, 50*time.Millisecond)
	}
	if s := tm.Size(); s != 25 {
		t.Fatalf("size was %d != 25", s)
	}

	tm.Flush()
}
