package app

import (
	"structs"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var appKitThread struct {
	once sync.Once
	err  error
	post func(func())
}

func loadAppKit() error {
	appKitThread.once.Do(func() { appKitThread.err = initAppKitThread() })
	return appKitThread.err
}

func initAppKitThread() error {
	system, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY)
	if err != nil {
		return err
	}
	queue, err := purego.Dlsym(system, "_dispatch_main_q")
	if err != nil {
		return err
	}
	var dispatch func(uintptr, uintptr, uintptr)
	purego.RegisterLibFunc(&dispatch, system, "dispatch_async_f")
	var mu sync.Mutex
	var jobs []func()
	drain := purego.NewCallback(func(uintptr) {
		mu.Lock()
		work := jobs
		jobs = nil
		mu.Unlock()
		pool := send(send(cls("NSAutoreleasePool"), "alloc"), "init")
		defer send(pool, "drain")
		for _, run := range work {
			run()
		}
	})
	appKitThread.post = func(run func()) {
		mu.Lock()
		first := len(jobs) == 0
		jobs = append(jobs, run)
		mu.Unlock()
		if first {
			dispatch(queue, 0, drain)
		}
	}
	return nil
}

type appKitPoint struct {
	_ structs.HostLayout
	X float64
	Y float64
}

type appKitSize struct {
	_      structs.HostLayout
	Width  float64
	Height float64
}

type appKitRect struct {
	_      structs.HostLayout
	Origin appKitPoint
	Size   appKitSize
}

func cocoaString(text objc.ID) string {
	value := objc.Send[*byte](text, objc.RegisterName("UTF8String"))
	if value == nil {
		return ""
	}
	length := objc.Send[uint](text, objc.RegisterName("lengthOfBytesUsingEncoding:"), uint(4))
	return string(unsafe.Slice(value, length))
}
