// SPDX-FileCopyrightText: 2024 M. Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"log"
	"os"

	"git.sr.ht/~shulhan/pakakeh.go/lib/memfs"
)

func main() {
	embedDatabase()
	embedWui()
}

func embedDatabase() {
	var memfsOpts = memfs.Options{
		Embed: memfs.EmbedOptions{
			PackageName: `haminer`,
			VarName:     `memfsDatabase`,
			GoFileName:  `memfs_database.go`,
			CommentHeader: `// SPDX-FileCopyrightText: 2024 M. Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

`,
		},
		Root: `_database`,
		Includes: []string{
			`.*\.sql$`,
		},
	}

	var (
		mfs *memfs.MemFS
		err error
	)

	mfs, err = memfs.New(&memfsOpts)
	if err != nil {
		log.Fatal(os.Args[0], err)
	}

	err = mfs.GoEmbed()
	if err != nil {
		log.Fatal(os.Args[0], err)
	}
}

func embedWui() {
	var memfsOpts = memfs.Options{
		Embed: memfs.EmbedOptions{
			PackageName: `haminer`,
			VarName:     `memfsWUI`,
			GoFileName:  `memfs_wui.go`,
			CommentHeader: `// SPDX-FileCopyrightText: 2024 M. Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

`,
		},
		Root: `_wui`,
		Includes: []string{
			`.*\.(html|js)$`,
		},
	}

	var (
		mfs *memfs.MemFS
		err error
	)

	mfs, err = memfs.New(&memfsOpts)
	if err != nil {
		log.Fatal(os.Args[0], err)
	}

	err = mfs.GoEmbed()
	if err != nil {
		log.Fatal(os.Args[0], err)
	}
}
