package main

import (
	"bufio"
	"fmt"
	"image/color"
	"math"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/kercre123/vector-gobot/pkg/vbody"
	"github.com/kercre123/vector-gobot/pkg/vscreen"
)

var BootProgressKill bool
var SysProgressKill bool

type Progress struct {
	Size     int64
	Total    int64
	Progress int64
}

func ExecCmds(commands []string) {
	for _, cmd := range commands {
		exec.Command("/bin/bash", "-c", cmd).Run()
	}
}

func SetCPUToPerf() {
	cmds := []string{
		"echo 1267200 > /sys/devices/system/cpu/cpu0/cpufreq/scaling_max_freq",
		"echo performance > /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor",
		"echo disabled > /sys/kernel/debug/msm_otg/bus_voting",
		"echo 0 > /sys/kernel/debug/msm-bus-dbg/shell-client/update_request",
		"echo 1 > /sys/kernel/debug/msm-bus-dbg/shell-client/mas",
		"echo 512 > /sys/kernel/debug/msm-bus-dbg/shell-client/slv",
		"echo 0 > /sys/kernel/debug/msm-bus-dbg/shell-client/ab",
		"echo active clk2 0 1 max 800000 > /sys/kernel/debug/rpm_send_msg/message",
		"echo 1 > /sys/kernel/debug/msm-bus-dbg/shell-client/update_request",
	}
	ExecCmds(cmds)
}

// Helper function to read an integer from a file
func readIntFromFile(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strconv.ParseInt(scanner.Text(), 10, 64)
	}

	return 0, scanner.Err()
}

// Function to update progress display on screen
func updateProgressScreen(progress float64) {
	linesShow := []vscreen.Line{
		{
			Text:  "Installing OTA...",
			Color: color.RGBA{0, 255, 0, 255},
		},
		{
			Text:  " ",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  fmt.Sprintf("Progress: %.1f%%", progress),
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  " ",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  " ",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  " ",
			Color: color.RGBA{255, 255, 255, 255},
		},
		{
			Text:  "> Cancel",
			Color: color.RGBA{255, 255, 255, 255},
		},
	}

	scrnData := vscreen.CreateTextImageFromLines(linesShow)
	vscreen.SetScreen(scrnData)
}

// Monitor progress by reading files in /run/update-engine
func monitorProgress(stopChan chan bool) {
	progressPath := "/run/update-engine/progress"
	expectedSizePath := "/run/update-engine/expected-size"

	for {
		select {
		case <-stopChan:
			BootProgressKill = true
			return
		default:
			// Read progress and expected size
			progressSize, err := readIntFromFile(progressPath)
			if err != nil {
				fmt.Println("Error reading progress:", err)
				updateProgressScreen(0.0)
			}

			expectedSize, err := readIntFromFile(expectedSizePath)
			if err != nil {
				fmt.Println("Error reading expected size:", err)
				updateProgressScreen(0.0)
			}

			if _, err := os.ReadFile("/run/update-engine/done"); err == nil {
				return
			}

			// Calculate and round progress
			if expectedSize > 0 {
				progressPercentage := (float64(progressSize) / float64(expectedSize)) * 100
				progressPercentage = math.Round(progressPercentage*10) / 10

				// Update progress on screen
				updateProgressScreen(progressPercentage)
			}

			time.Sleep(time.Second / 3)
		}
	}
}

func StreamOTA(url string) error {
	os.Remove("/run/update-engine/done")
	stopChan := make(chan bool, 1)

	// Run the update process in a goroutine
	updateCmd := exec.Command("/update", url)
	err := updateCmd.Start()
	if err != nil {
		return fmt.Errorf("Error starting update process: %w", err)
	}

	// Button listener for cancel
	go func() {
		time.Sleep(time.Second*2);
		buttonChan := vbody.GetButtonChan()
		for button := range buttonChan {
			if !button {
				stopChan <- true
				updateCmd.Process.Kill()
				break
			}
		}
	}()

	// Monitor progress
	monitorProgress(stopChan)

	// Wait for update command to complete
	err = updateCmd.Wait()
	if BootProgressKill {
		return fmt.Errorf("Operation stopped: button pressed")
	}
	if err != nil {
		return fmt.Errorf("Update process failed: %w", err)
	}

	return nil
}
