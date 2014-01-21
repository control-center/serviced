package isvcs

var zookeeper ContainerDescription

func init() {
	zookeeper = ContainerDescription{
		Name:    "zookeeper",
		Repo:    "zctrl/isvcs",
		Tag:     "v1",
		Command: "/opt/zookeeper-3.4.5/bin/zkServer.sh start-foreground",
		Ports:   []int{2181, 12181},
		Volumes: []string{"/tmp"},
	}

}
