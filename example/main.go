// +ignore

package main

import (
	"log"

	"github.com/globalcyberalliance/ftp-go/driver/file"
)

func main() {
	driver, err := file.NewDriver("./")
	if err != nil {
		log.Fatal(err)
	}

	s, err := ftp.NewServer(&ftp.Options{
		Driver: driver,
		Auth: &ftp.SimpleAuth{
			Name:     "admin",
			Password: "admin",
		},
		Perm:      ftp.NewSimplePerm("root", "root"),
		RateLimit: 1000000, // 1MB/s limit
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
