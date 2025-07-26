package main

import (
	"bufio"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"strings"
	"time"
	"encoding/json"
	"github.com/kercre123/vector-gobot/pkg/vbody"
	"github.com/kercre123/vector-gobot/pkg/vscreen"
)

// program which will run in recovery partition

var CurrentList *List
var ScreenInited bool
var BodyInited bool
var MaxTM uint32
var MinTM uint32
var StopListening bool
var HangBody bool

type List struct {
	Info      string
	InfoColor color.Color
	Lines     []vscreen.Line
	// len and position start with 1
	Len       int
	Position  int
	ClickFunc []func()
	inited    bool
}

type OTA struct {
	Name string
	URL string
}

var availableOTAs []OTA

func LoadOTAConfig() error {
    data, err := os.ReadFile("/data/vic-menu/ota-list.json")
    if err != nil {
        return err
    }
    return json.Unmarshal(data, &availableOTAs)
}

func (c *List) MoveDown() {
	if c.Len == c.Position {
		c.Position = 1
	} else {
		c.Position = c.Position + 1
	}
	c.UpdateScreen()
}

func (c *List) MoveUp() {
	// i'm not sure how to determine direction from the encoders, so i am doing always down
	fmt.Println("up")
}

func (c *List) UpdateScreen() {
	var linesShow []vscreen.Line
	// if info, have list go to bottom
	// 7 lines fit comfortably on screen
	if c.Info != "" {
		newLine := vscreen.Line{
			Text:  c.Info,
			Color: c.InfoColor,
		}
		linesShow = append(linesShow, newLine)
		numOfSpaces := 7 - c.Len
		if numOfSpaces < 0 {
			panic("too many items in list" + fmt.Sprint(numOfSpaces))
		}
		for i := 1; i < numOfSpaces; i++ {
			newLine = vscreen.Line{
				Text:  " ",
				Color: c.InfoColor,
			}
			linesShow = append(linesShow, newLine)
		}
	}
	for i, line := range c.Lines {
		var newLine vscreen.Line
		if i == c.Position-1 {
			newLine.Text = "> " + line.Text
			newLine.Color = line.Color
		} else {
			newLine.Text = "  " + line.Text
			newLine.Color = line.Color
		}
		linesShow = append(linesShow, newLine)
	}
	scrnData := vscreen.CreateTextImageFromLines(linesShow)
	vscreen.SetScreen(scrnData)
}

func (c *List) Init() {
	c.Position = 1
	c.Len = len(c.Lines)
	if !BodyInited {
		vbody.InitSpine()
		InitFrameGetter()
		BodyInited = true
	}
	if !ScreenInited {
		vscreen.InitLCD()
		vscreen.BlackOut()
		ScreenInited = true
	}
	c.UpdateScreen()
	c.inited = true
}

func ListenToBody() {
	if !CurrentList.inited {
		fmt.Println("error: init list before listening dummy")
		os.Exit(1)
	}
	for {
		if StopListening {
			fmt.Println("not listening anymore")
			StopListening = false
			return
		}
		if !CurrentList.inited || HangBody {
			for {
				time.Sleep(time.Second / 5)
				if CurrentList.inited && !HangBody {
					break
				}
			}
		}
		frame := GetFrame()
		if frame.ButtonState {
			CurrentList.ClickFunc[CurrentList.Position-1]()
			time.Sleep(time.Second / 3)
		}
		for i, enc := range frame.Encoders {
			if i > 1 {
				// only read wheels
				break
			}
			if enc.DLT < -1 {
				stopTimer := false
				stopWatch := false
				go func() {
					timer := 0
					for {
						if StopListening {
							fmt.Println("not listening anymore")
							StopListening = false
							return
						}
						if stopTimer {
							break
						}
						if timer == 30 {
							CurrentList.MoveDown()
							stopWatch = true
							break
						}
						timer = timer + 1
						time.Sleep(time.Millisecond * 10)
					}
				}()
				for {
					if StopListening {
						fmt.Println("not listening anymore")
						StopListening = false
						return
					}
					frame = GetFrame()
					if stopWatch {
						break
					}
					if frame.Encoders[i].DLT == 0 {
						stopTimer = true
						break
					}
				}
			}
		}
		time.Sleep(time.Millisecond * 10)
	}
}

