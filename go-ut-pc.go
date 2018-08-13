// You can edit this code!
// Click here and start typing.
package main

import "fmt"
import "encoding/hex"
import "github.com/jacobsa/go-serial/serial"
import "time"
import "os"
import "bufio"
import "strings"
import "strconv"
import "math"
import "encoding/binary"
import "github.com/fatih/color"

const (
	head = 170
	tail = 187

	ping = 116
	name = 65
	datasize = 67
	qeury = 81
	disconnect = 66
)

var toEncoder chan []byte
var toDecoder chan []byte
var toSender  chan []byte

var current_file byte

func xor(b []byte) []byte {
	
	b = append(b,0)
	LEN := len(b)

	for i := 0; i < LEN-1; i++ {
		b[LEN-1] = b[i]^b[LEN-1]
 	}
	return b
}

func Float32frombytes(bytes []byte) float32 {
    bits := binary.LittleEndian.Uint32(bytes)
    float := math.Float32frombits(bits)
    return float
}

func Decoder() {
	for{
		buf := <-toDecoder
		buflen := len(buf)
		if 1 < buflen {
			if buf[1] == 67 {
				if 2 < buflen {
					if buf[2] == 2 {
						buf_xor := []byte{81, 2, 0, current_file, 0}
						buf_xor = xor(buf_xor)
						//fmt.Println(buf_xor) 
						buf_send := make([]byte,1)
						buf_send[0] = 170
						buf_send = append(buf_send[:],buf_xor[:]...)
						buf_send = append(buf_send[:],[]byte{187}...)
						//fmt.Println(buf_send)
						toSender <- buf_send
						//fmt.Println("current_file: ",current_file)
					}
				}
			} else if buf[1] == 81 {
				//fmt.Println("buflen",buflen)
				datalen := int(buf[2])
				datalen = datalen/4
 				for i := 0; i < datalen; i++ {
					buf_data := buf[4+4*i:8+4*i]
					float32_data := Float32frombytes(buf_data)
					//fmt.Printf("%.2f\n",float32_data)
					color.Yellow("%.2f\n",float32_data)
				}				
			}
		}
	}
}

func main() {
	color.Yellow("HUMMING SYSTEM ULTRASONIC THICKNESS GAUGE\n")
	color.Green("PTTEP & CiiMAV Lab")
	toEncoder = make(chan []byte, 100)
	toDecoder = make(chan []byte, 100)
	toSender  = make(chan []byte, 100)

	current_file = 0

	go Decoder()
	/* define option */
	options := serial.OpenOptions{
		PortName: "COM13",
		BaudRate: 9600,
		DataBits: 8,
    	StopBits: 1,
    	InterCharacterTimeout: 100,
    	MinimumReadSize: 4,
	}

	/* make connection */
	port, err := serial.Open(options)
	if err != nil {
		fmt.Println("serial.Open: %v", err)
		/*  open port */
		for{
			/* try open every 1 second */
			port, err = serial.Open(options)
			if err != nil {
				fmt.Println("serial.Open: %v", err)
			} else {
				break
			}
			time.Sleep(1000 * time.Millisecond)
		}
	} 	
	defer port.Close()

	/* reading */
	go func () {
		for{
			//reading
    		var data []byte
			buf := make([]byte,1)
			n, err := port.Read(buf)
			if err != nil {
				fmt.Println("Error reading from TELE: ", err)
			} else{
				buf = buf[:n]
				//fmt.Println("UT Rx: ", hex.EncodeToString(buf))
			}

			if hex.EncodeToString(buf) == "32" {
			buf2 := make([]byte,1)
			n, err := port.Read(buf2)
			if err != nil {
				fmt.Println("Error reading from TELE: ", err)
			} else{
				buf2 = buf2[:n]
				//fmt.Println("UT Rx: ", hex.EncodeToString(buf2))
				data = append(data[:],buf[:]...)
			}

			if hex.EncodeToString(buf2) == "32" {
				buf3 := make([]byte,1)
				n, err := port.Read(buf3)
				if err != nil {
					fmt.Println("Error reading from TELE: ", err)
				} else{
					buf3 = buf3[:n]
					//fmt.Println("UT Rx: ", hex.EncodeToString(buf3))
					data = append(data[:],buf2[:]...)
				}

				if hex.EncodeToString(buf3) == "21" {
					data = append(data[:],buf3[:]...)
					color.Red("connected")
				}
			}
			//fmt.Println("toDecoder: ",data,"\n")
			toDecoder <- data
			} else if hex.EncodeToString(buf) == "aa" {
			data = append(data[:],buf[:]...)
			for{
				buf := make([]byte,1)
				n, err := port.Read(buf)
				if err != nil {
					fmt.Println("Error reading from TELE: ", err)
				} else{
					buf = buf[:n]
					//fmt.Println("UT Rx: ", hex.EncodeToString(buf))
				}

				if hex.EncodeToString(buf) == "bb" {
					data = append(data[:],buf[:]...)
					break
				} else {
					data = append(data[:],buf[:]...)
				}
			}

			//fmt.Println("toDecoder: ",data,"\n")
			toDecoder <- data
			}
		}
	}()

	/* sending */
	go func (){
		for{
			buf := <- toSender
			port.Write(buf)
		}
	}()

	fmt.Println("Type: connect, disconnect, file")
	scanner := bufio.NewScanner(os.Stdin)
	var words []string 
	for scanner.Scan() {
    	//fmt.Println(scanner.Text())
    	words = strings.Fields(scanner.Text())
		//fmt.Println(words, len(words)) 

		if words[0] == "connect" {
			buf := make([]byte,2)
			buf[0] = ping
			buf[1] = 10
			port.Write(buf)

		} else if words[0] == "file" && len(words) == 2 {
			file , err := strconv.Atoi(words[1])
			if err != nil {
				fmt.Println("Invalid file name")
			}

			if file >= 0 && file <= 99 {
				//fmt.Println("request file ",file)
				// read file 1 
				// [170 67 3 0 49 1 0 112 187]
				buf_xor := []byte{67,3,0,49,byte(file),0}
				buf_xor = xor(buf_xor)
				//fmt.Println(buf_xor)
				buf := make([]byte,1)
				buf[0] = 170
				buf = append(buf[:],buf_xor[:]...)
				buf = append(buf[:],[]byte{187}...)					
				//fmt.Println(buf)
				toSender <- buf
				current_file = byte(file)
			}					
		}
	}

	

	fmt.Println("Hello, 世界")
	for{
		time.Sleep(10000 * time.Millisecond)
	}
}