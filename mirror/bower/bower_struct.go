// Copyright © 2016 Thomas Rabaix <thomas.rabaix@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package bower

type Package struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type Packages []*Package