func StartAnki_Confirm() {
	c := *CurrentList
	CurrentList = Confirm_Create(StartAnki, c)
	CurrentList.Init()
}

func StartAnki() {
	scrnData := vscreen.CreateTextImage("To go back to the recovery menu, go to CCIS and select `BACK TO MENU`. Starting in 3 seconds...")
	vscreen.SetScreen(scrnData)
	time.Sleep(time.Second * 4)
	scrnData = vscreen.CreateTextImage("Stopping body...")
	vscreen.SetScreen(scrnData)
	CurrentList.inited = false
	time.Sleep(time.Second / 3)
	StopFrameGetter()
	vbody.StopSpine()
	scrnData = vscreen.CreateTextImage("Starting anki-robot...")
	vscreen.SetScreen(scrnData)
	vscreen.StopLCD()
	ScreenInited = false
	BodyInited = false
	time.Sleep(time.Second / 2)
	exec.Command("/bin/bash", "-c", "systemctl start anki-robot.target").Run()
	// watch logcat for clear user data screen
	cmd := exec.Command("logcat", "-T", "1")
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		//09-17 15:51:52.178  2040  2040 D vic-anim: [@FaceInfoScreenManager] FaceInfoScreenManager.SetScreen.EnteringScreen: (tc5185) : 5
		if strings.Contains(line, "FaceInfoScreenManager.SetScreen.EnteringScreen") {
			if strings.Contains(line, " : 5") {
				break
			}
		}
	}
	exec.Command("/bin/bash", "-c", "systemctl stop anki-robot.target").Run()
	time.Sleep(time.Second)
	CurrentList = Recovery_Create()
	CurrentList.Init()
}

func StartRescue_Confirm() {
	c := *CurrentList
	CurrentList = Confirm_Create(StartRescue, c)
	CurrentList.Init()
}

func StartRescue() {
	KillButtonDetect := false
	// rescue can crash, often
	HangBody = true
	scrnData := vscreen.CreateTextImage("vic-rescue will start in 3 seconds. Press the button anytime to return to the menu.")
	vscreen.SetScreen(scrnData)
	vscreen.StopLCD()
	ScreenInited = false
	time.Sleep(time.Second * 3)
	cmd := exec.Command("/bin/bash", "-c", "/anki/bin/vic-rescue")
	go func() {
		for {
			frame := GetFrame()
			if frame.ButtonState || KillButtonDetect {
				break
			}
			time.Sleep(time.Millisecond * 10)
		}
		fmt.Println("killing rescue")
		cmd.Process.Kill()
	}()
	cmd.Run()
	CurrentList = Recovery_Create()
	CurrentList.Init()
	time.Sleep(time.Second / 3)
	HangBody = false
}

func Reboot_Do() {
	exec.Command("/bin/bash", "-c", "bootctl f set_active a")
	scrnData := vscreen.CreateTextImage("Rebooting...")
	vscreen.SetScreen(scrnData)
	StopListening = true
	time.Sleep(time.Second / 2)
	vbody.StopSpine()
	vscreen.StopLCD()
	exec.Command("/bin/bash", "-c", "reboot").Run()
}

func Reboot_Create() *List {
	// "ARE YOU SURE?"
	var Reboot List

	Reboot.Info = "Reboot?"
	Reboot.InfoColor = color.RGBA{0, 255, 0, 255}
	Reboot.ClickFunc = []func(){Reboot_Do, func() {
		CurrentList = Recovery_Create()
		CurrentList.Init()
	}}

	Reboot.Lines = []vscreen.Line{
		{
			Text:  "Yes",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  "No",
			Color: color.RGBA{255, 255, 255, 255},
		},
	}

	return &Reboot
}

func ClearUserData_Do() {
	vscreen.SetScreen(vscreen.CreateTextImage("Clearing..."))
	exec.Command("/bin/bash", "-c", "blkdiscard -s /dev/block/bootdevice/by-name/userdata").Run()
	exec.Command("/bin/bash", "-c", "blkdiscard -s /dev/block/bootdevice/by-name/switchboard").Run()
	Reboot_Do()
}

