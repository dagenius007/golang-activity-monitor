//go:build darwin
// +build darwin

package processes

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	// . "binalyze-test/types"

	"golang.org/x/sys/unix"
)

func calculateTotalTime() {
}

func statInfoFromPs(pid int32) (any, error) {
	args := "-o user=,pcpu=,rss= -p"

	stdout, _ := exec.Command("ps", args, strconv.Itoa(int(pid))).Output()

	fmt.Println("stdout", strings.Fields(string(stdout)))

	return nil, nil
}

func getPids() ([]int32, error) {
	pids, err := unix.SysctlKinfoProcSlice("kern.proc.all")

	pidSlice := make([]int32, 0)

	for _, pid := range pids {
		// myString := string(pid.Eproc.Wmesg[:])
		statInfoFromPs(pid.Proc.P_pid)
		pidSlice = append(pidSlice, pid.Proc.P_pid)
	}

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return pidSlice, nil
}

// 73363
func formatLine(line string) []string {
	slice := strings.Fields(line)
	array := []string{}

	i := 0
	for i <= 11 {
		if len(slice) == 0 {
			continue
		} else {
			if len(array) == 10 {

				_string := strings.Join(slice[i:], " ")

				array = append(array, _string)

				break
			} else {
				array = append(array, slice[i])
			}
			i++
		}
	}

	return array
}

func getProcessList() ([]any, error) {
	// fmt.Sprintf()
	pids, err := getPids()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("pids", pids)
	cmd := exec.Command("ps", "aux")
	out, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	err = cmd.Start()

	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(out)

	processLists := make([][]string, 0)

	for scanner.Scan() {
		line := scanner.Text()

		processLists = append(processLists, formatLine(line))

	}

	processes := make([]any, 0)

	// for i, processList := range processLists {

	// 	if i == 0 {
	// 		continue
	// 	}

	// 	process := Process{}

	// 	for i, _process := range processList {
	// 		if i == 0 {
	// 			process.User = _process
	// 		}
	// 		if i == 1 {
	// 			_pid, _ := strconv.Atoi(_process)
	// 			process.PID = int32(_pid)
	// 		}
	// 		if i == 2 {
	// 			cpu_usage, _ := strconv.ParseFloat(_process, 32)
	// 			process.CpuUsage = float32(cpu_usage)
	// 		}
	// 		if i == 3 {
	// 			memory_percentage_usage, _ := strconv.ParseFloat(_process, 32)
	// 			process.MemoryPercentageUsage = float32(memory_percentage_usage)
	// 		}
	// 		if i == 4 {
	// 			virtual_memory_size, _ := strconv.Atoi(_process)
	// 			process.VirtualMemorySize = int64(virtual_memory_size)
	// 		}
	// 		if i == 5 {
	// 			resident_memory_size, _ := strconv.Atoi(_process)
	// 			process.ResidentMemorySize = int64(resident_memory_size)
	// 		}
	// 		if i == 6 {
	// 			process.Tty = _process
	// 		}
	// 		if i == 7 {
	// 			process.State = _process
	// 		}
	// 		if i == 8 {
	// 			process.TotalTime = _process
	// 		}
	// 		if i == 9 {
	// 			process.Command = _process
	// 		}
	// 	}

	// 	processes = append(processes, process)
	// }

	return processes, nil
}
