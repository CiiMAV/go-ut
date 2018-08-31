// You can edit this code!
// Click here and start typing.
package main

import "log"
import "fmt"
import "encoding/hex"
import "github.com/jacobsa/go-serial/serial"
import "time"
import "os"
import "os/signal"
import "bufio"
import "strings"
import "strconv"
import "math"
import "encoding/binary"
import "github.com/fatih/color"
//import "encoding/csv"

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

var next_input chan bool

var connect_chan chan bool

var current_file byte

func checkError(message string, err error) {
    if err != nil {
        log.Fatal(message, err)
    }
}

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
				//fmt.Println("buf:",buf)
				buf_ch := buf[1:buflen-2]
				buf_ch = xor(buf_ch)
				//fmt.Println("buf_ch:",buf_ch)
				if buf_ch[len(buf_ch)-1] == buf[len(buf)-2] {
					fmt.Println("XOR check ok")
				} else {
					fmt.Println("Package loss, try again ...")
					next_input <- true
					continue
				}
				
				datalen := int(buf[2])
				datalen = datalen/4

				file_name := "log-" + time.Now().Format("20060102") + "-file-" + strconv.Itoa(int(current_file)) + ".txt"
				file, err := os.Create(file_name)
    			checkError("Cannot create file", err)
    			
 				for i := 0; i < datalen; i++ {
					buf_data := buf[4+4*i:8+4*i]
					float32_data := Float32frombytes(buf_data)
					//fmt.Printf("%.2f\n",float32_data)
					//fmt.Println(i%2)
					if i%3 == 0 || i%3 == 1 {
						//color.Yellow("%2d: %6.2f\t",i,float32_data)
						fmt.Fprintf(color.Output, "%s", color.YellowString("%2d: %6.2f\t\t",i+1,float32_data))
						_,err := file.WriteString(fmt.Sprintf("%2d: %6.2f\r\n",i+1,float32_data))					
						checkError("Cannot write to file", err)
					} else {
						//color.Yellow("%2d: %6.2f",i,float32_data)
						fmt.Fprintf(color.Output, "%s", color.YellowString("%2d: %6.2f\n",i+1,float32_data))
						_,err := file.WriteString(fmt.Sprintf("%2d: %6.2f\r\n",i+1,float32_data))				
						checkError("Cannot write to file", err)
					}					
				}				
				//fmt.Println()
				fmt.Println()
				next_input <- true	
				file.Close()			
			}
		}
	}
}

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	color.Yellow("HUMMING SYSTEM ULTRASONIC THICKNESS GAUGE\n")
	color.Green("PTTEP & CiiMAV Lab\n")
	//color.Green("Enter UT port (windows: COMx, linux: /dev/ttyUSBx) :")
	fmt.Fprintf(color.Output, "%s", color.YellowString("Enter UT port (windows: COMx, linux: /dev/ttyUSBx) : "))
	reader_utport := bufio.NewReader(os.Stdin)
	text_serial, _ := reader_utport.ReadString('\n')
	words_serial := strings.Fields(text_serial)
	fmt.Println(words_serial)
	//fmt.Printf("%v %v", color.GreenString("Info:"), "an important message.")
	//fmt.Fprintf(color.Output, "Windows support: %s", color.GreenString("PASS"))

	toEncoder = make(chan []byte, 100)
	toDecoder = make(chan []byte, 100)
	toSender  = make(chan []byte, 100)

	next_input = make(chan bool,5)

	connect_chan = make(chan bool, 1)

	current_file = 0

	go Decoder()
	
	/* define option */
	options := serial.OpenOptions{
		PortName: words_serial[0],
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
					connect_chan <- true
					color.Red("connected")
					next_input <- true
				} else {
					next_input <- true
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

	
	reader := bufio.NewReader(os.Stdin)
	var words []string 
	next_input <- true
	for {
		<-next_input		
		//fmt.Println("Type: connect, file x (x= file nummber)")
		fmt.Fprintf(color.Output, "%s", color.YellowString("Type: connect, file x (x= file nummber) : "))
		text, _ := reader.ReadString('\n')
    	//fmt.Println(scanner.Text())
    	select{
    	case <-signalChan:
    		fmt.Println("Keyboard Interrupt")
    		time.Sleep(3000 * time.Millisecond)
    		return
    	default:
    		words = strings.Fields(text)
			//fmt.Println(words, len(words)) 

			if words[0] == "connect" {
				buf := make([]byte,2)
				buf[0] = ping
				buf[1] = 10
				port.Write(buf)

				count := 0
				loop:
				for{
					select{
					case <- connect_chan:
						break loop
					default:
						if count >= 5 {
							fmt.Println("connection timeout, please check UT and Telemetry")
							break loop
						}
					}
					count++
					time.Sleep(50*time.Millisecond)
				}
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
	}
	/*
	fmt.Println("Hello, 世界")
	for{
		time.Sleep(10000 * time.Millisecond)
	}
	*/
}