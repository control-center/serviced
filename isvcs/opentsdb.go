package isvcs

var OpenTsdbContainer ISvc

func init() {
	OpenTsdbContainer = ISvc{
		Name:       "opentsdb",
		Repository: "zctrl/opentsdb",
		Tag:        "v1",
		Ports:      []int{4242},
	}
}
