package main

import (
"os"
"strings"
"strconv"
"encoding/hex"
"log"
"fmt"
//"github.com/tidwall/buntdb"
"time"
"github.com/jacobsa/go-serial/serial"
"bufio"
)

var toDecoder chan []byte
var toUTwrite chan []string

func xor(b []byte) []byte {	
	b = append(b,0)
	LEN := len(b)
	for i := 0; i < LEN-1; i++ {
		b[LEN-1] = b[i]^b[LEN-1]
 	}
	return b
}

func UTside() {
	/* define option */
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

	fmt.Println("telemetry connected")
	defer fmt.Println("disconnected and end routine")

	//anony function
	ReadOneByte := func() ([]byte, error){
		buf := make([]byte,1)
		_, err := port.Read(buf)
		if err != nil {
			fmt.Println("Error reading form UT")
			return []byte{0}, err
		}
		return buf , nil
	}

	//read function
	go func(){
		loop:
		for{
			var data []byte
			buf, err := ReadOneByte()
			if err != nil {
				log.Fatal(err)
			}
			switch hex.EncodeToString(buf) {
				case "32": // ping
					start := time.Now()
					buf, err := ReadOneByte()
					if err != nil {
						log.Fatal(err)
					}
					buf2, err := ReadOneByte()
					if err != nil {
						log.Fatal(err)
					}
					if hex.EncodeToString(buf) == "32" && hex.EncodeToString(buf2) =="21"{
						fmt.Println("connected")
						goto loop
					}

					if time.Since(start) >= time.Second {
							fmt.Println("Time out")
							goto loop
						}
				case "aa": //read package
					data = append(data[:],buf[:]...)
					start := time.Now()
					for{
						buf, err := ReadOneByte()
						if err != nil {
							log.Fatal(err)
						}
						if hex.EncodeToString(buf) == "bb"{
							data = append(data[:],buf[:]...)
							break
						} else {
							data = append(data[:],buf[:]...)
						}

						if time.Since(start) >= time.Second {
							fmt.Println("Time out")
							goto loop
						}
					}
			}
			toDecoder <- data			
		}	
	}()

	//write function
	go func () {
		for{
			var buf []byte
			select{
			case word := <-toUTwrite :
				switch word[0] {
				case "connect":
					buf := make([]byte,2)
					buf[0] = 116
					buf[1] = 10
					break 
				case "readfile":
					if len(word) != 2 {
						fmt.Println("Invalid input")
						continue
					}

					file, err := strconv.Atoi(word[1])
					if err != nil {
						fmt.Println("Invalid file number")
						continue
					}
					
					if file >= 0 && file <= 99 {
						buf := []byte{67,3,0,49,byte(file),0}
						buf = xor(buf)
						header := []byte{170}
						buf = append(header,buf[:]...)
						buf = append(buf,[]byte{187}...)
					}
				case "readfiledata":
					if len(word) != 2 {
						fmt.Println("Invalid input")
						continue
					}

					file, err := strconv.Atoi(word[1])
					if err != nil {
						fmt.Println("Invalid file number")
						continue
					}

					if file >= 0 && file <= 99 {
						buf := []byte{81,2,0,byte(file),0}
						buf = xor(buf)
						header := []byte{170}
						buf = append(header,buf[:]...)
						buf = append(buf,[]byte{187}...)
					}					
				}
			}
			port.Write(buf)
		}
	}()
}

func PCside() {
	/* define option */
	options := serial.OpenOptions{
		PortName: "/dev/ttyUSB1",
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

	fmt.Println("telemetry connected")
	defer fmt.Println("disconnected and end routine")
}

func main() {
	fmt.Println("Hello, 世界")

	// Open the data.db file. It will be created if it doesn't exist.
	/*
	db, err := buntdb.Open("data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	*/

	go UTside()
	//go PCside()

	for{
		select{
			case msg := <- toDecoder :
				fmt.Println(msg)
		}
	}

	// cmd input
	toUTwrite := make(chan []string,10)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("Enter command: ")
		scanner.Scan()
		words := strings.Fields(scanner.Text())
		toUTwrite <- words
	}
}