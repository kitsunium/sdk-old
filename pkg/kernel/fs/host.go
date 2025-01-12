package fs

import (
	"os/user"
	"strconv"
)

var (
	uid int = 0
	gid int = 0
)

func init() {
	u, err := user.Current()
	if err != nil {
		uid, gid = 0, 0
		return
	}

	uid, err = strconv.Atoi(u.Uid)
	if err != nil {
		uid = 0
	}

	gid, err = strconv.Atoi(u.Gid)
	if err != nil {
		gid = 0
	}
}
