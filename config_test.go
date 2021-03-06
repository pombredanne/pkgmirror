// Copyright © 2016 Thomas Rabaix <thomas.rabaix@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package pkgmirror

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
)

func Test_Config(t *testing.T) {
	c := &Config{}

	confStr := `
DataDir = "/var/lib/pkgmirror"
PublicServer = "https://mirror.example.com"
InternalServer = "localhost:8000"

[Composer]
    [Composer.packagist]
    Server = "https://packagist.org"

    [Composer.satis]
    Server = "https://satis.internal.org"

[Npm]
    [Npm.npm]
    Server = "https://registry.npmjs.org"

[Git]
    [Git.github]
    Server = "github.com"
    Clone = "git@gitbub.com:"

`

	_, err := toml.Decode(confStr, c)

	assert.NoError(t, err)
	assert.Equal(t, "/var/lib/pkgmirror", c.DataDir)
	assert.Equal(t, 2, len(c.Composer))
	assert.Equal(t, "https://satis.internal.org", c.Composer["satis"].Server)
	assert.Equal(t, "https://packagist.org", c.Composer["packagist"].Server)

	assert.Equal(t, 1, len(c.Npm))
	assert.Equal(t, "https://registry.npmjs.org", c.Npm["npm"].Server)

	assert.Equal(t, 1, len(c.Git))
	assert.Equal(t, "github.com", c.Git["github"].Server)
	assert.Equal(t, "git@gitbub.com:", c.Git["github"].Clone)
}
