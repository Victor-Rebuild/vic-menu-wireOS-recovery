package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/kercre123/vector-gobot/pkg/vbody"
)

// recreate GetFrame

var currentFrame vbody.DataFrame
var inited bool
var stopFrameGetter chan bool
var frameGetterStopped chan bool
var stopFrames bool
var mu sync.Mutex

func GetFrame() vbody.DataFrame {
	mu.Lock()
	defer mu.Unlock()
	return currentFrame
}

func StopFrameGetter() {
	go func() {
		time.Sleep(time.Millisecond * 10)
		stopFrameGetter <- true
	}()
	for range frameGetterStopped {
		return
	}
}

func InitFrameGetter() {
	frameGetterStopped = make(chan bool)
	stopFrameGetter = make(chan bool)
	if inited {
		fmt.Println("frame getter inited while already inited")
		return
	}
	inited = true
	go func() {
		for range stopFrameGetter {
			inited = false
			stopFrames = true
			return
		}
	}()
	frameChan := vbody.GetFrameChan()
	go func() {
		for frame := range frameChan {
			if stopFrames {
				stopFrames = false
				frameGetterStopped <- true
				return
			}
			mu.Lock()
			currentFrame = frame
			mu.Unlock()
		}
	}()

}