func ClearUserData_Create() *List {
	// "ARE YOU SURE?"
	var Reboot List

	Reboot.Info = "Clear user data?"
	Reboot.InfoColor = color.RGBA{0, 255, 0, 255}
	Reboot.ClickFunc = []func(){ClearUserData_Do, func() {
		CurrentList = Recovery_Create()
		CurrentList.Init()
	}}

	Reboot.Lines = []vscreen.Line{
		{
			Text:  "Yes",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  "No",
			Color: color.RGBA{255, 255, 255, 255},
		},
	}

	return &Reboot
}

func InstallSelectedOTA(url string) {
	HangBody = true
	if ssid, _ := getNet(); ssid == "<not connected>" {
		scrnData := vscreen.CreateTextImage("The robot must first be connected to Wi-Fi.")
		vscreen.SetScreen(scrnData)
		time.Sleep(time.Second * 3)
		CurrentList = Recovery_Create()
		CurrentList.Init()
		HangBody = false
		return
	}
	err := StreamOTA(url)
	if err != nil {
		fmt.Println(err)
		if strings.Contains(err.Error(), "button") {
			CurrentList = Recovery_Create()
			CurrentList.Init()
			time.Sleep(time.Second / 3)
			HangBody = false
		} else {
			scrnData := vscreen.CreateTextImage("Error downloading OTA: " + err.Error())
			vscreen.SetScreen(scrnData)
			time.Sleep(time.Second * 3)
			CurrentList = Recovery_Create()
			CurrentList.Init()
			HangBody = false
		}
	} else {
		HangBody = false
		CurrentList = Reboot_Create()
		CurrentList.Init()
		time.Sleep(time.Second / 3)
	}
}

func getNet() (ssid string, ip string) {
	out, _ := exec.Command("/bin/bash", "-c", "iwgetid").Output()
	iwcmd := strings.TrimSpace(string(out))
	if !strings.Contains(iwcmd, "ESSID") {
		ssid = "<not connected>"
		ip = "<not connected>"
	} else {
		ssid = strings.Replace(strings.TrimSpace(strings.Split(iwcmd, "ESSID:")[1]), `"`, "", -1)
		out, _ = exec.Command("/bin/bash", "-c", `/sbin/ifconfig wlan0 | grep 'inet addr' | cut -d: -f2 | awk '{print $1}'`).Output()
		ip = strings.TrimSpace(string(out))
	}
	return ssid, ip
}

