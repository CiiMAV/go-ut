package main

import (
"fmt"
"time"
"github.com/jacobsa/go-serial/serial"
"encoding/hex"
"encoding/binary"
"math"
"bufio"
"os"
"strings"
"strconv"
"github.com/fatih/color"
)

var toSender chan []byte
var toDecoder chan []byte
var toCommander chan []string

var filesize_map map[int]int
var filedata_map map[int][]float32

func xor(b []byte) []byte {	
	b = append(b,0)
	LEN := len(b)
	for i := 0; i < LEN-1; i++ {
		b[LEN-1] = b[i]^b[LEN-1]
 	}
	return b
}

func Float32frombytes(bytes []byte) float32{
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}

func main() {
	toSender = make(chan []byte, 100)
	toDecoder = make(chan []byte, 100)
	toCommander = make(chan []string, 100)

	filesize_map = make(map[int]int)
	filedata_map = make(map[int][]float32)

	current_file_number := 0
	/* define option */
	// /dev/ttyUSB0
	// COMx
	options := serial.OpenOptions{
		PortName: "/dev/ttyUSB0",
		BaudRate: 9600,
		DataBits: 8,
   		StopBits: 1,
   		MinimumReadSize: 4,
	}
	/* make connection */
	port, err := serial.Open(options)
	for{
		/* try open every 1 second */
		port, err = serial.Open(options)
		if err != nil {
			fmt.Printf("serial.Open: %v\n", err)
		} else {
			break
		}
		time.Sleep(1000 * time.Millisecond)
	}
	defer port.Close()
	fmt.Println("initialized..")
	fmt.Println("telemetry connected")
	defer fmt.Println("disconnected and end routine")

	err_reading := 0
	ReadOneByte := func () []byte {
		buf := make([]byte, 1)
		n, err := port.Read(buf)
		if err != nil {
			if err_reading == 0 {
				fmt.Printf("Error reading from telemetry")
				err_reading++
			} else if err_reading % 1000 == 0 {
				fmt.Printf(".")
			}
		} else {
			buf = buf[:n]
			err_reading = 0
		}
		return buf
	}

	/* reading */
	go func () {
		loop:
		for{
			var data []byte
			buf := ReadOneByte()
			switch hex.EncodeToString(buf){
			case "32":
				fmt.Println("connected")
				break
			case "L":
				fmt.Println("telemetry packet loss")
				break
			case "aa":
				//fmt.Println("GET aa")
				data = append(data[:], buf[:]...)
				start := time.Now()
				for{
					buf = ReadOneByte()
					if hex.EncodeToString(buf) == "bb"{
						data = append(data[:], buf[:]...)
						ch := data[1:len(data)-1]
						ch = xor(ch)
						if ch[len(ch)-2] == data[len(data)-2] {
							//fmt.Println("XOR OK")
							toDecoder <- data
						} else {
							fmt.Println("packet loss")
							continue
						}						
						break
					} else {
						data = append(data[:], buf[:]...)
					}
					if time.Since(start) >= time.Second {
						fmt.Println("time out")
						goto loop
					}
				}
				break
			}
		}
	}()
	/* sender */
	go func () {
		for{
			buf := <- toSender
			port.Write(buf)
		}
	}()

	/* decoder */
	go func () {
		for{
			select{
			case data:= <-toDecoder:
				//fmt.Println(data)
				switch data[1]{
					case 65:
						//fmt.Println("GET 65")
						filesize_map[int(data[2])] = int(data[3])
						fmt.Println(filesize_map[int(data[2])])
						break
					case 66:
						//fmt.Println("GET 66")
						file_name := int(data[2])
						val := Float32frombytes(data[3:7])						
						fmt.Println("file: ", file_name, "  value: ", val)
						break
					case 67:
						//fmt.Println("GET 67")
						file_name := int(data[2])
						filesize := int(data[3])
						// clean map
						filedata_map[file_name] = []float32{}
						for i := 0; i < filesize; i++ {
							val := Float32frombytes(data[4+4*i:8+4*i])
							filedata_map[file_name] = append(filedata_map[file_name][:], []float32{val}...)
							fmt.Println("file: ", file_name, " number: ", i+1,"  value: ", val)
							//fmt.Println(filedata_map[file_name])
						}
						break
					case 68:
						//subscript
						//fmt.Println("GET 68")
						file_name := int(data[2])
						number := int(data[3])
						val := Float32frombytes(data[4:8])						
						fmt.Println("subscript file: ", file_name, "  ", number+1 ,"  value: ", val)
						break
				}
			}
		}
	}()
	/* commander */
	go func () {
		for{
			select{
			case words := <- toCommander:
				//fmt.Println("send: ", words)
				if words[0] == "connect"{
					buf := make([]byte,1)
					buf[0] = 50
					toSender <- buf
				} else if words[0] == "filesize" && len(words) == 2{
					file, err := strconv.Atoi(words[1])
					if err != nil {
						fmt.Println("Invalid file name")
						continue
					}
					if file >= 0 && file <= 99 {
						buf := []byte{65,byte(file)}
						buf = xor(buf)
						buf = append([]byte{170}, buf[:]...) //aa
						buf = append(buf[:],[]byte{187}...)
						toSender <- buf
						current_file_number = file
					}
				} else if words[0] == "filedata" && len(words) == 3{
					file, err := strconv.Atoi(words[1])
					if err != nil {
						fmt.Println("Invalid file name")
						continue
					}
					number, err := strconv.Atoi(words[2])
					if err != nil {
						fmt.Println("Invalid number")
						continue
					}
					if file >= 0 && file <= 99 {
						buf := []byte{66,byte(file),byte(number)}
						buf = xor(buf)
						buf = append([]byte{170}, buf[:]...) //aa
						buf = append(buf[:],[]byte{187}...)
						toSender <- buf
						current_file_number = file
					}
				} else if words[0] == "filedataall" && len(words) == 2{
					file, err := strconv.Atoi(words[1])
					if err != nil {
						fmt.Println("Invalid file name")
						continue
					}
					if file >= 0 && file <= 99 {
						buf := []byte{67,byte(file)}
						buf = xor(buf)
						buf = append([]byte{170}, buf[:]...) //aa
						buf = append(buf[:],[]byte{187}...)
						toSender <- buf
						current_file_number = file
					}
				} else if words[0] == "subscribe" && len(words) == 2{
					file, err := strconv.Atoi(words[1])
					if err != nil {
						fmt.Println("Invalid file name")
						continue
					}
					if file >= 0 && file <= 99 {
						buf := []byte{68,byte(file)}
						buf = xor(buf)
						buf = append([]byte{170}, buf[:]...) //aa
						buf = append(buf[:],[]byte{187}...)
						toSender <- buf
						current_file_number = file
					}
				} else if words[0] == "stop" && len(words) == 2{
					file, err := strconv.Atoi(words[1])
					if err != nil {
						fmt.Println("Invalid file name")
						continue
					}
					buf := []byte{69,byte(file)}
					buf = xor(buf)
					buf = append([]byte{170}, buf[:]...)
					buf = append(buf[:], []byte{187}...)
					toSender <- buf
					//fmt.Println("send stop")
				} else if words[0] == "savemap"{
					//save to text file
					fmt.Println(filedata_map)
				}
			}
		}
	}()

	color.Yellow("HUMMING SYSTEM ULTRASONIC THICKNESS GAUGE INSPECTION")
	color.Green("powered by PTTEP & CiiMAV Lab")
	fmt.Println("Type: connect, filesize, filedata, filedataall, subscript, stop")
	fmt.Println("connect")
	fmt.Println("filesize n    , n = file nummber")
	fmt.Println("filedata n m  , n = file nummber, m = data number")
	fmt.Println("filedataall n , n = file nummber")
	fmt.Println("subscribe n   , n = file number")
	fmt.Println("stop n        , n = file number which is subscribing")
	reader := bufio.NewReader(os.Stdin)
	for{
		text, _  := reader.ReadString('\n')
		words := strings.Fields(text)
		if len(words) < 1 {
			continue
		}
		toCommander <- words
	}
}