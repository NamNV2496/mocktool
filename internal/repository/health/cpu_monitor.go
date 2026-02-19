package health

// import (
// 	"time"

// 	"github.com/shirou/gopsutil/v3/cpu"
// )

// func StartCPUMonitor(threshold float64) {
// 	go func() {
// 		for {
// 			percent, _ := cpu.Percent(1*time.Second, false)
// 			if len(percent) > 0 && percent[0] > threshold {
// 				SetOverloaded(true)
// 			}
// 			time.Sleep(2 * time.Second)
// 		}
// 	}()
// }
