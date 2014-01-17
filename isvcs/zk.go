package isvcs

var ZookeeperContainer ISvc

func init() {
	ZookeeperContainer = NewISvc(
		"zookeeper",
		"zctrl/isvcs",
		"v1",
		"/opt/zookeeper-3.4.5/bin/zkServer.sh start-foreground",
		[]int{2181, 12181},
		[]string{"/tmp"},
	)
}