func DetectButtonPress() {
	// for functions which show on screen, but aren't lists. hangs ListenToBody, returns when button is presed
	for {
		frame := GetFrame()
		if frame.ButtonState {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}

}

func PrintNetworkInfo() {
	c := *CurrentList
	ssid, ip := getNet()
	lines := []string{"SSID: " + ssid, "IP: " + ip, " ", " ", " ", "> Back"}
	scrnData := vscreen.CreateTextImageFromSlice(lines)
	vscreen.SetScreen(scrnData)
	HangBody = true
	time.Sleep(time.Second / 3)
	DetectButtonPress()
	CurrentList = &c
	CurrentList.Init()
	time.Sleep(time.Second / 3)
	HangBody = false
}

func Rebooter() {
	CurrentList = Reboot_Create()
	CurrentList.Init()
}

func ClearUserData() {
	CurrentList = ClearUserData_Create()
	CurrentList.Init()
}

func Recovery_Create() *List {
	var Test List

	Test.Info = "Vec2.0 Recovery Menu"
	Test.InfoColor = color.RGBA{0, 255, 0, 255}

	Test.ClickFunc = []func(){
       StartAnki_Confirm,
       ClearUserData,
       PrintNetworkInfo,
       Rebooter,
       func() {
           CurrentList = ShowOTAList_Create()
          CurrentList.Init()
       },
   	} 

	Test.Lines = []vscreen.Line{
		{
			Text:  "Start anki-robot",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  "Clear user data",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  "Print network info",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  "Reboot to system_a",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  "Install latest OTA",
			Color: color.RGBA{255, 255, 255, 255},
		},
	}

	return &Test
}

func Confirm_Create(do func(), origList List) *List {
	// "ARE YOU SURE?"
	var Test List

	Test.Info = "Are you sure?"
	Test.InfoColor = color.RGBA{0, 255, 0, 255}
	Test.ClickFunc = []func(){do, func() {
		CurrentList = &origList
		CurrentList.Init()
	}}

	Test.Lines = []vscreen.Line{
		{
			Text:  "Yes",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  "No",
			Color: color.RGBA{255, 255, 255, 255},
		},
	}

	return &Test
}

func ShowOTAListPage(page int) *List {
    const perPage = 3
    total := len(availableOTAs)
    start := page * perPage
    end := start + perPage
    if end > total {
        end = total
    }

    var l List
    l.Info = fmt.Sprintf("OTAs %d-%d of %d", start+1, end, total)
    l.InfoColor = color.RGBA{0, 255, 0, 255}

    // show this page's OTAs
    for _, ota := range availableOTAs[start:end] {
        url := ota.URL
        l.ClickFunc = append(l.ClickFunc, func() {
            CurrentList = Confirm_Create(func() {
                InstallSelectedOTA(url)
            }, *CurrentList)
            CurrentList.Init()
        })
        l.Lines = append(l.Lines, vscreen.Line{
            Text:  ota.Name,
            Color: color.RGBA{255, 255, 255, 255},
        })
    }

    // Prev button?
    if page > 0 {
        prev := page - 1
        l.ClickFunc = append(l.ClickFunc, func() {
            CurrentList = ShowOTAListPage(prev)
            CurrentList.Init()
        })
        l.Lines = append(l.Lines, vscreen.Line{
            Text:  "< Prev",
            Color: color.RGBA{255, 255, 255, 255},
        })
    }

    // Next button?
    if end < total {
        next := page + 1
        l.ClickFunc = append(l.ClickFunc, func() {
            CurrentList = ShowOTAListPage(next)
            CurrentList.Init()
        })
        l.Lines = append(l.Lines, vscreen.Line{
            Text:  "Next >",
            Color: color.RGBA{255, 255, 255, 255},
        })
    }

    // Back to main recovery menu
    l.ClickFunc = append(l.ClickFunc, func() {
        CurrentList = Recovery_Create()
        CurrentList.Init()
    })
    l.Lines = append(l.Lines, vscreen.Line{
        Text:  "Exit",
        Color: color.RGBA{255, 255, 255, 255},
    })

    return &l
}

// entry point remains page 0
func ShowOTAList_Create() *List {
	if availableOTAs == nil {
        if err := LoadOTAConfig(); err != nil {
            // show an error and return to recovery
            var errList List
            errList.Info = "Error loading OTA list"
            errList.InfoColor = color.RGBA{255, 0, 0, 255}
            errList.Lines = []vscreen.Line{
                {Text: err.Error(), Color: errList.InfoColor},
                {Text: "Back",       Color: color.RGBA{255, 255, 255, 255}},
            }
            errList.ClickFunc = []func(){
                func() {
                    CurrentList = Recovery_Create()
                    CurrentList.Init()
                },
            }
            return &errList
        }
    }
    return ShowOTAListPage(0)
}

func TestIfBodyWorking() {
	// if body isn't working, start anki processes
	err := vbody.InitSpine()
	if err != nil {
		vscreen.InitLCD()
		vscreen.BlackOut()
		data := vscreen.CreateTextImage("Error! Not able to communicate with the body. Starting Anki processes...")
		vscreen.SetScreen(data)
		vbody.StopSpine()
		vscreen.StopLCD()
		exec.Command("/bin/bash", "-c", "systemctl start anki-robot.target").Run()
		os.Exit(0)
	} else {
		BodyInited = true
	}
}

func main() {
	TestIfBodyWorking()
	vbody.SetLEDs(vbody.LED_OFF, vbody.LED_OFF, vbody.LED_OFF)
	vbody.SetMotors(0, 0, -100, -100)
	time.Sleep(time.Second * 2)
	vbody.SetMotors(0, 0, 0, 0)
	time.Sleep(time.Second)
	vbody.SetMotors(0, 0, 0, 150)
	go func() {
		time.Sleep(time.Second * 2)
		vbody.SetMotors(0, 0, 0, 0)
	}()
	CurrentList = Recovery_Create()
	CurrentList.Init()
	fmt.Println("started")
	InitFrameGetter()
	ListenToBody()
}
