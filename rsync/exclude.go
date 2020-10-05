package rsync

import (
	"path/filepath"
	"strings"
)

// Reference: rsync 2.6.0
// --exclude & --exclude-from

/* These default ignored items come from the CVS manual.
 * "RCS SCCS CVS CVS.adm RCSLOG cvslog.* tags TAGS"
 * " .make.state .nse_depinfo *~ #* .#* ,* _$* *$"
 * " *.old *.bak *.BAK *.orig *.rej .del-*"
 * " *.a *.olb *.o *.obj *.so *.exe"
 * " *.Z *.elc *.ln core"
 The rest we added to suit ourself.
 * " .svn/ .bzr/"
 */

// Filter List
type Exclusion struct {
	patterns []string
	root string
}

func (e *Exclusion) Match(name string) (matched bool, err error) {
	matched = false
	for _, p := range e.patterns {
		if strings.HasPrefix(name, p) && name[len(p)] == '/' {
			return true, nil
		}
		if matched, err = filepath.Match(p, name); matched || err != nil {break}
	}
	return
}

func (e *Exclusion) Add(pattern string) {
	// Check the root, if not empty, join them
	e.patterns = append(e.patterns, filepath.Join(e.root, pattern))
}

// This is only called by the client
func (e *Exclusion) SendExlusion(conn Conn) error {
	// If list_only && !recurse, add '/*/*'


	// For each item, send its length first
	for _, p := range e.patterns {
		plen := int32(len(p))
		// TODO: If a dir, append a '/' at the end
		if err := conn.WriteInt(plen); err != nil {
			return err
		}
		if _, err := conn.Write([]byte(p)); err != nil {
			return err
		}
	}

	if err := conn.WriteInt(EXCLUSION_END); err != nil {
		return err
	}
	return nil
}

