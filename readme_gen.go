// +build ignore

package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

func main() {
	err := vfsgen.Generate(http.Dir("html-docs"), vfsgen.Options{
		PackageName:  "main",
		VariableName: "Readme",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
