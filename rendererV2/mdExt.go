package md2jsV2

import (
	"fmt"
	"bytes"
)


func GetYaml (buf []byte) (mdBuf []byte, yamlBuf []byte, err error) {

	idx := bytes.Index(buf, []byte("---\n"))
	if idx != 0 {return buf, yamlBuf, nil}

	idx = bytes.Index(buf[4:], []byte("---\n"))
	if idx ==-1 {return mdBuf, yamlBuf, fmt.Errorf("no end of yaml section!")}

//	ridx := bytes.EqualByte(buf[7:], byte('\n'))
//	if ridx > 8 {return mdBuf, yamlBuf, fmt.Errorf("no end of eol!")}

	return buf[idx+5:], buf[:idx+5], nil
}

func GetSummary(buf []byte) (sumBuf []byte, err error) {

	idx := bytes.Index(buf, []byte("# summary"))
	if idx == -1 {return sumBuf, nil}

	hdidx := bytes.Index(buf[idx+9:], []byte("\n# ")) 
	if hdidx == -1 {return sumBuf, fmt.Errorf("no summary end")}

	return buf[idx:idx+9+hdidx], nil
}

func PrintSum(buf []byte) {

	fmt.Println("****** Summary ******")
	fmt.Printf("%s\n",string(buf))
	fmt.Println("**** End Summary ****")
}
