package main

import "fmt"
import "encoding/hex"
import "time"
import "github.com/jacobsa/go-serial/serial"

var PCtoUT chan []byte
var UTtoPC chan []byte

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
	/********************/

	fmt.Println("telemetry connected")
	/* loop */	
	for{
		// sending
		go func() {
			for{
        	port.Write(<-UTtoPC)
        	}
    	}()

    	//reading
		var data []byte
		buf := make([]byte,1)
		n, err := port.Read(buf)
		if err != nil {
			fmt.Println("Error reading from TELE: ", err)
		} else{
			buf = buf[:n]
			//fmt.Println("PC Rx: ", hex.EncodeToString(buf))
		}

		if hex.EncodeToString(buf) == "74" {
			buf2 := make([]byte,1)
			n, err := port.Read(buf2)
			if err != nil {
				fmt.Println("Error reading from TELE: ", err)
			} else{
				buf2 = buf2[:n]
				//fmt.Println("PC Rx: ", hex.EncodeToString(buf2))
				data = append(data[:],buf[:]...)
			}

			if hex.EncodeToString(buf2) == "0a" {
				data = append(data[:],buf2[:]...)
			}

			fmt.Println("toUT",data)
			PCtoUT <- data
		} else if hex.EncodeToString(buf) == "aa" {
			data = append(data[:],buf[:]...)
			for{
				n, err := port.Read(buf)
				if err != nil {
					fmt.Println("Error reading from TELE: ", err)
				} else{
					buf = buf[:n]
					//fmt.Println("PC Rx: ", hex.EncodeToString(buf))
				}

				if hex.EncodeToString(buf) == "bb" {
					data = append(data[:],buf[:]...)
					break
				} else {
					data = append(data[:],buf[:]...)
				}
			}

			fmt.Println("toUT",data)
			PCtoUT <- data
		}		
	}

	buf := make([]byte, 2)
	n, err := port.Read(buf)
	if err != nil {
		fmt.Println("Error reading from TELE: ", err)
	} else{
		buf = buf[:n]
		//fmt.Println("Rx: ", hex.EncodeToString(buf))
	}
}

func UTside() {
	fmt.Println("start UTside")

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
	/********************/

	fmt.Println("UT connected")
	for{
		// sending
		go func() {
			for{
        	port.Write(<-PCtoUT)
        	}
    	}()

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
				}
			}
			fmt.Println("toPC: ",data,"\n")
			UTtoPC <- data
		} else if hex.EncodeToString(buf) == "aa" {
			data = append(data[:],buf[:]...)
			for{
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

			fmt.Println("toPC: ",data,"\n")
			UTtoPC <- data
		}
	}
}

func main() {
	PCtoUT = make(chan []byte, 100)
	UTtoPC = make(chan []byte, 100)
	go PCside()
	go UTside()

	fmt.Println("Hello, 世界")
	for{
		time.Sleep(1000 * time.Millisecond)
	}
}