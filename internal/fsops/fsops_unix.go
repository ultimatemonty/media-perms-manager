package fsops

import (
    "fmt"
    "os"
    "os/user"
    "strconv"
)

// ResolveUserGroup resolves username and group (either names or numeric strings)
// and returns uid,gid ints.
func ResolveUserGroup(userStr, groupStr string) (int, int, error) {
    var uid, gid int
    if userStr == "" {
        return 0, 0, fmt.Errorf("empty user")
    }
    // try numeric
    if u, err := strconv.Atoi(userStr); err == nil {
        uid = u
    } else {
        u, err := user.Lookup(userStr)
        if err != nil {
            return 0, 0, err
        }
        uu, err := strconv.Atoi(u.Uid)
        if err != nil {
            return 0, 0, err
        }
        uid = uu
    }

    if groupStr == "" {
        return uid, -1, nil
    }
    if g, err := strconv.Atoi(groupStr); err == nil {
        gid = g
    } else {
        grp, err := user.LookupGroup(groupStr)
        if err != nil {
            return 0, 0, err
        }
        gg, err := strconv.Atoi(grp.Gid)
        if err != nil {
            return 0, 0, err
        }
        gid = gg
    }
    return uid, gid, nil
}

func Chown(path string, uid, gid int, dryRun bool) error {
    if dryRun {
        return nil
    }
    return os.Chown(path, uid, gid)
}

func Chmod(path string, mode os.FileMode, dryRun bool) error {
    if dryRun {
        return nil
    }
    return os.Chmod(path, mode)
}
