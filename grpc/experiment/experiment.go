package trial

import (
	"fmt"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

func PrintSettingsFrame(frame *http2.SettingsFrame) {
	fmt.Println("This is a SETTINGS frame")
	//fmt.Println(frame.Setting(2).String())
}
func PrintDataFrame(frame *http2.DataFrame) {
	fmt.Println(string(frame.Data()))

}

func PrintFrame(frame http2.Frame) {
	switch frame.(type) {
	case *http2.SettingsFrame:
		PrintSettingsFrame(frame.(*http2.SettingsFrame))
	case *http2.DataFrame:
		PrintDataFrame(frame.(*http2.DataFrame))
	}
}
func HeaderFields(framer *http2.Framer) int {
	frame, err := framer.ReadFrame()
	if err != nil {
		return 0
		//panic(err)
	}

	fmt.Printf("fh type: %s\n", frame.Header().Type)
	fmt.Printf("fh flag: %d\n", frame.Header().Flags)
	fmt.Printf("fh length: %d\n", frame.Header().Length)
	fmt.Printf("fh streamid: %d\n", frame.Header().StreamID)

	PrintFrame(frame)

	headersFrame, ok := frame.(*http2.HeadersFrame)
	if !ok {
		fmt.Println("not a valid header frame")
		return 0
		//panic("not a valid header frame")
	}
	decoder := hpack.NewDecoder(2048, nil)
	hf, _ := decoder.DecodeFull(headersFrame.HeaderBlockFragment())
	for _, h := range hf {
		fmt.Printf("%s\n", h.Name+":"+h.Value)
	}
	return 1
}

//func main() {
//	data := []byte{0, 0, 4, 8, 0, 0, 0, 0, 0, 0, 0, 0, 16, 0, 0, 8, 6, 0, 0, 0, 0, 0, 2, 4, 16, 16, 9, 14, 7, 7, 0, 0, 2, 1, 4, 0, 0, 0, 3, 136, 192, 0, 0, 22, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 17, 10, 15, 72, 101, 108, 108, 111, 32, 103, 82, 80, 67, 45, 99, 97, 108, 108, 0, 0, 2, 1, 5, 0, 0, 0, 3, 191, 190}
//	//data = []byte{0, 0, 14, 1, 4, 0, 0, 0, 1, 136, 95, 139, 29, 117, 208, 98, 13, 38, 61, 76, 77, 101, 100, 0, 0, 22, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 17, 10, 15, 72, 101, 108, 108, 111, 32, 103, 82, 80, 67, 45, 99, 97, 108, 108, 0, 0, 24, 1, 5, 0, 0, 0, 1, 64, 136, 154, 202, 200, 178, 18, 52, 218, 143, 1, 48, 64, 137, 154, 202, 200, 181, 37, 66, 7, 49, 127, 0}
//	//data = []byte{0, 0, 6, 4, 0, 0, 0, 0, 0, 0, 5, 0, 0, 64, 0}
//
//	//data = []byte{0, 0, 0, 4, 1, 0, 0, 0, 0} // SETTINGS frame.
//	//data = []byte{0, 0, 6, 4, 0, 0, 0, 0, 0, 0, 5, 0, 0, 64, 0}
//	//data = []byte{0, 0, 0, 4, 1, 0, 0, 0, 0}
//	//data = []byte{0, 0, 4, 8, 0, 0, 0, 0, 0, 0, 0, 0, 16, 0, 0, 8, 6, 0, 0, 0, 0, 0, 2, 4, 16, 16, 9, 14, 7, 7}
//	data = []byte{0, 0, 14, 1, 4, 0, 0, 0, 1, 136, 95, 139, 29, 117, 208, 98, 13, 38, 61, 76, 77, 101, 100, 0, 0, 22, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 17, 10, 15, 72, 101, 108, 108, 111, 32, 103, 82, 80, 67, 45, 99, 97, 108, 108}
//	//data = []byte{0, 0, 24, 1, 5, 0, 0, 0, 1, 64, 136, 154, 202, 200, 178, 18, 52, 218, 143, 1, 48, 64, 137, 154, 202, 200, 181, 37, 66, 7, 49, 127, 0}
//	buf := bytes.NewBuffer(data)
//	framer := http2.NewFramer(buf, buf)
//
//	for {
//		headerFields(framer)
//	}
//}
