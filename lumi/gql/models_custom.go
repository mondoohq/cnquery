package gql

type Group struct {
	Gid     int64  `json:"gid"`
	Name    string `json:"name"`
	Members []User `json:"members"`
}

type Process struct {
	Pid        int64  `json:"pid"`
	State      string `json:"state"`
	Executable string `json:"executable"`
	Command    string `json:"command"`
	Uid        int64  `json:"uid"`
}

type User struct {
	Uid         int64  `json:"uid"`
	Gid         int64  `json:"gid"`
	Username    string `json:"username"`
	Home        string `json:"home"`
	Shell       string `json:"shell"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}
