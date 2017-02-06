package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"
)

const (
	//破解成功的标识
	SuccessFlag = "Everything is Ok"
	//7z工具的路径
	Exe7zPath = "7-Zip\\7z.exe"
	//需要破解的7z文件
	Path = "C:\\1.7z"
	//破解从第几次
	TryFromTimes = 0
	//破解到第几次
	TryToTimes = 99
	//并发线程数
	ThreadCount = 8
)

//密码库，使用密码库可大大加快破解速度
var Keys = []string{"1", "23", "0", "-", "."}

//如果不设置密码库将使用默认字符集 下面可以设置需要使用哪些字符
//这样的暴力破解9位纯数字的密码，如果 80次每秒 也需要 621 天才能破解完成
//因此密码库是非常关键的。
//var Keys = []string{}
const (
	DefaultKey_09    = true
	DefaultKey_az    = false
	DefaultKey_AZ    = false
	DefaultKey_Other = "!@#$%^&*."
)

func main() {
	cracker := new(Cracker)
	cracker.Init(Keys)

	pass := int64(0)
	if ThreadCount <= 1 {
		pass = cracker.SingleThreadDeal(Path, TryFromTimes, TryToTimes)
	} else {
		pass = cracker.MultiThreadDeal(Path, ThreadCount, TryFromTimes, TryToTimes)
	}

	fmt.Println(cracker.Property())

	if pass != -1 {
		fmt.Println(pass, cracker.ToString(pass))
	} else {
		fmt.Println("cracker failed")
	}

}

type Cracker struct {
	keys []string
	stop int32

	start int64
	end   int64
	times int64
}

func (cracker *Cracker) ToString(passIndex int64) string {
	r := bytes.NewBuffer(nil)
	k := cracker.keys
	l := int64(len(k))
	c := passIndex
	r.WriteString(k[c%l])
	for c > l {
		c /= l
		r.WriteString(k[c%l])
	}
	return r.String()
}
func (cracker *Cracker) Property() (r string) {
	end := atomic.LoadInt64(&cracker.end)
	start := atomic.LoadInt64(&cracker.start)
	times := atomic.LoadInt64(&cracker.times)

	second := float64(end-start) / 1000000000
	r += fmt.Sprintln("try", times, "times")
	r += fmt.Sprintln("pass", second, "second")
	r += fmt.Sprintln(float64(times)/second, "times/second")
	return r
}

func (cracker *Cracker) Init(keys []string) {
	cracker.stop = 0
	if keys == nil || len(keys) == 0 {
		if DefaultKey_09 {
			for i := 48; i <= 57; i++ {
				cracker.keys = append(cracker.keys, string(i))
			}
		}
		if DefaultKey_az {
			for i := 97; i <= 122; i++ {
				cracker.keys = append(cracker.keys, string(i))
			}
		}
		if DefaultKey_AZ {
			for i := 65; i <= 90; i++ {
				cracker.keys = append(cracker.keys, string(i))
			}
		}
		for _, v := range DefaultKey_Other {
			cracker.keys = append(cracker.keys, string(v))
		}

	} else {
		cracker.keys = keys
	}
}

func (cracker *Cracker) SingleThreadDeal(path string, min int64, max int64) int64 {
	atomic.StoreInt64(&cracker.start, time.Now().UnixNano())
	defer func() {
		atomic.StoreInt64(&cracker.end, time.Now().UnixNano())
	}()

	return cracker.deal(path, min, max)
}

func (cracker *Cracker) MultiThreadDeal(path string, maxThread int64, min int64, max int64) int64 {
	atomic.StoreInt64(&cracker.start, time.Now().UnixNano())
	defer func() {
		atomic.StoreInt64(&cracker.end, time.Now().UnixNano())
	}()

	stop := make(chan int64, maxThread)
	length := max - min + maxThread
	eleLength := length / maxThread
	for i := int64(0); i < maxThread; i++ {
		i1 := i
		go func() {
			ret := cracker.deal(path, min+eleLength*i1, min+eleLength*(i1+1))
			if ret != -1 {
				atomic.StoreInt32(&cracker.stop, 1)
			}
			stop <- ret
		}()
	}
	result := int64(-1)
	for i := int64(0); i < maxThread; i++ {
		temp := <-stop
		if temp == -1 {
			continue
		} else {
			result = temp
		}
	}
	return result
}

func (cracker *Cracker) deal(path string, min int64, max int64) int64 {
	in := bytes.NewBuffer(nil)
	var out bytes.Buffer
	k := cracker.keys
	l := int64(len(k))
	c := int64(0)
	s := Exe7zPath + " x " + path + " -aos -p"
	for i := min; i <= max; i++ {
		if atomic.LoadInt32(&cracker.stop) == 1 {
			return -1
		}
		cmd := exec.Command("cmd")
		cmd.Stdin = in    //绑定输入
		cmd.Stdout = &out //绑定输出
		in.WriteString(s)
		c = i
		in.WriteString(k[c%l])
		for c > l {
			c /= l
			in.WriteString(k[c%l])
		}
		in.WriteString("\n")

		//写入你的命令，可以有多行，"\n"表示回车
		err := cmd.Start()
		if err != nil {
			fmt.Println("ERR", err)
			return -1
		}

		err = cmd.Wait()
		if err != nil {
			fmt.Println("ERR", err)
			return -1
		}
		if strings.Contains(out.String(), SuccessFlag) {
			return i
		}
		in.Reset()
		out.Reset()
		atomic.AddInt64(&cracker.times, 1)
	}
	return -1
}
