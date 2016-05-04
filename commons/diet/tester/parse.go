package main

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/control-center/serviced/commons/diet"
)

var test = strings.NewReader(`
<superblock uuid="" time="1" transaction="2" data_block_size="128" nr_data_blocks="0">
  <device dev_id="1" mapped_blocks="4664" transaction="0" creation_time="0" snap_time="1">
    <range_mapping origin_begin="0" data_begin="0" length="555" time="0"/>
    <single_mapping origin_block="2048" data_block="555" time="0"/>
    <single_mapping origin_block="6144" data_block="556" time="0"/>
    <single_mapping origin_block="10240" data_block="557" time="0"/>
    <single_mapping origin_block="14336" data_block="558" time="0"/>
    <single_mapping origin_block="18432" data_block="559" time="0"/>
    <single_mapping origin_block="32768" data_block="560" time="0"/>
    <range_mapping origin_begin="32770" data_begin="561" length="512" time="0"/>
    <single_mapping origin_block="51200" data_block="1073" time="0"/>
    <single_mapping origin_block="55296" data_block="1074" time="0"/>
    <single_mapping origin_block="65536" data_block="1075" time="0"/>
    <range_mapping origin_begin="65538" data_begin="1076" length="512" time="0"/>
    <range_mapping origin_begin="67584" data_begin="1588" length="2048" time="0"/>
    <single_mapping origin_block="98304" data_block="3636" time="0"/>
    <range_mapping origin_begin="98306" data_begin="3637" length="512" time="0"/>
    <single_mapping origin_block="100352" data_block="4149" time="0"/>
    <single_mapping origin_block="131072" data_block="4150" time="0"/>
    <range_mapping origin_begin="131074" data_begin="4151" length="512" time="0"/>
    <single_mapping origin_block="163839" data_block="4663" time="0"/>
  </device>
  <device dev_id="2" mapped_blocks="8760" transaction="1" creation_time="1" snap_time="1">
    <single_mapping origin_block="0" data_block="4664" time="1"/>
    <range_mapping origin_begin="1" data_begin="1" length="554" time="0"/>
    <single_mapping origin_block="2048" data_block="555" time="0"/>
    <range_mapping origin_begin="2176" data_begin="4666" length="3968" time="1"/>
    <single_mapping origin_block="6144" data_block="556" time="0"/>
    <range_mapping origin_begin="6272" data_begin="8634" length="128" time="1"/>
    <single_mapping origin_block="10240" data_block="557" time="0"/>
    <single_mapping origin_block="14336" data_block="558" time="0"/>
    <single_mapping origin_block="18432" data_block="559" time="0"/>
    <single_mapping origin_block="32768" data_block="560" time="0"/>
    <range_mapping origin_begin="32770" data_begin="561" length="512" time="0"/>
    <single_mapping origin_block="51200" data_block="1073" time="0"/>
    <single_mapping origin_block="55296" data_block="1074" time="0"/>
    <single_mapping origin_block="65536" data_block="1075" time="0"/>
    <range_mapping origin_begin="65538" data_begin="1076" length="512" time="0"/>
    <single_mapping origin_block="67584" data_block="4665" time="1"/>
    <range_mapping origin_begin="67585" data_begin="1589" length="2047" time="0"/>
    <single_mapping origin_block="98304" data_block="3636" time="0"/>
    <range_mapping origin_begin="98306" data_begin="3637" length="512" time="0"/>
    <single_mapping origin_block="100352" data_block="4149" time="0"/>
    <single_mapping origin_block="131072" data_block="4150" time="0"/>
    <range_mapping origin_begin="131074" data_begin="4151" length="512" time="0"/>
    <single_mapping origin_block="163839" data_block="4663" time="0"/>
  </device>
</superblock>`)

func main() {
	decoder := xml.NewDecoder(test)

	devices := make(map[int]*diet.Diet)
	current := 0

	for {
		t, _ := decoder.Token()
		if t == nil {
			for id, d := range devices {
				d.Balance()
				fmt.Printf("DIET %d\n", id)
				fmt.Printf("TOTAL %d BLOCKS\n", d.Total())
				cmp := devices[1]
				fmt.Printf("UNIQUE %d BLOCKS\n", d.Total()-d.IntersectionAll(cmp))
			}
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "device" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "dev_id" {
						current, _ = strconv.Atoi(attr.Value)
						break
					}
				}
				continue
			}
			var (
				d  *diet.Diet
				ok bool
			)
			if se.Name.Local == "range_mapping" {
				var m RangeMapping
				decoder.DecodeElement(&m, &se)
				if d, ok = devices[current]; !ok {
					d = diet.NewDiet()
					devices[current] = d
				}
				d.Insert(uint64(m.DataBegin), uint64(m.DataBegin+m.Length-1))
			}
			if se.Name.Local == "single_mapping" {
				var m SingleMapping
				decoder.DecodeElement(&m, &se)
				if d, ok = devices[current]; !ok {
					d = diet.NewDiet()
					devices[current] = d
				}
				d.Insert(uint64(m.DataBlock), uint64(m.DataBlock))
			}

		}
	}
}
