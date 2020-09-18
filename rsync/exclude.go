package rsync

import (
	"path/filepath"
	"strings"
)

// Reference: rsync 2.6.9
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
type Exclusion []string

func (e *Exclusion) Match(name string) (matched bool, err error) {
	matched = false
	for _, p := range *e {
		if strings.HasPrefix(name, p) && name[len(p)] == '/' {
			return true, nil
		}
		if matched, err = filepath.Match(p, name); matched || err != nil {break}
	}
	return
}

func (e *Exclusion) Add(pattern string) {
	for _, p := range *e {
		if strings.HasPrefix(p, pattern) {
			p = pattern
			return
		}
		if strings.HasPrefix(pattern, p) {
			return
		}
	}
	*e = append(*e, pattern)
}

// This is only called by the client
func sendExlusion(conn Conn) error {
	// send rule

	// For each item, send it len as


	// If local or (sender && receiver_wants_list), client won't send 0
	return nil
}

