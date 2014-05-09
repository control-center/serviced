package nfs

type Server struct {
	localNetwork  string
	exportOptions string
	clients       map[string]struct{}
}

func NewServer(nativePath, exported string) *Server {
	return &Server{
		exportOptions: "rw,nohide,insecure,no_subtree_check,async",
	}
}

func (c *Server) SetClient(client string) {
	c.clients[client] = struct{}{}
}

func (c *Server) RemoveClient(client string) {
	delete(c.clients, client)
}

func (c *Server) Sync() {
	c.hostsDeny()
	c.hostsAllow()
}

func (c *Server) hostsDeny() {
	// write to hosts.deny
	// rpcbind mountd nfsd statd lockd rquotad : ALL
}

func (c *Server) hostsAllow() {
	// write to hosts.allow
	// ensure 127.0.0.1 can access
	// rpcbind mountd nfsd statd lockd rquotad : list of IP addresses
}
