package main

import (
	"log"
	"os"

	"git.sr.ht/~shulhan/pakakeh.go/lib/memfs"
)

func main() {
	embedDatabase()
}

func embedDatabase() {
	var memfsOpts = memfs.Options{
		Embed: memfs.EmbedOptions{
			PackageName: `haminer`,
			VarName:     `memfsDatabase`,
			GoFileName:  `memfs_database.go`,
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
