package nfs

type Client struct {
	serverIP  string
	nfsPath   string
	localPath string
}

func NewClient(serverIP, nfsPath, localPath string) *Client {
	return &Client{
		serverIP:  serverIP,
		nfsPath:   nfsPath,
		localPath: localPath,
	}
}

func (c *Client) Mount() {

}

func (c *Client) Unmount() {
}

func (c *Client) hostsAllow() {
}

func (c *Client) hostsDeny() {
}
