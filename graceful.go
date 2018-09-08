package graceful

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type (
	stopFunc func() error
	Stopper  interface {
		Stop() error
	}

	Manager struct {
		fs              []stopFunc
		signalThreshold int
		stop            chan os.Signal
		timeout         time.Duration
		signals         []os.Signal
	}
)

var defaultManager = New()

func New() *Manager {
	return &Manager{
		timeout:         30 * time.Second,
		signals:         []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		signalThreshold: 1,
		stop:            make(chan os.Signal),
	}
}

func SetSignalThreshold(st int) *Manager { return defaultManager.SetSignalThreshold(st) }
func (m *Manager) SetSignalThreshold(st int) *Manager {
	if st > 1 {
		m.signalThreshold = st
	}
	return m
}

func SetTimeout(timeout time.Duration) *Manager { return defaultManager.SetTimeout(timeout) }
func (m *Manager) SetTimeout(timeout time.Duration) *Manager {
	m.timeout = timeout
	return m
}

func SetSignals(signals ...os.Signal) *Manager { return defaultManager.SetSignals(signals...) }
func (m *Manager) SetSignals(signals ...os.Signal) *Manager {
	m.signals = signals
	return m
}

func Register(stopper Stopper) *Manager { return defaultManager.Register(stopper) }
func (m *Manager) Register(stopper Stopper) *Manager {
	return m.RegisterFunc(stopper.Stop)
}

func RegisterFunc(f stopFunc) *Manager { return defaultManager.RegisterFunc(f) }
func (m *Manager) RegisterFunc(f func() error) *Manager {
	m.fs = append(m.fs, f)
	return m
}

func Shutdown() { defaultManager.Shutdown() }
func (m *Manager) Shutdown() {
	signalThreshold := m.signalThreshold
	signal.Notify(m.stop, m.signals...)
	signalThreshold--
	log.Printf("[graceful] starting shutdown, signal: %s, countdown: %d\n", <-m.stop, signalThreshold)

	wg := new(sync.WaitGroup)
	for _, f := range m.fs {
		f := f
		go func() {
			wg.Add(1)
			defer wg.Done()

			if err := f(); err != nil {
				log.Printf("[graceful] stop error, error: %v\n", err)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	for {
		select {
		case <-time.After(m.timeout):
			log.Printf("[graceful] forced shutdown due to timeout, timeout: %v\n", m.timeout)
			return
		case sig := <-m.stop:
			signalThreshold--
			if signalThreshold > 0 {
				log.Printf("[graceful] received signal, signal: %v, countdown: %d\n", sig, signalThreshold)
			} else if signalThreshold == 0 {
				log.Printf("[graceful] forced shutdown after %d consecutive signals, signal: %v\n", m.signalThreshold, sig)
				return
			}
		case <-done:
			log.Println("[graceful] normal shutdown")
			return
		}
	}
}
