package isvcs

var ZookeeperContainer ISvc

func init() {
	ZookeeperContainer = ISvc{
		Name:       "zookeeper",
		Repository: "zctrl/zk",
		Tag:        "v1",
		Ports:      []int{2181, 12181},
	}
}
