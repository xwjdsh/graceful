package graceful

import (
	"log"
	"syscall"
	"testing"
	"time"
)

func TestShutdown(t *testing.T) {
	done := make(chan struct{})
	s := New()
	go func() {
		s.RegisterFunc(func() error {
			log.Println("[stop func 1] start")
			time.Sleep(1 * time.Second)
			log.Println("[stop func 1] end")
			return nil
		}).RegisterFunc(func() error {
			log.Println("[stop func 2] start")
			time.Sleep(5 * time.Second)
			log.Println("[stop func 2] end")
			return nil
		}).RegisterFunc(func() error {
			log.Println("[stop func 3] start")
			time.Sleep(3 * time.Second)
			log.Println("[stop func 3] end")
			return nil
		}).Shutdown()
		close(done)
	}()

	// waiting for signal.Notify
	time.Sleep(1 * time.Second)

	s.stop <- syscall.SIGINT
	select {
	case <-time.After(7 * time.Second):
		t.Error("timeout")
	case <-done:
	}
}

func TestTimeoutShutdown(t *testing.T) {
	done := make(chan struct{})
	go func() {
		SetSignals(syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
		SetTimeout(3 * time.Second)
		RegisterFunc(func() error {
			log.Println("[stop func] start")
			time.Sleep(5 * time.Second)
			log.Println("[stop func] end")
			return nil
		})
		Shutdown()
		close(done)
	}()

	// waiting for signal.Notify
	time.Sleep(1 * time.Second)

	defaultManager.stop <- syscall.SIGABRT
	select {
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	case <-done:
	}
}

func TestSignalThresholdShutdown(t *testing.T) {
	done := make(chan struct{})
	st := 8
	s := New().SetSignalThreshold(st)
	go func() {
		s.RegisterFunc(func() error {
			log.Println("[stop func] start")
			time.Sleep(30 * time.Second)
			log.Println("[stop func] end")
			return nil
		}).Shutdown()
		close(done)
	}()

	// waiting for signal.Notify
	time.Sleep(1 * time.Second)

	for i := 0; i < st; i++ {
		s.stop <- syscall.SIGINT
	}
	select {
	case <-time.After(2 * time.Second):
		t.Error("timeout")
	case <-done:
	}
}
